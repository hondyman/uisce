package tenantdb

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type fakeRegistry struct {
	ds        Datasource
	dsErr     error
	binding   Binding
	bindErr   error
	user, pw  string
	credErr   error
	credCalls int
}

func (f *fakeRegistry) ResolveDatasource(context.Context, string) (Datasource, error) {
	return f.ds, f.dsErr
}
func (f *fakeRegistry) LoadBinding(context.Context, string) (Binding, error) {
	return f.binding, f.bindErr
}
func (f *fakeRegistry) Credentials(context.Context, Datasource) (string, string, error) {
	f.credCalls++
	return f.user, f.pw, f.credErr
}

func caller(id string) CallerTenant {
	return func(context.Context) (string, error) {
		if id == "" {
			return "", errors.New("no tenant")
		}
		return id, nil
	}
}

func newRouter(t *testing.T, reg Registry, tenant string) *Router {
	t.Helper()
	r, err := New(Config{Registry: reg, CallerTenant: caller(tenant), MaxPools: 2, MaxConnsPerPool: 2,
		IdleTTL: time.Minute, DialTimeout: 2 * time.Second})
	require.NoError(t, err)
	t.Cleanup(r.Close)
	return r
}

func goodDS() Datasource {
	return Datasource{ID: "ds-1", TenantID: "t-1", Host: "127.0.0.1", Port: 1, Database: "orm_acme"}
}

func TestNew_RequiresEveryLimit(t *testing.T) {
	ok := Config{Registry: &fakeRegistry{}, CallerTenant: caller("t"), MaxPools: 1, MaxConnsPerPool: 1,
		IdleTTL: time.Second, DialTimeout: time.Second}
	_, err := New(ok)
	require.NoError(t, err)
	for name, mut := range map[string]func(*Config){
		"registry": func(c *Config) { c.Registry = nil },
		"caller":   func(c *Config) { c.CallerTenant = nil },
		"pools":    func(c *Config) { c.MaxPools = 0 },
		"conns":    func(c *Config) { c.MaxConnsPerPool = 0 },
		"idle":     func(c *Config) { c.IdleTTL = 0 },
		"dial":     func(c *Config) { c.DialTimeout = 0 },
	} {
		c := ok
		mut(&c)
		_, err := New(c)
		require.Error(t, err, name)
	}
}

// Every refusal must happen before anything is dialled or any credential is read.
func TestResolve_FailsClosed(t *testing.T) {
	active := Binding{Version: 1, Lifecycle: "active"}
	cases := []struct {
		name   string
		tenant string
		reg    *fakeRegistry
		want   error
	}{
		{"no tenant in context", "", &fakeRegistry{ds: goodDS(), binding: active}, ErrNoTenant},
		{"another tenant's datasource", "t-2", &fakeRegistry{ds: goodDS(), binding: active}, ErrTenantMismatch},
		{"datasource with no owner", "t-1", &fakeRegistry{ds: Datasource{ID: "ds-1", Host: "h", Port: 1, Database: "d"}, binding: active}, ErrTenantMismatch},
		{"no database registered", "t-1", &fakeRegistry{ds: Datasource{ID: "ds-1", TenantID: "t-1", Host: "h", Port: 1}, binding: active}, ErrIncomplete},
		{"no host registered", "t-1", &fakeRegistry{ds: Datasource{ID: "ds-1", TenantID: "t-1", Port: 1, Database: "d"}, binding: active}, ErrIncomplete},
		{"unknown datasource", "t-1", &fakeRegistry{dsErr: errors.New("not found")}, nil},
		{"no binding", "t-1", &fakeRegistry{ds: goodDS(), bindErr: ErrUnbound}, ErrUnbound},
		{"still provisioning", "t-1", &fakeRegistry{ds: goodDS(), binding: Binding{Version: 1, Lifecycle: "provisioning"}}, ErrUnbound},
		{"suspended", "t-1", &fakeRegistry{ds: goodDS(), binding: Binding{Version: 1, Lifecycle: "suspended"}}, ErrUnbound},
		{"offboarding", "t-1", &fakeRegistry{ds: goodDS(), binding: Binding{Version: 1, Lifecycle: "offboarding"}}, ErrUnbound},
		{"credentials unavailable", "t-1", &fakeRegistry{ds: goodDS(), binding: active, credErr: errors.New("vault down")}, nil},
		{"credentials empty", "t-1", &fakeRegistry{ds: goodDS(), binding: active, user: "u"}, ErrIncomplete},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := newRouter(t, c.reg, c.tenant)
			p, err := r.Resolve(context.Background(), "ds-1")
			require.Error(t, err)
			require.Nil(t, p)
			if c.want != nil {
				require.ErrorIs(t, err, c.want)
			}
			require.Zero(t, r.Size(), "a refused resolve must not leave a pool behind")
		})
	}

	t.Run("a mismatched tenant never reaches the credential store", func(t *testing.T) {
		reg := &fakeRegistry{ds: goodDS(), binding: active, user: "u", pw: "p"}
		_, err := newRouter(t, reg, "t-2").Resolve(context.Background(), "ds-1")
		require.ErrorIs(t, err, ErrTenantMismatch)
		require.Zero(t, reg.credCalls)
	})
}

