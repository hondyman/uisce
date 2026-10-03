// Package tenantdb resolves a tenant datasource to a live Postgres pool. It is the only
// package that opens connections to a tenant's own database (ADR-030).
//
// Every failure returns an error. There is no fallback database, no development default and
// no reuse of a pool built for another tenant: a missing, ambiguous or unauthorized tenant,
// an unbound or inactive datasource, a database the connection does not land on, or a user
// other than the one the registry names, all fail closed.
package tenantdb

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrNoTenant: the caller's context carries no verified tenant.
	ErrNoTenant = errors.New("tenantdb: no tenant in context")
	// ErrTenantMismatch: the datasource belongs to a different tenant than the caller's.
	ErrTenantMismatch = errors.New("tenantdb: datasource not owned by the caller's tenant")
	// ErrUnbound: the datasource has no binding, or it is not active.
	ErrUnbound = errors.New("tenantdb: datasource has no active binding")
	// ErrDatabaseMismatch: a connection landed on a database other than the registered one.
	ErrDatabaseMismatch = errors.New("tenantdb: connection is not on the expected database")
	// ErrUserMismatch: a connection is authenticated as a user other than the registered one.
	ErrUserMismatch = errors.New("tenantdb: connection is not the expected user")
	// ErrIncomplete: the datasource row lacks a host, port, database or credential.
	ErrIncomplete = errors.New("tenantdb: datasource connection details are incomplete")
)

// Datasource is the registry's description of one tenant database. No credential is in it.
type Datasource struct {
	ID       string
	TenantID string // owner, read from the registry, never from the caller
	Host     string
	Port     int
	Database string

	config []byte // the row's raw config, for the Registry that hydrates its credentials
}

// Binding is the live registry binding for a datasource.
type Binding struct {
	Version   int
	Lifecycle string // provisioning | active | suspended | offboarding | offboarded
}

// Registry is implemented over alpha (see AlphaRegistry).
type Registry interface {
	// ResolveDatasource returns the datasource and its owning tenant. An unknown, inactive
	// or ambiguous datasource is an error.
	ResolveDatasource(ctx context.Context, datasourceID string) (Datasource, error)
	// LoadBinding returns the datasource's binding; no binding is ErrUnbound.
	LoadBinding(ctx context.Context, datasourceID string) (Binding, error)
	// Credentials returns the database user and password for the datasource.
	Credentials(ctx context.Context, ds Datasource) (user, password string, err error)
}

// CallerTenant returns the verified tenant of the request. An absent tenant is an error.
type CallerTenant func(ctx context.Context) (string, error)

// Config configures a Router. Every limit is required: a router with unbounded pools is a
// connection leak waiting for the tenant count to grow.
type Config struct {
	Registry     Registry
	CallerTenant CallerTenant

	MaxPools        int           // bounded: the least recently used pool is closed past this
	MaxConnsPerPool int32         // connections per tenant database
	IdleTTL         time.Duration // a pool unused for this long is closed
	DialTimeout     time.Duration
}

type cacheKey struct {
	datasourceID string
	tenantID     string
	version      int // a rebinding (a cutover) builds a new pool; the old one is evicted
}

type entry struct {
	pool    *pgxpool.Pool
	lastUse time.Time
}

// Router hands out pools. Resolve once per request or job and pass the *Pool down.
type Router struct {
	cfg   Config
	now   func() time.Time
	mu    sync.Mutex
	pools map[cacheKey]*entry
}

// New validates the config and returns a Router.
func New(cfg Config) (*Router, error) {
	switch {
	case cfg.Registry == nil:
		return nil, errors.New("tenantdb: Registry is required")
	case cfg.CallerTenant == nil:
		return nil, errors.New("tenantdb: CallerTenant is required")
	case cfg.MaxPools <= 0 || cfg.MaxConnsPerPool <= 0:
		return nil, errors.New("tenantdb: MaxPools and MaxConnsPerPool must be positive")
	case cfg.IdleTTL <= 0 || cfg.DialTimeout <= 0:
		return nil, errors.New("tenantdb: IdleTTL and DialTimeout must be positive")
	}
	return &Router{cfg: cfg, now: time.Now, pools: map[cacheKey]*entry{}}, nil
}

// Resolve maps a datasource to a pool for the caller's tenant.
func (r *Router) Resolve(ctx context.Context, datasourceID string) (*Pool, error) {
	actor, err := r.cfg.CallerTenant(ctx)
	if err != nil || actor == "" {
		return nil, ErrNoTenant
	}
	ds, err := r.cfg.Registry.ResolveDatasource(ctx, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("tenantdb: resolve datasource: %w", err)
	}
	if ds.TenantID == "" || ds.TenantID != actor {
		return nil, ErrTenantMismatch
	}
	if ds.Host == "" || ds.Port <= 0 || ds.Database == "" {
		return nil, ErrIncomplete
	}
	b, err := r.cfg.Registry.LoadBinding(ctx, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("tenantdb: load binding: %w", err)
	}
	if b.Lifecycle != "active" {
		return nil, fmt.Errorf("%w: lifecycle=%q", ErrUnbound, b.Lifecycle)
	}
	return r.get(ctx, ds, b)
}

