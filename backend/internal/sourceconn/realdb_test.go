package sourceconn

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	uiscedb "github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/dscreds"
	"github.com/hondyman/uisce/backend/internal/secrets"
	"github.com/hondyman/uisce/backend/internal/security"
)

// Real-Postgres tests. Two environments (all skipped when unset):
//
//	SOURCECONN_ALPHA_ADMIN_DSN / SOURCECONN_ALPHA_APP_DSN
//	    a SCRATCH database standing in for alpha (its public schema is dropped; the name must contain
//	    "alpha" and not be "alpha") and the ordinary role the code runs as (no superuser, no
//	    BYPASSRLS, a member of uisce_gold_copy_sync). The datasource chain is rebuilt with strict RLS
//	    on the datasource table; that policy stands in for production's.
//	SOURCECONN_TEST_HOST/PORT/USER/PASSWORD/DB
//	    a source database the role can connect to.

const alphaSchema = `
CREATE FUNCTION uisce_get_current_tenant() RETURNS uuid AS $$
BEGIN
    RETURN NULLIF(current_setting('uisce.current_tenant', true), '')::uuid;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;
CREATE TABLE tenants (id uuid PRIMARY KEY, name text, code text, allowed_regions jsonb, gold_copy bool NOT NULL DEFAULT false);
CREATE TABLE tenant_instance (id uuid PRIMARY KEY, tenant_id uuid NOT NULL REFERENCES tenants(id), is_active bool NOT NULL DEFAULT true);
CREATE TABLE tenant_product (id uuid PRIMARY KEY, datasource_id uuid NOT NULL REFERENCES tenant_instance(id), is_active bool NOT NULL DEFAULT true);
CREATE TABLE tenant_product_datasource (id uuid PRIMARY KEY, tenant_product_id uuid NOT NULL REFERENCES tenant_product(id),
    is_active bool NOT NULL DEFAULT true, config jsonb NOT NULL DEFAULT '{}');
ALTER TABLE tenant_product_datasource ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_product_datasource FORCE ROW LEVEL SECURITY;
CREATE POLICY tpd_isolation ON tenant_product_datasource FOR ALL USING (
    EXISTS (SELECT 1 FROM tenant_product tp JOIN tenant_instance ti ON ti.id = tp.datasource_id
            WHERE tp.id = tenant_product_datasource.tenant_product_id AND ti.tenant_id = uisce_get_current_tenant()));
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'uisce_gold_copy_sync') THEN
        CREATE ROLE uisce_gold_copy_sync NOLOGIN BYPASSRLS;
    END IF;
END $$;
GRANT USAGE ON SCHEMA public TO PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO PUBLIC;
`

type alpha struct{ app *sql.DB }

func newAlpha(t *testing.T) *alpha {
	t.Helper()
	admin, app := os.Getenv("SOURCECONN_ALPHA_ADMIN_DSN"), os.Getenv("SOURCECONN_ALPHA_APP_DSN")
	if admin == "" || app == "" {
		t.Skip("SOURCECONN_ALPHA_*_DSN not set")
	}
	adm, err := sql.Open("pgx", admin)
	require.NoError(t, err)
	t.Cleanup(func() { adm.Close() })
	var name string
	require.NoError(t, adm.QueryRow(`SELECT current_database()`).Scan(&name))
	require.Contains(t, name, "alpha")
	require.NotEqual(t, "alpha", name, "refusing to run against the shared alpha database")
	for _, q := range []string{`DROP SCHEMA public CASCADE`, `CREATE SCHEMA public`, alphaSchema} {
		_, err := adm.Exec(q)
		require.NoError(t, err)
	}
	db, err := sql.Open("pgx", app)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	var bypass bool
	require.NoError(t, db.QueryRow(`SELECT rolsuper OR rolbypassrls FROM pg_roles WHERE rolname = current_user`).Scan(&bypass))
	require.False(t, bypass, "the app role bypasses row-level security; the isolation tests would be vacuous")
	return &alpha{app: db}
}