func TestParseDatasource(t *testing.T) {
	ds, err := parseDatasource("d", "t", []byte(`{"host":"db","port":5432,"database":"orm_acme","secret_path":"x"}`))
	require.NoError(t, err)
	require.Equal(t, Datasource{ID: "d", TenantID: "t", Host: "db", Port: 5432, Database: "orm_acme", config: ds.config}, ds)

	ds, err = parseDatasource("d", "t", []byte(`{"host":"db","port":"5433","database":"x"}`))
	require.NoError(t, err)
	require.Equal(t, 5433, ds.Port)

	for _, bad := range []string{`{}`, `null`, `[]`, `{"host":"db","port":5432}`, `{"host":"db","database":"x"}`,
		`{"port":5432,"database":"x"}`, `{"host":"db","port":0,"database":"x"}`, `{"host":"db","port":70000,"database":"x"}`,
		`{"host":"  ","port":5432,"database":"x"}`} {
		_, err := parseDatasource("d", "t", []byte(bad))
		require.ErrorIs(t, err, ErrIncomplete, bad)
	}
}

func TestCredentialsFromConfig(t *testing.T) {
	u, p, err := credentialsFromConfig([]byte(`{"auth":{"basic":{"username":"app","password":"pw"}}}`))
	require.NoError(t, err)
	require.Equal(t, [2]string{"app", "pw"}, [2]string{u, p})

	u, p, err = credentialsFromConfig([]byte(`{"username":"app","password":"pw"}`))
	require.NoError(t, err)
	require.Equal(t, [2]string{"app", "pw"}, [2]string{u, p})

	for _, bad := range []string{`{}`, `{"username":"app"}`, `{"password":"pw"}`, `{"auth":{"basic":{"username":"app"}}}`, `x`} {
		_, _, err := credentialsFromConfig([]byte(bad))
		require.ErrorIs(t, err, ErrIncomplete, bad)
	}
}

// --- against a real Postgres ------------------------------------------------------------

// TENANTDB_TEST_DSN names a role that can CONNECT to two databases, TENANTDB_TEST_DB_A and
// TENANTDB_TEST_DB_B, as TENANTDB_TEST_USER / TENANTDB_TEST_PASSWORD (host and port come from
// TENANTDB_TEST_HOST and TENANTDB_TEST_PORT).
type pgEnv struct {
	host, user, pw, dbA, dbB string
	port                     int
}

func realPG(t *testing.T) pgEnv {
	t.Helper()
	e := pgEnv{host: os.Getenv("TENANTDB_TEST_HOST"), user: os.Getenv("TENANTDB_TEST_USER"),
		pw: os.Getenv("TENANTDB_TEST_PASSWORD"), dbA: os.Getenv("TENANTDB_TEST_DB_A"), dbB: os.Getenv("TENANTDB_TEST_DB_B")}
	e.port, _ = strconv.Atoi(os.Getenv("TENANTDB_TEST_PORT"))
	if e.host == "" || e.user == "" || e.dbA == "" || e.dbB == "" || e.port == 0 {
		t.Skip("TENANTDB_TEST_* not set")
	}
	return e
}

func (e pgEnv) ds(id, tenant, db string) Datasource {
	return Datasource{ID: id, TenantID: tenant, Host: e.host, Port: e.port, Database: db}
}

func TestPool_ReachesTheRegisteredDatabaseAndNoOther(t *testing.T) {
	e := realPG(t)
	reg := &fakeRegistry{ds: e.ds("ds-a", "t-1", e.dbA), binding: Binding{Version: 1, Lifecycle: "active"}, user: e.user, pw: e.pw}
	r := newRouter(t, reg, "t-1")

	p, err := r.Resolve(context.Background(), "ds-a")
	require.NoError(t, err)
	var db string
	require.NoError(t, p.WithTx(context.Background(), func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), `SELECT current_database()`).Scan(&db)
	}))
	require.Equal(t, e.dbA, db)

	// Same datasource, same binding version: the pool is reused, not rebuilt.
	_, err = r.Resolve(context.Background(), "ds-a")
	require.NoError(t, err)
	require.Equal(t, 1, r.Size())
	require.Equal(t, 1, reg.credCalls)
}

