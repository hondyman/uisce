// Package sourceconn is the one audited way to open a tenant's SOURCE database: the Postgres a
// datasource row points at (a customer system, or the backend a business object's records live in).
// It is the sibling of internal/tenantdb, which serves the per-tenant database the platform
// provisions and binds. A source has no binding and its host is the customer's, so it cannot assert
// "the registered database"; what it enforces instead:
//
//   - the datasource is looked up for the VERIFIED caller tenant under an explicit Policy, never by
//     id alone (OwnerOnly, or OwnerOrGoldCopy for the gold-copy inheritance business objects use);
//   - credentials come from the secrets store through dscreds, by the datasource ROW's own tenant;
//   - there is no default host, database or credential and no fallback to any other database;
//   - connections are pooled with a bound, keyed by the credential, so a rotated secret gets a new
//     pool and a datasource that is deactivated stops resolving within AuthTTL.
//
// archguard allows connections to be opened only here (and in tenantdb for bound tenant databases).
package sourceconn

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"crypto/sha256"
	"encoding/hex"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/sync/singleflight"
)

var (
	// ErrNoTenant: the caller's context carries no verified tenant.
	ErrNoTenant = errors.New("sourceconn: no tenant in context")
	// ErrNotFound: no active datasource with that id.
	ErrNotFound = errors.New("sourceconn: datasource not found")
	// ErrNotAllowed: the datasource exists but is not the caller's under the policy.
	ErrNotAllowed = errors.New("sourceconn: datasource not available to the caller's tenant")
)

// Policy says whose datasources a caller may open.
type Policy int

const (
	// OwnerOnly: the datasource must belong to the caller's tenant.
	OwnerOnly Policy = iota
	// OwnerOrGoldCopy: or to the gold-copy tenant, whose metadata every tenant inherits read-only.
	// Business-object records use it; nothing else should.
	OwnerOrGoldCopy
)

// Source is a datasource row the caller may open. Config is its connection config as stored (the
// credential is NOT in it when the datasource has a secret_path).
type Source struct {
	ID       string
	TenantID string // the row's own tenant: credentials are resolved by it, not by the caller's
	Config   []byte
	GoldCopy bool
}

// Registry finds the datasource for a caller. An unknown id is ErrNotFound; one the caller may not
// use is ErrNotAllowed. Implemented over alpha by AlphaRegistry.
type Registry interface {
	Source(ctx context.Context, tenantID, datasourceID string, p Policy) (Source, error)
}

// Credentials turns a Source's stored config into complete connection details (credential filled in
// from the secrets store).
type Credentials interface {
	Hydrate(ctx context.Context, s Source) ([]byte, error)
}

// CallerTenant returns the verified tenant of the request.
type CallerTenant func(ctx context.Context) (string, error)

// Config configures a Connector. Every limit is required.
type Config struct {
	Registry     Registry
	Credentials  Credentials
	CallerTenant CallerTenant

	MaxPools        int
	MaxConnsPerPool int32
	IdleTTL         time.Duration
	DialTimeout     time.Duration

	// AuthTTL reuses a SUCCESSFUL lookup (ownership + credentials) for this long; 0 disables it.
	// Same trade as tenantdb.Config.AuthTTL: a deactivation or credential rotation is missed for at
	// most this long; refusals are never cached.
	AuthTTL time.Duration
	// Warn receives non-fatal findings (an mTLS config that does not verify the server). May be nil.
	Warn func(msg string)
}

// DefaultAuthTTL is what production wiring uses.
const DefaultAuthTTL = 3 * time.Second

type poolKey struct {
	tenant, datasourceID, credHash string
}

type poolEntry struct {
	pool    *pgxpool.Pool
	lastUse time.Time
	view    *sqlView
}

type sqlView struct {
	once sync.Once
	db   *sql.DB
}

type authKey struct {
	tenant, datasourceID string
	policy               Policy
}

type authEntry struct {
	key     poolKey
	hydr    []byte
	expires time.Time
}

// Connector opens and pools source databases.
type Connector struct {
	cfg    Config
	now    func() time.Time
	mu     sync.Mutex
	pools  map[poolKey]*poolEntry
	auth   map[authKey]authEntry
	flight singleflight.Group
}