func (r *Router) get(ctx context.Context, ds Datasource, b Binding) (*Pool, error) {
	key := cacheKey{datasourceID: ds.ID, tenantID: ds.TenantID, version: b.Version}

	r.mu.Lock()
	if e, ok := r.pools[key]; ok {
		e.lastUse = r.now()
		p := e.pool
		r.mu.Unlock()
		return &Pool{ds: ds, p: p}, nil
	}
	r.mu.Unlock()

	user, password, err := r.cfg.Registry.Credentials(ctx, ds)
	if err != nil {
		return nil, fmt.Errorf("tenantdb: credentials: %w", err)
	}
	if user == "" || password == "" {
		return nil, ErrIncomplete
	}

	pc, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, fmt.Errorf("tenantdb: pool config: %w", err)
	}
	pc.ConnConfig.Host = ds.Host
	pc.ConnConfig.Port = uint16(ds.Port)
	pc.ConnConfig.Database = ds.Database
	pc.ConnConfig.User = user
	pc.ConnConfig.Password = password
	pc.ConnConfig.ConnectTimeout = r.cfg.DialTimeout
	pc.MaxConns = r.cfg.MaxConnsPerPool
	pc.MinConns = 0
	pc.MaxConnIdleTime = r.cfg.IdleTTL

	// Identity is asserted when a connection is made and again on every checkout, so a
	// connection that was re-pointed or re-authenticated underneath the pool cannot serve a query.
	pc.AfterConnect = func(ctx context.Context, c *pgx.Conn) error { return assertIdentity(ctx, c, ds.Database, user) }
	pc.BeforeAcquire = func(ctx context.Context, c *pgx.Conn) bool {
		return assertIdentity(ctx, c, ds.Database, user) == nil
	}
	pc.AfterRelease = func(c *pgx.Conn) bool {
		// A connection with leftover session state (a SET, a role switch) is never reused.
		ctx, cancel := context.WithTimeout(context.Background(), r.cfg.DialTimeout)
		defer cancel()
		_, err := c.Exec(ctx, "RESET ALL")
		return err == nil
	}

	p, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, fmt.Errorf("tenantdb: open pool: %w", err)
	}
	// NewWithConfig dials lazily. Prove the connection is the right database and user NOW,
	// so a wrong target is an error at Resolve, not a surprise on the first query.
	if err := p.Ping(ctx); err != nil {
		p.Close()
		return nil, fmt.Errorf("tenantdb: connect: %w", err)
	}

	r.mu.Lock()
	if e, ok := r.pools[key]; ok { // lost a race; keep the existing pool
		r.mu.Unlock()
		p.Close()
		return &Pool{ds: ds, p: e.pool}, nil
	}
	r.pools[key] = &entry{pool: p, lastUse: r.now()}
	victims := r.evictLocked(key)
	r.mu.Unlock()

	// Close outside the lock: Close waits for in-flight connections to be released.
	for _, v := range victims {
		v.Close()
	}
	return &Pool{ds: ds, p: p}, nil
}

// evictLocked removes pools idle past IdleTTL and, if still over MaxPools, the least recently
// used ones, never the pool just built. The caller closes the returned pools.
func (r *Router) evictLocked(keep cacheKey) []*pgxpool.Pool {
	var out []*pgxpool.Pool
	now := r.now()
	for k, e := range r.pools {
		if k != keep && now.Sub(e.lastUse) > r.cfg.IdleTTL {
			out = append(out, e.pool)
			delete(r.pools, k)
		}
	}
	if len(r.pools) > r.cfg.MaxPools {
		keys := make([]cacheKey, 0, len(r.pools))
		for k := range r.pools {
			if k != keep {
				keys = append(keys, k)
			}
		}
		sort.Slice(keys, func(i, j int) bool { return r.pools[keys[i]].lastUse.Before(r.pools[keys[j]].lastUse) })
		for _, k := range keys {
			if len(r.pools) <= r.cfg.MaxPools {
				break
			}
			out = append(out, r.pools[k].pool)
			delete(r.pools, k)
		}
	}
	return out
}

// Size is the number of live pools (for tests and metrics).
func (r *Router) Size() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.pools)
}

// Close releases every pool; call it from shutdown.
func (r *Router) Close() {
	r.mu.Lock()
	var all []*pgxpool.Pool
	for k, e := range r.pools {
		all = append(all, e.pool)
		delete(r.pools, k)
	}
	r.mu.Unlock()
	for _, p := range all {
		p.Close()
	}
}

func assertIdentity(ctx context.Context, c *pgx.Conn, wantDB, wantUser string) error {
	var db, user string
	if err := c.QueryRow(ctx, `SELECT current_database(), current_user`).Scan(&db, &user); err != nil {
		return err
	}
	if db != wantDB {
		return fmt.Errorf("%w: got %q, want %q", ErrDatabaseMismatch, db, wantDB)
	}
	if user != wantUser {
		return fmt.Errorf("%w: got %q", ErrUserMismatch, user)
	}
	return nil
}

// Pool is a tenant's database, already proven to be the right one.
type Pool struct {
	ds Datasource
	p  *pgxpool.Pool
}

// Database is the registered database name.
func (p *Pool) Database() string { return p.ds.Database }

// WithTx runs fn in a transaction, committing if fn returns nil. fn never sees the pool.
func (p *Pool) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := p.p.Begin(ctx)
	if err != nil {
		return fmt.Errorf("tenantdb: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // a no-op after Commit
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("tenantdb: commit: %w", err)
	}
	return nil
}