// tenant creates a tenant with one instance and product and returns the ids.
func (a *alpha) tenant(t *testing.T, gold bool) (tenant, product string) {
	t.Helper()
	tenant, inst, product := uuid.NewString(), uuid.NewString(), uuid.NewString()
	tx, err := a.app.Begin()
	require.NoError(t, err)
	_, err = tx.Exec(`SELECT set_config('uisce.current_tenant', $1, true)`, tenant)
	require.NoError(t, err)
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO tenants (id, name, code, gold_copy) VALUES ($1, 'n', $2, $3)`, []any{tenant, "c" + tenant[:8], gold}},
		{`INSERT INTO tenant_instance (id, tenant_id) VALUES ($1, $2)`, []any{inst, tenant}},
		{`INSERT INTO tenant_product (id, datasource_id) VALUES ($1, $2)`, []any{product, inst}},
	} {
		_, err := tx.Exec(q.sql, q.args...)
		require.NoError(t, err, q.sql)
	}
	require.NoError(t, tx.Commit())
	return tenant, product
}

func (a *alpha) datasource(t *testing.T, tenant, product, config string, active bool) string {
	t.Helper()
	id := uuid.NewString()
	tx, err := a.app.Begin()
	require.NoError(t, err)
	_, err = tx.Exec(`SELECT set_config('uisce.current_tenant', $1, true)`, tenant)
	require.NoError(t, err)
	_, err = tx.Exec(`INSERT INTO tenant_product_datasource (id, tenant_product_id, is_active, config) VALUES ($1, $2, $3, $4::jsonb)`, id, product, active, config)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	return id
}

func (a *alpha) registry() *AlphaRegistry {
	return &AlphaRegistry{DB: a.app, Resolver: security.NewDBDatasourceResolver(sqlx.NewDb(a.app, "pgx"))}
}

func TestAlphaRegistry_Policy(t *testing.T) {
	a := newAlpha(t)
	ctx := context.Background()
	t1, p1 := a.tenant(t, false)
	t2, p2 := a.tenant(t, false)
	gold, pg := a.tenant(t, true)
	dsOwn := a.datasource(t, t1, p1, `{"host":"h","port":5432,"database":"d","username":"u"}`, true)
	dsOther := a.datasource(t, t2, p2, `{"host":"h","port":5432,"database":"d","username":"u"}`, true)
	dsGold := a.datasource(t, gold, pg, `{"host":"h","port":5432,"database":"d","username":"u"}`, true)
	dsInactive := a.datasource(t, t1, p1, `{"host":"h"}`, false)
	dsEmpty := a.datasource(t, t1, p1, `{}`, true)
	dsNull := a.datasource(t, t1, p1, `null`, true)
	reg := a.registry()

	t.Run("a tenant's own datasource", func(t *testing.T) {
		for _, p := range []Policy{OwnerOnly, OwnerOrGoldCopy} {
			s, err := reg.Source(ctx, t1, dsOwn, p)
			require.NoError(t, err)
			require.Equal(t, t1, s.TenantID)
			require.False(t, s.GoldCopy)
			require.Contains(t, string(s.Config), `"database"`)
		}
	})

	t.Run("another tenant's datasource is refused under both policies", func(t *testing.T) {
		for _, p := range []Policy{OwnerOnly, OwnerOrGoldCopy} {
			_, err := reg.Source(ctx, t1, dsOther, p)
			require.ErrorIs(t, err, ErrNotAllowed, "policy %d", p)
			_, err = reg.Source(ctx, t2, dsOwn, p)
			require.ErrorIs(t, err, ErrNotAllowed)
		}
	})

	t.Run("the gold copy's datasource is reachable only under OwnerOrGoldCopy", func(t *testing.T) {
		_, err := reg.Source(ctx, t1, dsGold, OwnerOnly)
		require.ErrorIs(t, err, ErrNotAllowed, "OwnerOnly never reaches the gold copy's datasource")
		s, err := reg.Source(ctx, t1, dsGold, OwnerOrGoldCopy)
		require.NoError(t, err)
		require.Equal(t, gold, s.TenantID, "credentials are resolved by the ROW's tenant, not the caller's")
		require.True(t, s.GoldCopy)
	})

	t.Run("unknown, malformed and inactive ids are not found", func(t *testing.T) {
		for _, id := range []string{uuid.NewString(), "not-a-uuid", "", dsInactive} {
			_, err := reg.Source(ctx, t1, id, OwnerOnly)
			require.ErrorIs(t, err, ErrNotFound, id)
		}
	})

	t.Run("a datasource with no connection configuration is a bad config, never a guess", func(t *testing.T) {
		_, err := reg.Source(ctx, t1, dsNull, OwnerOnly)
		require.ErrorIs(t, err, ErrBadConfig)
		// An empty object is a stored config; connConfig refuses it before any dial.
		_, err = reg.Source(ctx, t1, dsEmpty, OwnerOnly)
		require.NoError(t, err)
		_, _, _, _, _, err = connConfig([]byte(`{}`))
		require.ErrorIs(t, err, ErrBadConfig)
	})
}

func srcEnv(t *testing.T) (host string, port int, user, pw, db string) {
	t.Helper()
	host, user, pw, db = os.Getenv("SOURCECONN_TEST_HOST"), os.Getenv("SOURCECONN_TEST_USER"), os.Getenv("SOURCECONN_TEST_PASSWORD"), os.Getenv("SOURCECONN_TEST_DB")
	port, _ = strconv.Atoi(os.Getenv("SOURCECONN_TEST_PORT"))
	if host == "" || user == "" || db == "" || port == 0 {
		t.Skip("SOURCECONN_TEST_* not set")
	}
	return
}

func realConnector(t *testing.T, a *alpha, store secrets.Provider, ttl time.Duration, maxPools int) *Connector {
	t.Helper()
	c, err := New(Config{
		Registry: a.registry(), Credentials: DSCreds{R: dscreds.NewResolver(store, dscreds.WithCacheTTL(0))},
		CallerTenant: uiscedb.GetTenantIDFromCtx, MaxPools: maxPools, MaxConnsPerPool: 2, IdleTTL: time.Minute,
		DialTimeout: 3 * time.Second, AuthTTL: ttl,
	})
	require.NoError(t, err)
	t.Cleanup(c.Close)
	return c
}

func tctx(tenant string) context.Context {
	return uiscedb.WithTenantContextToCtx(context.Background(), tenant)
}

func TestConnector_ReachesTheSourceAndPoolsIt(t *testing.T) {
	host, port, user, pw, dbName := srcEnv(t)
	a := newAlpha(t)
	tenant, prod := a.tenant(t, false)
	ds := a.datasource(t, tenant, prod, fmt.Sprintf(`{"host":%q,"port":%d,"database":%q,"username":%q,"password":%q}`, host, port, dbName, user, pw), true)
	c := realConnector(t, a, secrets.NewMemoryProvider(), time.Minute, 4)

	p, err := c.Pool(tctx(tenant), ds, OwnerOnly)
	require.NoError(t, err)
	var got string
	require.NoError(t, p.QueryRow(context.Background(), `SELECT current_database()`).Scan(&got))
	require.Equal(t, dbName, got)

	d1, err := c.SQLDB(tctx(tenant), ds, OwnerOnly)
	require.NoError(t, err)
	d2, err := c.SQLDB(tctx(tenant), ds, OwnerOnly)
	require.NoError(t, err)
	require.Same(t, d1, d2, "one shared view over one pool")
	require.Equal(t, 1, c.Size())
	require.NoError(t, d1.QueryRow(`SELECT 1`).Scan(new(int)))

	// Another tenant, same datasource id: refused, no pool.
	other, _ := a.tenant(t, false)
	_, err = c.Pool(tctx(other), ds, OwnerOnly)
	require.ErrorIs(t, err, ErrNotAllowed)
	require.Equal(t, 1, c.Size())
}

func TestConnector_ARotatedCredentialGetsANewPoolAndClosesTheOld(t *testing.T) {
	host, port, user, pw, dbName := srcEnv(t)
	a := newAlpha(t)
	tenant, prod := a.tenant(t, false)
	path := ""
	store := secrets.NewMemoryProvider()
	ds := a.datasource(t, tenant, prod, "{}", true)
	path, _ = dscreds.CanonicalPath(dscreds.KindDatasource, tenant, ds)
	setConfig := func(app string) {
		cfg := fmt.Sprintf(`{"host":%q,"port":%d,"database":%q,"secret_path":%q,"application_name":%q}`, host, port, dbName, path, app)
		tx, err := a.app.Begin()
		require.NoError(t, err)
		_, err = tx.Exec(`SELECT set_config('uisce.current_tenant', $1, true)`, tenant)
		require.NoError(t, err)
		_, err = tx.Exec(`UPDATE tenant_product_datasource SET config = $2::jsonb WHERE id = $1`, ds, cfg)
		require.NoError(t, err)
		require.NoError(t, tx.Commit())
	}
	require.NoError(t, store.PutMap(context.Background(), path, map[string]string{dscreds.KeyUsername: user, dscreds.KeyPassword: pw + "-v1"}))
	setConfig("v1")
	c := realConnector(t, a, store, 0, 4) // AuthTTL 0: every call re-checks, so a rotation is seen at once

	p1, err := c.Pool(tctx(tenant), ds, OwnerOnly)
	require.NoError(t, err)
	require.NoError(t, p1.Ping(context.Background()))

	// The credential is rotated in the secrets store: the pool key follows it.
	require.NoError(t, store.PutMap(context.Background(), path, map[string]string{dscreds.KeyUsername: user, dscreds.KeyPassword: pw + "-v2"}))
	p2, err := c.Pool(tctx(tenant), ds, OwnerOnly)
	require.NoError(t, err)
	require.NotSame(t, p1, p2, "a rotated credential must not be served by the old pool")
	require.Equal(t, 1, c.Size(), "the superseded pool is dropped, not left open beside the new one")
	require.Error(t, p1.Ping(context.Background()), "and its connections are closed")
	require.NoError(t, p2.Ping(context.Background()))
}

func TestConnector_BoundedAndCloseReleases(t *testing.T) {
	host, port, user, pw, dbName := srcEnv(t)
	a := newAlpha(t)
	tenant, prod := a.tenant(t, false)
	c := realConnector(t, a, secrets.NewMemoryProvider(), 0, 2)
	var ids []string
	for i := 0; i < 4; i++ {
		ids = append(ids, a.datasource(t, tenant, prod,
			fmt.Sprintf(`{"host":%q,"port":%d,"database":%q,"username":%q,"password":%q,"application_name":"src%d"}`, host, port, dbName, user, pw, i), true))
	}
	for _, id := range ids {
		_, err := c.Pool(tctx(tenant), id, OwnerOnly)
		require.NoError(t, err)
		require.LessOrEqual(t, c.Size(), 2, "the pool count must stay bounded")
	}
	c.Close()
	require.Zero(t, c.Size())
}

func TestConnector_ConcurrentUseIsSafeAndOnePoolPerSource(t *testing.T) {
	host, port, user, pw, dbName := srcEnv(t)
	a := newAlpha(t)
	tenant, prod := a.tenant(t, false)
	ds := a.datasource(t, tenant, prod, fmt.Sprintf(`{"host":%q,"port":%d,"database":%q,"username":%q,"password":%q}`, host, port, dbName, user, pw), true)
	c := realConnector(t, a, secrets.NewMemoryProvider(), 2*time.Second, 4)

	var wg sync.WaitGroup
	errs := make(chan error, 40)
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := c.Pool(tctx(tenant), ds, OwnerOnly)
			if err == nil {
				err = p.Ping(context.Background())
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, 1, c.Size(), "forty concurrent callers share one pool")
}

func TestConnector_SessionStateNeverLeaksBetweenUses(t *testing.T) {
	host, port, user, pw, dbName := srcEnv(t)
	a := newAlpha(t)
	tenant, prod := a.tenant(t, false)
	ds := a.datasource(t, tenant, prod, fmt.Sprintf(`{"host":%q,"port":%d,"database":%q,"username":%q,"password":%q}`, host, port, dbName, user, pw), true)
	c, err := New(Config{Registry: a.registry(), Credentials: DSCreds{R: dscreds.NewResolver(secrets.NewMemoryProvider(), dscreds.WithCacheTTL(0))},
		CallerTenant: uiscedb.GetTenantIDFromCtx, MaxPools: 2, MaxConnsPerPool: 1, IdleTTL: time.Minute, DialTimeout: 3 * time.Second})
	require.NoError(t, err)
	t.Cleanup(c.Close)

	// ONE connection per pool, so the second use is necessarily the first connection reused.
	db, err := c.SQLDB(tctx(tenant), ds, OwnerOnly)
	require.NoError(t, err)
	_, err = db.Exec(`SET application_name = 'leaked'`)
	require.NoError(t, err)
	var app string
	require.NoError(t, db.QueryRow(`SHOW application_name`).Scan(&app))
	require.NotContains(t, app, "leaked", "a connection with leftover session state must not be reused")
}