// New validates the config.
func New(cfg Config) (*Connector, error) {
	switch {
	case cfg.Registry == nil || cfg.Credentials == nil || cfg.CallerTenant == nil:
		return nil, errors.New("sourceconn: Registry, Credentials and CallerTenant are required")
	case cfg.MaxPools <= 0 || cfg.MaxConnsPerPool <= 0 || cfg.IdleTTL <= 0 || cfg.DialTimeout <= 0:
		return nil, errors.New("sourceconn: MaxPools, MaxConnsPerPool, IdleTTL and DialTimeout must be positive")
	case cfg.AuthTTL < 0:
		return nil, errors.New("sourceconn: AuthTTL must not be negative")
	}
	return &Connector{cfg: cfg, now: time.Now, pools: map[poolKey]*poolEntry{}, auth: map[authKey]authEntry{}}, nil
}

// Pool returns the pooled connection to a datasource the caller may open under the policy.
func (c *Connector) Pool(ctx context.Context, datasourceID string, p Policy) (*pgxpool.Pool, error) {
	e, err := c.entry(ctx, datasourceID, p)
	if err != nil {
		return nil, err
	}
	return e.pool, nil
}

// SQLDB is a database/sql view of the same pool, for callers written against *sql.DB. It is the
// SAME pool (capped by MaxConnsPerPool, no idle connections of its own, and closing it does not
// close the pool). Do not Close it.
func (c *Connector) SQLDB(ctx context.Context, datasourceID string, p Policy) (*sql.DB, error) {
	e, err := c.entry(ctx, datasourceID, p)
	if err != nil {
		return nil, err
	}
	e.view.once.Do(func() { e.view.db = stdlib.OpenDBFromPool(e.pool) })
	return e.view.db, nil
}

// OpenAdHoc opens a connection from details the caller typed but has not saved (a "test this
// connection" form). It is not pooled or cached, and nothing is looked up or authorized here: the
// caller must already have decided the requester may probe an arbitrary host. The returned *sql.DB
// is the caller's to Close.
func (c *Connector) OpenAdHoc(ctx context.Context, rawDetails []byte) (*sql.DB, error) {
	cc, _, _, _, warns, err := connConfig(rawDetails)
	if err != nil {
		return nil, err
	}
	for _, w := range warns {
		c.warn(w)
	}
	db := stdlib.OpenDB(*cc)
	pctx, cancel := context.WithTimeout(ctx, c.cfg.DialTimeout)
	defer cancel()
	if err := db.PingContext(pctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("sourceconn: ping: %w", err)
	}
	return db, nil
}

func (c *Connector) warn(msg string) {
	if c.cfg.Warn != nil {
		c.cfg.Warn(msg)
	}
}

func (c *Connector) entry(ctx context.Context, datasourceID string, p Policy) (*poolEntry, error) {
	actor, err := c.cfg.CallerTenant(ctx)
	if err != nil || actor == "" {
		return nil, ErrNoTenant
	}
	ak := authKey{actor, datasourceID, p}
	a, err := c.authorize(ctx, ak)
	if err != nil {
		return nil, err
	}
	return c.pool(ctx, a)
}

// authorize is the lookup (ownership under the policy, then credentials), cached for AuthTTL.
func (c *Connector) authorize(ctx context.Context, ak authKey) (authEntry, error) {
	if c.cfg.AuthTTL > 0 {
		c.mu.Lock()
		if e, ok := c.auth[ak]; ok && c.now().Before(e.expires) {
			c.mu.Unlock()
			return e, nil
		}
		c.mu.Unlock()
	}
	do := func(ctx context.Context) (authEntry, error) {
		src, err := c.cfg.Registry.Source(ctx, ak.tenant, ak.datasourceID, ak.policy)
		if err != nil {
			return authEntry{}, err
		}
		hydr, err := c.cfg.Credentials.Hydrate(ctx, src)
		if err != nil {
			return authEntry{}, fmt.Errorf("sourceconn: credentials for datasource %s: %w", src.ID, err)
		}
		sum := sha256.Sum256(hydr)
		e := authEntry{key: poolKey{ak.tenant, src.ID, hex.EncodeToString(sum[:])}, hydr: hydr}
		if c.cfg.AuthTTL > 0 {
			e.expires = c.now().Add(c.cfg.AuthTTL)
			c.mu.Lock()
			c.auth[ak] = e
			c.trimAuthLocked()
			c.mu.Unlock()
		}
		return e, nil
	}
	if c.cfg.AuthTTL <= 0 {
		return do(ctx)
	}
	ch := c.flight.DoChan(fmt.Sprintf("%s\x00%s\x00%d", ak.tenant, ak.datasourceID, ak.policy), func() (any, error) {
		cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), c.cfg.DialTimeout)
		defer cancel()
		return do(cctx)
	})
	select {
	case res := <-ch:
		if res.Err != nil {
			return authEntry{}, res.Err
		}
		return res.Val.(authEntry), nil
	case <-ctx.Done():
		return authEntry{}, ctx.Err()
	}
}

