package tenantdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/dscreds"
	"github.com/hondyman/uisce/backend/internal/security"
)

// These tests run AlphaRegistry against a real Postgres whose schema this test builds itself:
// the datasource chain tables the resolver reads (reduced to the columns it uses), the REAL
// 20261206_001 binding migration, and strict row-level security keyed on uisce.current_tenant.
//
// What they prove: ownership comes from the resolver, the row and binding are read inside the
// OWNER's tenant transaction, and a row that belongs to another tenant is never returned.
// What they do not prove: the production RLS policies on tenant_product_datasource, which this
// stub replaces with an equivalent one. A DSN against a copy of real alpha would close that.
//
//	TENANTDB_ALPHA_ADMIN_DSN  superuser on a SCRATCH database (its public schema is dropped!)
//	TENANTDB_ALPHA_APP_DSN    the ordinary role the code runs as (no superuser, no BYPASSRLS)
//
// The app role must be a member of uisce_gold_copy_sync (the resolver switches to it).
func alphaDBs(t *testing.T) *sql.DB {
	t.Helper()
	admin, app := os.Getenv("TENANTDB_ALPHA_ADMIN_DSN"), os.Getenv("TENANTDB_ALPHA_APP_DSN")
	if admin == "" || app == "" {
		t.Skip("TENANTDB_ALPHA_*_DSN not set")
	}
	adm, err := sql.Open("pgx", admin)
	require.NoError(t, err)
	t.Cleanup(func() { adm.Close() })

	var name string
	require.NoError(t, adm.QueryRow(`SELECT current_database()`).Scan(&name))
	require.Contains(t, name, "alpha", "refusing to drop the public schema of %q", name)
	require.NotEqual(t, "alpha", name, "refusing to run against the shared alpha database")

	migration, err := os.ReadFile("../../db/migrations/20261206_001_tenant_datasource_binding.up.sql")
	require.NoError(t, err)
	for _, stmt := range []string{`DROP SCHEMA public CASCADE`, `CREATE SCHEMA public`, alphaSchema, string(migration), alphaPolicies} {
		_, err := adm.Exec(stmt)
		require.NoError(t, err)
	}

	db, err := sql.Open("pgx", app)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	var bypass bool
	require.NoError(t, db.QueryRow(`SELECT rolsuper OR rolbypassrls FROM pg_roles WHERE rolname = current_user`).Scan(&bypass))
	require.False(t, bypass, "the app role bypasses row-level security; the isolation test would be vacuous")
	return db
}

const alphaSchema = `
CREATE FUNCTION uisce_get_current_tenant() RETURNS uuid AS $$
BEGIN
    RETURN NULLIF(current_setting('uisce.current_tenant', true), '')::uuid;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

CREATE TABLE tenants (id uuid PRIMARY KEY, name text, code text, allowed_regions jsonb);
CREATE TABLE tenant_instance (id uuid PRIMARY KEY, tenant_id uuid NOT NULL REFERENCES tenants(id), is_active bool NOT NULL DEFAULT true);
CREATE TABLE tenant_product (id uuid PRIMARY KEY, datasource_id uuid NOT NULL REFERENCES tenant_instance(id), is_active bool NOT NULL DEFAULT true);
CREATE TABLE tenant_product_datasource (
    id uuid PRIMARY KEY, tenant_product_id uuid NOT NULL REFERENCES tenant_product(id),
    is_active bool NOT NULL DEFAULT true, config jsonb NOT NULL DEFAULT '{}');
`

const alphaPolicies = `
-- A datasource row is visible only to the tenant that owns it (through its instance).
ALTER TABLE tenant_product_datasource ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_product_datasource FORCE ROW LEVEL SECURITY;
CREATE POLICY tpd_isolation ON tenant_product_datasource FOR ALL USING (
    EXISTS (SELECT 1 FROM tenant_product tp JOIN tenant_instance ti ON ti.id = tp.datasource_id
            WHERE tp.id = tenant_product_datasource.tenant_product_id
              AND ti.tenant_id = uisce_get_current_tenant()));

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'uisce_gold_copy_sync') THEN
        CREATE ROLE uisce_gold_copy_sync NOLOGIN BYPASSRLS;
    END IF;
END $$;
GRANT USAGE ON SCHEMA public TO PUBLIC;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO PUBLIC;
`

type seeded struct{ t1, t2, ds1, ds2 string }