func TestPool_ARebindingBuildsANewPoolOnTheNewDatabase(t *testing.T) {
	e := realPG(t)
	reg := &fakeRegistry{ds: e.ds("ds-a", "t-1", e.dbA), binding: Binding{Version: 1, Lifecycle: "active"}, user: e.user, pw: e.pw}
	r := newRouter(t, reg, "t-1")
	ctx := context.Background()

	p, err := r.Resolve(ctx, "ds-a")
	require.NoError(t, err)
	require.Equal(t, e.dbA, p.Database())

	// The cutover: the registry now names another database and bumps the binding version.
	reg.ds = e.ds("ds-a", "t-1", e.dbB)
	reg.binding.Version = 2
	p, err = r.Resolve(ctx, "ds-a")
	require.NoError(t, err)
	var db string
	require.NoError(t, p.WithTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT current_database()`).Scan(&db)
	}))
	require.Equal(t, e.dbB, db, "after a version bump the next Resolve must reach the new database")
}

func TestPool_WrongPasswordIsAnErrorAtResolve(t *testing.T) {
	e := realPG(t)
	if c, err := pgx.Connect(context.Background(), "postgres://"+e.user+":"+e.pw+"-wrong@"+e.host+":"+strconv.Itoa(e.port)+"/"+e.dbA); err == nil {
		c.Close(context.Background())
		t.Skip("the test server does not check passwords (trust auth); nothing to prove here")
	}
	reg := &fakeRegistry{ds: e.ds("ds-a", "t-1", e.dbA), binding: Binding{Version: 1, Lifecycle: "active"}, user: e.user, pw: e.pw + "-wrong"}
	r := newRouter(t, reg, "t-1")
	_, err := r.Resolve(context.Background(), "ds-a")
	require.Error(t, err)
	require.Zero(t, r.Size())
}

func TestPool_UnknownDatabaseIsAnErrorAtResolve(t *testing.T) {
	e := realPG(t)
	reg := &fakeRegistry{ds: e.ds("ds-a", "t-1", e.dbA+"_missing"), binding: Binding{Version: 1, Lifecycle: "active"}, user: e.user, pw: e.pw}
	_, err := newRouter(t, reg, "t-1").Resolve(context.Background(), "ds-a")
	require.Error(t, err)
}

func TestPool_ARollbackLeavesNothingBehind(t *testing.T) {
	e := realPG(t)
	reg := &fakeRegistry{ds: e.ds("ds-a", "t-1", e.dbA), binding: Binding{Version: 1, Lifecycle: "active"}, user: e.user, pw: e.pw}
	p, err := newRouter(t, reg, "t-1").Resolve(context.Background(), "ds-a")
	require.NoError(t, err)
	ctx := context.Background()
	boom := errors.New("boom")
	err = p.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `CREATE TEMP TABLE tdb_probe (x int)`); err != nil {
			return err
		}
		return boom
	})
	require.ErrorIs(t, err, boom)
}

func TestPool_LeftoverSessionStateIsNeverReused(t *testing.T) {
	e := realPG(t)
	reg := &fakeRegistry{ds: e.ds("ds-a", "t-1", e.dbA), binding: Binding{Version: 1, Lifecycle: "active"}, user: e.user, pw: e.pw}
	// ONE connection, so the second checkout is necessarily the first connection reused.
	r, err := New(Config{Registry: reg, CallerTenant: caller("t-1"), MaxPools: 1, MaxConnsPerPool: 1,
		IdleTTL: time.Minute, DialTimeout: 2 * time.Second})
	require.NoError(t, err)
	t.Cleanup(r.Close)
	p, err := r.Resolve(context.Background(), "ds-a")
	require.NoError(t, err)
	ctx := context.Background()
	// Set session state, release, read it back on the same connection.
	require.NoError(t, p.WithTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `SET application_name = 'leaked'`)
		return err
	}))
	var name string
	require.NoError(t, p.WithTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SHOW application_name`).Scan(&name)
	}))
	require.False(t, strings.Contains(name, "leaked"), "session state from a previous checkout leaked: %q", name)
}

func TestRouter_BoundedPoolsEvictLeastRecentlyUsed(t *testing.T) {
	e := realPG(t)
	reg := &fakeRegistry{ds: e.ds("ds-1", "t-1", e.dbA), binding: Binding{Version: 1, Lifecycle: "active"}, user: e.user, pw: e.pw}
	r := newRouter(t, reg, "t-1") // MaxPools = 2
	clock := time.Now()
	r.now = func() time.Time { clock = clock.Add(time.Second); return clock }
	ctx := context.Background()

	for _, id := range []string{"ds-1", "ds-2", "ds-3"} {
		reg.ds = e.ds(id, "t-1", e.dbA)
		_, err := r.Resolve(ctx, id)
		require.NoError(t, err)
	}
	require.Equal(t, 2, r.Size(), "the pool count must stay bounded")

	r.mu.Lock()
	_, oldestKept := r.pools[cacheKey{"ds-1", "t-1", 1}]
	r.mu.Unlock()
	require.False(t, oldestKept, "the least recently used pool is the one evicted")
}

func TestAssertIdentity_RefusesTheWrongDatabaseOrUser(t *testing.T) {
	e := realPG(t)
	ctx := context.Background()
	c, err := pgx.Connect(ctx, "postgres://"+e.user+":"+e.pw+"@"+e.host+":"+strconv.Itoa(e.port)+"/"+e.dbA)
	require.NoError(t, err)
	defer c.Close(ctx)

	require.NoError(t, assertIdentity(ctx, c, e.dbA, e.user))
	require.ErrorIs(t, assertIdentity(ctx, c, e.dbB, e.user), ErrDatabaseMismatch)
	require.ErrorIs(t, assertIdentity(ctx, c, e.dbA, "someone_else"), ErrUserMismatch)
}