// trimAuthLocked bounds the authorization cache.
func (c *Connector) trimAuthLocked() {
	max := 4 * c.cfg.MaxPools
	if len(c.auth) <= max {
		return
	}
	now := c.now()
	for k, e := range c.auth {
		if !now.Before(e.expires) {
			delete(c.auth, k)
		}
	}
	for len(c.auth) > max {
		var oldest authKey
		var t time.Time
		first := true
		for k, e := range c.auth {
			if first || e.expires.Before(t) {
				oldest, t, first = k, e.expires, false
			}
		}
		delete(c.auth, oldest)
	}
}

// InvalidateDatasource forgets every cached authorization for a datasource, so the next call asks
// alpha again. Call it where this process deactivates or re-credentials a datasource.
func (c *Connector) InvalidateDatasource(datasourceID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.auth {
		if k.datasourceID == datasourceID {
			delete(c.auth, k)
		}
	}
}

func (c *Connector) pool(ctx context.Context, a authEntry) (*poolEntry, error) {
	c.mu.Lock()
	if e, ok := c.pools[a.key]; ok {
		e.lastUse = c.now()
		c.mu.Unlock()
		return e, nil
	}
	c.mu.Unlock()

	cc, _, _, _, warns, err := connConfig(a.hydr)
	if err != nil {
		return nil, err
	}
	for _, w := range warns {
		c.warn(w)
	}
	pc, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, err
	}
	pc.ConnConfig = cc
	pc.ConnConfig.ConnectTimeout = c.cfg.DialTimeout
	pc.MaxConns = c.cfg.MaxConnsPerPool
	pc.MinConns = 0
	pc.MaxConnIdleTime = c.cfg.IdleTTL
	pc.AfterRelease = func(conn *pgx.Conn) bool {
		rctx, cancel := context.WithTimeout(context.Background(), c.cfg.DialTimeout)
		defer cancel()
		_, err := conn.Exec(rctx, "RESET ALL") // a connection with leftover session state is never reused
		return err == nil
	}
	p, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, fmt.Errorf("sourceconn: open pool: %w", err)
	}
	pctx, cancel := context.WithTimeout(ctx, c.cfg.DialTimeout)
	defer cancel()
	if err := p.Ping(pctx); err != nil {
		p.Close()
		return nil, fmt.Errorf("sourceconn: connect (timeout %v): %w", c.cfg.DialTimeout, err)
	}

	c.mu.Lock()
	if e, ok := c.pools[a.key]; ok { // lost a race
		c.mu.Unlock()
		p.Close()
		return e, nil
	}
	e := &poolEntry{pool: p, lastUse: c.now(), view: &sqlView{}}
	c.pools[a.key] = e
	victims := c.evictLocked(a.key)
	c.mu.Unlock()
	for _, v := range victims {
		v.Close() // outside the lock: Close waits for in-flight connections
	}
	return e, nil
}

// evictLocked closes pools idle past IdleTTL, then the least recently used past MaxPools, never the
// one just built. It also drops pools for a superseded credential of the same datasource: a rotated
// secret must not leave the old one's connections open.
func (c *Connector) evictLocked(keep poolKey) []*pgxpool.Pool {
	var out []*pgxpool.Pool
	now := c.now()
	for k, e := range c.pools {
		stale := k != keep && now.Sub(e.lastUse) > c.cfg.IdleTTL
		superseded := k != keep && k.tenant == keep.tenant && k.datasourceID == keep.datasourceID && k.credHash != keep.credHash
		if stale || superseded {
			out = append(out, e.pool)
			delete(c.pools, k)
		}
	}
	if len(c.pools) > c.cfg.MaxPools {
		keys := make([]poolKey, 0, len(c.pools))
		for k := range c.pools {
			if k != keep {
				keys = append(keys, k)
			}
		}
		sort.Slice(keys, func(i, j int) bool { return c.pools[keys[i]].lastUse.Before(c.pools[keys[j]].lastUse) })
		for _, k := range keys {
			if len(c.pools) <= c.cfg.MaxPools {
				break
			}
			out = append(out, c.pools[k].pool)
			delete(c.pools, k)
		}
	}
	return out
}

// Size is the number of live pools.
func (c *Connector) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.pools)
}

// Close releases every pool.
func (c *Connector) Close() {
	c.mu.Lock()
	c.auth = map[authKey]authEntry{}
	var all []*pgxpool.Pool
	for k, e := range c.pools {
		all = append(all, e.pool)
		delete(c.pools, k)
	}
	c.mu.Unlock()
	for _, p := range all {
		p.Close()
	}
}