func seed(t *testing.T, db *sql.DB) seeded {
	t.Helper()
	// Seeding is done as the app role's own admin path would: through a superuser-free
	// connection is impossible under FORCE RLS, so insert into the unprotected tables first and
	// the datasource rows with the owner's GUC set.
	s := seeded{t1: uuid.NewString(), t2: uuid.NewString(), ds1: uuid.NewString(), ds2: uuid.NewString()}
	for i, tenant := range []string{s.t1, s.t2} {
		inst, prod, ds := uuid.NewString(), uuid.NewString(), []string{s.ds1, s.ds2}[i]
		tx, err := db.Begin()
		require.NoError(t, err)
		_, err = tx.Exec(`SELECT set_config('uisce.current_tenant', $1, true)`, tenant)
		require.NoError(t, err)
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO tenants (id, name, code) VALUES ($1, 'n', $2)`, []any{tenant, "c" + tenant[:8]}},
			{`INSERT INTO tenant_instance (id, tenant_id) VALUES ($1, $2)`, []any{inst, tenant}},
			{`INSERT INTO tenant_product (id, datasource_id) VALUES ($1, $2)`, []any{prod, inst}},
			{`INSERT INTO tenant_product_datasource (id, tenant_product_id, config) VALUES ($1, $2, $3)`,
				[]any{ds, prod, fmt.Sprintf(`{"host":"db%d","port":5432,"database":"orm_t%d","username":"app%d","password":"pw%d"}`, i+1, i+1, i+1, i+1)}},
		} {
			_, err := tx.Exec(q.sql, q.args...)
			require.NoError(t, err, q.sql)
		}
		require.NoError(t, tx.Commit())
	}
	return s
}

func bind(t *testing.T, db *sql.DB, bindingTenant, ds, state string) {
	t.Helper()
	tx, err := db.Begin()
	require.NoError(t, err)
	_, err = tx.Exec(`SELECT set_config('uisce.current_tenant', $1, true)`, bindingTenant)
	require.NoError(t, err)
	_, err = tx.Exec(`INSERT INTO tenant_datasource_binding (datasource_id, tenant_id, lifecycle_state, version) VALUES ($1, $2, $3, 3)`,
		ds, bindingTenant, state)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
}

func registry(db *sql.DB) *AlphaRegistry {
	return &AlphaRegistry{
		DB:       db,
		Resolver: security.NewDBDatasourceResolver(sqlx.NewDb(db, "pgx")),
		Creds:    dscreds.NewResolver(nil, dscreds.WithCacheTTL(0)),
	}
}

func TestAlphaRegistry_ReadsOwnRowAndBinding(t *testing.T) {
	db := alphaDBs(t)
	s := seed(t, db)
	bind(t, db, s.t1, s.ds1, "active")
	r := registry(db)
	ctx := context.Background()

	ds, err := r.ResolveDatasource(ctx, s.ds1)
	require.NoError(t, err)
	require.Equal(t, s.t1, ds.TenantID)
	require.Equal(t, "orm_t1", ds.Database)
	require.Equal(t, "db1", ds.Host)

	b, err := r.LoadBinding(ctx, s.ds1)
	require.NoError(t, err)
	require.Equal(t, Binding{Version: 3, Lifecycle: "active"}, b)

	u, p, err := r.Credentials(ctx, ds)
	require.NoError(t, err)
	require.Equal(t, [2]string{"app1", "pw1"}, [2]string{u, p})
}

// The failure mode this whole design exists to prevent: tenant A's caller must never be handed
// tenant B's database, binding or credentials.
func TestAlphaRegistry_NeverReturnsAnotherTenantsRows(t *testing.T) {
	db := alphaDBs(t)
	s := seed(t, db)
	bind(t, db, s.t1, s.ds1, "active")
	bind(t, db, s.t2, s.ds2, "active")
	r := registry(db)
	ctx := context.Background()

	// Resolved datasources carry THEIR owner, so the router's tenant check can refuse the caller.
	ds2, err := r.ResolveDatasource(ctx, s.ds2)
	require.NoError(t, err)
	require.Equal(t, s.t2, ds2.TenantID)
	require.Equal(t, "orm_t2", ds2.Database)

	router, err := New(Config{Registry: r, CallerTenant: func(context.Context) (string, error) { return s.t1, nil },
		MaxPools: 1, MaxConnsPerPool: 1, IdleTTL: time.Minute, DialTimeout: time.Second})
	require.NoError(t, err)
	defer router.Close()
	p, err := router.Resolve(ctx, s.ds2)
	require.ErrorIs(t, err, ErrTenantMismatch)
	require.Nil(t, p)
	require.Zero(t, router.Size())
}

// A binding row filed under the wrong tenant must not authorize its datasource: it is read in
// the owner's transaction, where row-level security hides it.
func TestAlphaRegistry_ABindingFiledUnderTheWrongTenantIsInvisible(t *testing.T) {
	db := alphaDBs(t)
	s := seed(t, db)
	bind(t, db, s.t2, s.ds1, "active") // ds1 belongs to t1; the binding claims t2
	_, err := registry(db).LoadBinding(context.Background(), s.ds1)
	require.ErrorIs(t, err, ErrUnbound)
}

func TestAlphaRegistry_RefusalsFailClosed(t *testing.T) {
	db := alphaDBs(t)
	s := seed(t, db)
	r := registry(db)
	ctx := context.Background()

	_, err := r.LoadBinding(ctx, s.ds1)
	require.ErrorIs(t, err, ErrUnbound, "a datasource with no binding is unbound, not active")

	for _, id := range []string{uuid.NewString(), "not-a-uuid", ""} {
		_, err := r.ResolveDatasource(ctx, id)
		require.Error(t, err, id)
		_, err = r.LoadBinding(ctx, id)
		require.Error(t, err, id)
	}

}
