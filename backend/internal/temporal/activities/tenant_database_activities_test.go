package activities_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.uber.org/zap"

	"github.com/hondyman/uisce/backend/internal/dscreds"
	"github.com/hondyman/uisce/backend/internal/migrations"
	"github.com/hondyman/uisce/backend/internal/provisioning"
	"github.com/hondyman/uisce/backend/internal/secrets"
	"github.com/hondyman/uisce/backend/internal/temporal/activities"
)

// These tests run the tenant-database saga steps against a real Postgres. Environment:
//
//	SAGA_TEST_ALPHA_ADMIN_DSN  superuser on a SCRATCH database standing in for alpha (its public
//	                           schema is dropped!); never the shared alpha database
//	SAGA_TEST_ALPHA_APP_DSN    the ordinary role the code runs as (no superuser, no BYPASSRLS),
//	                           a member of uisce_gold_copy_sync
//	SAGA_TEST_PG_HOST/PORT/USER  a superuser on the cluster that holds the tenant databases
//	SAGA_TEST_PG_PASSWORD        its password (required)
//
// The cluster must be a DEDICATED, hardened test cluster: provisioning now proves a tenant's role
// can connect to no other database, so every other database on the cluster, including postgres and
// template1, must have PUBLIC's CONNECT revoked (the scratch alpha is hardened by the rig itself).
// A shared dev cluster whose other databases are open will fail the probe, by design. A throwaway
// one is two commands: initdb -A trust, then REVOKE CONNECT ON DATABASE postgres, template1 FROM PUBLIC.
//
// Alpha's datasource chain is rebuilt here from the columns the code reads, with the REAL binding
// migrations and an equivalent isolation policy on the datasource table. That policy is a stub
// for production's, and the cluster here trusts all connections, so password rejection is not
// exercised.
type sagaRig struct {
	acts       *activities.TenantProvisioningActivities
	app, admin *sql.DB
	cluster    activities.TenantDatabaseAdmin
	sec        *secrets.MemoryProvider
	tenant     string
	instance   string
	dsOrm      string
	dsOther    string
	database   string
	gold       string
}

const sagaStub = `
CREATE FUNCTION uisce_get_current_tenant() RETURNS uuid AS $$
BEGIN
    RETURN NULLIF(current_setting('uisce.current_tenant', true), '')::uuid;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;
CREATE TABLE tenants (id uuid PRIMARY KEY, name text, code text, status text, allowed_regions jsonb, gold_copy boolean NOT NULL DEFAULT false);
CREATE TABLE tenant_instance (id uuid PRIMARY KEY, tenant_id uuid NOT NULL REFERENCES tenants(id), is_active bool NOT NULL DEFAULT true);
CREATE TABLE tenant_product (id uuid PRIMARY KEY, datasource_id uuid NOT NULL REFERENCES tenant_instance(id), is_active bool NOT NULL DEFAULT true);
CREATE TABLE alpha_datasource (id uuid PRIMARY KEY, datasource_code text NOT NULL);
CREATE TABLE tenant_product_datasource (
    id uuid PRIMARY KEY, tenant_product_id uuid NOT NULL REFERENCES tenant_product(id),
    alpha_datasource_id uuid NOT NULL REFERENCES alpha_datasource(id),
    is_active bool NOT NULL DEFAULT true, config jsonb NOT NULL DEFAULT '{}', core_id uuid);
`

const sagaPolicies = `
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
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO PUBLIC;
`

func newSagaRig(t *testing.T) *sagaRig { return newSagaRigOn(t, nil) }

// newSagaRigOn builds a rig, creating the tenant's database through the real activity.
func newSagaRigOn(t *testing.T, shared *sagaRig) *sagaRig { return newSagaRigOpts(t, shared, true) }

// newSagaRigOpts builds a rig. With shared == nil it resets the scratch alpha and starts from
// nothing. With a shared rig it adds ANOTHER tenant (own instance, datasources and database) to
// the same alpha and the same secrets store, which is what an isolation test needs. With
// createDB false the tenant's database is left for the caller to create (a concurrency test wants
// to drive CreateTenantDatabase itself); its cleanup still drops it.
func newSagaRigOpts(t *testing.T, shared *sagaRig, createDB bool) *sagaRig {
	t.Helper()
	adminDSN, appDSN := os.Getenv("SAGA_TEST_ALPHA_ADMIN_DSN"), os.Getenv("SAGA_TEST_ALPHA_APP_DSN")
	host, user := os.Getenv("SAGA_TEST_PG_HOST"), os.Getenv("SAGA_TEST_PG_USER")
	port, _ := strconv.Atoi(os.Getenv("SAGA_TEST_PG_PORT"))
	if adminDSN == "" || appDSN == "" || host == "" || user == "" || port == 0 {
		t.Skip("SAGA_TEST_* not set")
	}
	// No default: a rig that runs with whatever password it likes diverges from CI the day it is run against trust auth.
	// scripts/ci/realdb-local.sh is the supported way to get a rig, and it sets this.
	password := os.Getenv("SAGA_TEST_PG_PASSWORD")
	require.NotEmpty(t, password, "SAGA_TEST_PG_PASSWORD is required; use scripts/ci/realdb-local.sh")
	adm, err := sql.Open("pgx", adminDSN)
	require.NoError(t, err)
	t.Cleanup(func() { adm.Close() })
	var name string
	require.NoError(t, adm.QueryRow(`SELECT current_database()`).Scan(&name))
	require.Contains(t, name, "saga", "refusing to drop the public schema of %q", name)

	migDir := "../../../db/migrations/"
	b1, err := os.ReadFile(migDir + "20261206_001_tenant_datasource_binding.up.sql")
	require.NoError(t, err)
	b2, err := os.ReadFile(migDir + "20261210_001_binding_pg_credential.up.sql")
	require.NoError(t, err)
	b3, err := os.ReadFile(migDir + "20261217_001_structure_template_marker.up.sql")
	require.NoError(t, err)
	if shared == nil {
		for _, q := range []string{`DROP SCHEMA public CASCADE`, `CREATE SCHEMA public`, sagaStub, string(b1), string(b2), string(b3), sagaPolicies} {
			_, err := adm.Exec(q)
			require.NoError(t, err)
		}
	}
	app, err := sql.Open("pgx", appDSN)
	require.NoError(t, err)
	t.Cleanup(func() { app.Close() })
	var bypass bool
	require.NoError(t, app.QueryRow(`SELECT rolsuper OR rolbypassrls FROM pg_roles WHERE rolname = current_user`).Scan(&bypass))
	require.False(t, bypass, "the app role bypasses row-level security; the isolation tests would be vacuous")
	if shared == nil {
		// The scratch alpha stands in for the control plane. Like production's it must not be
		// connectable by tenant roles, so close PUBLIC's CONNECT and keep the app role's.
		var appUser string
		require.NoError(t, app.QueryRow(`SELECT current_user`).Scan(&appUser))
		_, err := adm.Exec(fmt.Sprintf(`REVOKE CONNECT ON DATABASE "%s" FROM PUBLIC; GRANT CONNECT ON DATABASE "%s" TO "%s"`, name, name, appUser))
		require.NoError(t, err)
	}

	r := &sagaRig{
		app: app, admin: adm, sec: secrets.NewMemoryProvider(),
		cluster: activities.TenantDatabaseAdmin{Host: host, Port: port, User: user, Password: password},
		tenant:  uuid.NewString(), instance: uuid.NewString(), dsOrm: uuid.NewString(), dsOther: uuid.NewString(),
		database: "tdb_saga_" + strings.ReplaceAll(uuid.NewString()[:8], "-", ""), gold: "gold_copy_db",
	}
	if shared != nil {
		r.sec = shared.sec
	}
	r.acts = &activities.TenantProvisioningActivities{
		ControlDB: sqlx.NewDb(app, "pgx"), Logger: zap.NewNop().Sugar(),
		Secrets: r.sec, TenantDB: r.cluster,
		Creds:      dscreds.NewResolver(r.sec, dscreds.WithCacheTTL(0)),
		Migrations: &migrations.TenantRunner{Root: t.TempDir()},
	}

	// The tenant's database, as the gold-copy clone leaves it: tables owned by the administrator.
	t.Setenv("DATABASE_URL", fmt.Sprintf("postgres://%s:%s@%s:%d/postgres?sslmode=disable", r.cluster.User, r.cluster.Password, r.cluster.Host, r.cluster.Port))
	if createDB {
		require.NoError(t, r.acts.CreateTenantDatabase(context.Background(), r.database), "the real activity creates the database and closes PUBLIC's CONNECT")
	}
	if createDB {
		tdb, err := r.cluster.Open(r.database)
		require.NoError(t, err)
		_, err = tdb.Exec(`CREATE TABLE orders (id serial PRIMARY KEY, note text)`)
		require.NoError(t, err)
		tdb.Close()
	}
	t.Cleanup(func() {
		c, err := r.cluster.Open("postgres")
		if err != nil {
			return
		}
		defer c.Close()
		_, _ = c.Exec(`DROP DATABASE IF EXISTS ` + r.database + ` WITH (FORCE)`)
		_, _ = c.Exec(`DROP ROLE IF EXISTS ` + r.database + `_app`)
	})

	r.seed(t)
	return r
}

// seed writes the tenant, its instance and two cloned datasources, both still naming the gold
// copy's database as CloneGoldCopyInstance leaves them.
func (r *sagaRig) seed(t *testing.T) {
	t.Helper()
	tx, err := r.app.Begin()
	require.NoError(t, err)
	_, err = tx.Exec(`SELECT set_config('uisce.current_tenant', $1, true)`, r.tenant)
	require.NoError(t, err)
	prod := uuid.NewString()
	odsOrm, odsOther := uuid.NewString(), uuid.NewString()
	cfg := fmt.Sprintf(`{"host":"gold-host","port":5432,"database":"%s"}`, r.gold)
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO tenants (id, name, code, status) VALUES ($1, 'n', $2, 'provisioning')`, []any{r.tenant, "c" + r.tenant[:8]}},
		{`INSERT INTO tenant_instance (id, tenant_id) VALUES ($1, $2)`, []any{r.instance, r.tenant}},
		{`INSERT INTO tenant_product (id, datasource_id) VALUES ($1, $2)`, []any{prod, r.instance}},
		{`INSERT INTO alpha_datasource (id, datasource_code) VALUES ($1, 'orm'), ($2, 'other')`, []any{odsOrm, odsOther}},
		{`INSERT INTO tenant_product_datasource (id, tenant_product_id, alpha_datasource_id, config) VALUES ($1, $2, $3, $4::jsonb)`, []any{r.dsOrm, prod, odsOrm, cfg}},
		{`INSERT INTO tenant_product_datasource (id, tenant_product_id, alpha_datasource_id, config) VALUES ($1, $2, $3, $4::jsonb)`, []any{r.dsOther, prod, odsOther, cfg}},
	} {
		_, err := tx.Exec(q.sql, q.args...)
		require.NoError(t, err, q.sql)
	}
	require.NoError(t, tx.Commit())
}

func (r *sagaRig) in() provisioning.TenantDatabaseInput {
	return provisioning.TenantDatabaseInput{
		TenantID: r.tenant, InstanceID: r.instance, App: "orm", DatabaseName: r.database, GoldCopyDatabase: r.gold,
	}
}

// asTenant runs a query in a transaction that carries the tenant's GUC, as the code under test does.
func (r *sagaRig) one(t *testing.T, tenant, q string, args ...any) []any {
	t.Helper()
	tx, err := r.app.Begin()
	require.NoError(t, err)
	defer tx.Rollback()
	_, err = tx.Exec(`SELECT set_config('uisce.current_tenant', $1, true)`, tenant)
	require.NoError(t, err)
	rows, err := tx.Query(q, args...)
	require.NoError(t, err)
	defer rows.Close()
	cols, _ := rows.Columns()
	if !rows.Next() {
		return nil
	}
	out := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range out {
		ptrs[i] = &out[i]
	}
	require.NoError(t, rows.Scan(ptrs...))
	return out
}

func (r *sagaRig) bound(t *testing.T) provisioning.TenantDatabaseInput {
	t.Helper()
	b, err := r.acts.BindTenantDatabase(context.Background(), r.in())
	require.NoError(t, err)
	in := r.in()
	in.DatasourceID = b.DatasourceID
	return in
}

func (r *sagaRig) provisioned(t *testing.T) provisioning.TenantDatabaseInput {
	t.Helper()
	in := r.bound(t)
	require.NoError(t, r.acts.ProvisionTenantDatabaseAccess(context.Background(), in))
	return in
}

func (r *sagaRig) roleExists(t *testing.T) bool {
	t.Helper()
	var n int
	require.NoError(t, r.admin.QueryRow(`SELECT count(*) FROM pg_roles WHERE rolname = $1`, r.database+"_app").Scan(&n))
	return n == 1
}

func (r *sagaRig) asRole(t *testing.T, password string) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", fmt.Sprintf("postgres://%s_app:%s@%s:%d/%s", r.database, password, r.cluster.Host, r.cluster.Port, r.database))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func isNonRetryableOf(err error, typ string) bool {
	var app *sdktemporal.ApplicationError
	return errors.As(err, &app) && app.NonRetryable() && (typ == "" || app.Type() == typ)
}

func TestTenantDatabase_BindRepointsOnlyTheAppsDatasource(t *testing.T) {
	r := newSagaRig(t)
	in := r.bound(t)
	require.Equal(t, r.dsOrm, in.DatasourceID)

	row := r.one(t, r.tenant, `SELECT config->>'host', (config->>'port')::int, config->>'database', config->>'secret_path'
		FROM tenant_product_datasource WHERE id = $1`, r.dsOrm)
	require.Equal(t, r.cluster.Host, row[0])
	require.EqualValues(t, r.cluster.Port, row[1])
	require.Equal(t, r.database, row[2])
	want, _ := dscreds.CanonicalPath(dscreds.KindDatasource, r.tenant, r.dsOrm)
	require.Equal(t, want, row[3])

	// The other cloned datasource named the gold copy's database; it must not resolve to it.
	other := r.one(t, r.tenant, `SELECT config ? 'database' FROM tenant_product_datasource WHERE id = $1`, r.dsOther)
	require.Equal(t, false, other[0], "a cloned datasource must not keep naming the gold copy's database")

	b := r.one(t, r.tenant, `SELECT pg_role, lifecycle_state, pg_credential_issued_at IS NULL FROM tenant_datasource_binding WHERE datasource_id = $1`, r.dsOrm)
	require.Equal(t, []any{r.database + "_app", "provisioning", true}, b)

	// Running it again changes nothing.
	_, err := r.acts.BindTenantDatabase(context.Background(), r.in())
	require.NoError(t, err)
}

func TestTenantDatabase_BindRefusesAmbiguousOrForeignInput(t *testing.T) {
	r := newSagaRig(t)
	ctx := context.Background()

	in := r.in()
	in.App = "nosuchapp"
	_, err := r.acts.BindTenantDatabase(ctx, in)
	require.Error(t, err)
	require.True(t, isNonRetryableOf(err, "TenantDatabaseInvalidInput"), "%v", err)

	// Another tenant's instance finds nothing: row-level security hides it.
	in = r.in()
	in.TenantID = uuid.NewString()
	_, err = r.acts.BindTenantDatabase(ctx, in)
	require.Error(t, err)

	for _, bad := range []string{"Orm", "../orm", "orm; DROP", ""} {
		in = r.in()
		in.App = bad
		_, err = r.acts.BindTenantDatabase(ctx, in)
		require.Error(t, err, bad)
	}
	for _, bad := range []string{"x; DROP DATABASE postgres", "UPPER", "a b", strings.Repeat("a", 70)} {
		in = r.in()
		in.DatabaseName = bad
		_, err = r.acts.BindTenantDatabase(ctx, in)
		require.Error(t, err, bad)
	}

	unconfigured := *r.acts
	unconfigured.TenantDB = activities.TenantDatabaseAdmin{}
	_, err = (&unconfigured).BindTenantDatabase(ctx, r.in())
	require.True(t, isNonRetryableOf(err, "TenantDatabaseNotConfigured"), "%v", err)
}

func TestTenantDatabase_ProvisionGivesARoleThatCanUseDataButNotChangeSchema(t *testing.T) {
	r := newSagaRig(t)
	in := r.provisioned(t)

	path, _ := dscreds.CanonicalPath(dscreds.KindDatasource, r.tenant, in.DatasourceID)
	stored, err := r.sec.GetMap(context.Background(), path)
	require.NoError(t, err)
	require.Equal(t, r.database+"_app", stored[dscreds.KeyUsername])
	require.NotEmpty(t, stored[dscreds.KeyPassword])
	require.True(t, r.roleExists(t))

	b := r.one(t, r.tenant, `SELECT pg_credential_issued_at IS NOT NULL FROM tenant_datasource_binding WHERE datasource_id = $1`, in.DatasourceID)
	require.Equal(t, true, b[0])

	db := r.asRole(t, stored[dscreds.KeyPassword])
	_, err = db.Exec(`INSERT INTO orders (note) VALUES ('hello')`)
	require.NoError(t, err, "the role must be able to write the tenant's tables and use its sequences")
	var n int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM orders`).Scan(&n))
	require.Equal(t, 1, n)

	_, err = db.Exec(`CREATE TABLE sneaky (id int)`)
	require.Error(t, err, "the app role must not get DDL")
	_, err = db.Exec(`DROP TABLE orders`)
	require.Error(t, err)
	var super bool
	require.NoError(t, db.QueryRow(`SELECT rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls FROM pg_roles WHERE rolname = current_user`).Scan(&super))
	require.False(t, super)

	// A table a later migration creates (as the administrator) is usable too.
	adm, err := r.cluster.Open(r.database)
	require.NoError(t, err)
	defer adm.Close()
	_, err = adm.Exec(`CREATE TABLE later (id serial PRIMARY KEY)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO later DEFAULT VALUES`)
	require.NoError(t, err, "default privileges must cover tables created after provisioning")
}

func TestTenantDatabase_ProvisionNeverReplacesAnIssuedCredential(t *testing.T) {
	r := newSagaRig(t)
	ctx := context.Background()
	in := r.provisioned(t)
	path, _ := dscreds.CanonicalPath(dscreds.KindDatasource, r.tenant, in.DatasourceID)
	first, _ := r.sec.GetMap(ctx, path)

	require.NoError(t, r.acts.ProvisionTenantDatabaseAccess(ctx, in), "re-running must be safe")
	again, _ := r.sec.GetMap(ctx, path)
	require.Equal(t, first[dscreds.KeyPassword], again[dscreds.KeyPassword], "a re-run must not rotate the credential")

	// The secrets store loses it (it reports every failure as "not found"): issued, but gone.
	require.NoError(t, r.sec.Delete(ctx, path))
	err := r.acts.ProvisionTenantDatabaseAccess(ctx, in)
	require.Error(t, err)
	require.True(t, isNonRetryableOf(err, "TenantDatabaseCredentialLost"), "%v", err)
	_, gerr := r.sec.GetMap(ctx, path)
	require.ErrorIs(t, gerr, secrets.ErrSecretNotFound, "a lost credential must never be silently replaced")
}

func TestTenantDatabase_ProvisionResumesACrashBetweenStoringAndCreating(t *testing.T) {
	r := newSagaRig(t)
	ctx := context.Background()
	in := r.bound(t)
	path, _ := dscreds.CanonicalPath(dscreds.KindDatasource, r.tenant, in.DatasourceID)

	// The previous attempt stored the credential and died before creating the role.
	require.NoError(t, r.sec.PutMap(ctx, path, map[string]string{dscreds.KeyUsername: r.database + "_app", dscreds.KeyPassword: "stored-earlier"}))
	require.False(t, r.roleExists(t))

	require.NoError(t, r.acts.ProvisionTenantDatabaseAccess(ctx, in))
	got, _ := r.sec.GetMap(ctx, path)
	require.Equal(t, "stored-earlier", got[dscreds.KeyPassword], "the stored password is reused, not replaced")
	require.True(t, r.roleExists(t))
}

func TestTenantDatabase_ProvisionRefusesABindingForADifferentRole(t *testing.T) {
	r := newSagaRig(t)
	in := r.bound(t)
	_, err := r.admin.Exec(`UPDATE tenant_datasource_binding SET pg_role = 'someone_else' WHERE datasource_id = $1`, in.DatasourceID)
	require.NoError(t, err)
	err = r.acts.ProvisionTenantDatabaseAccess(context.Background(), in)
	require.True(t, isNonRetryableOf(err, "TenantDatabaseInvalidInput"), "%v", err)
	require.False(t, r.roleExists(t))
}

func TestTenantDatabase_ApplyAppliesTheAppsMigrationsAndReports(t *testing.T) {
	r := newSagaRig(t)
	ctx := context.Background()
	dir := filepath.Join(r.acts.Migrations.Root, "orm")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "0001_a.up.sql"), []byte(`CREATE TABLE already_cloned (id int);`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "0002_b.up.sql"), []byte(`CREATE TABLE added_later (id int);`), 0o644))

	in := r.in()
	in.BaselineThrough = "0001_a.up.sql" // the clone already had 0001; it must not run
	rep, err := r.acts.ApplyTenantMigrations(ctx, in)
	require.NoError(t, err)
	require.True(t, rep.Done)
	require.Equal(t, []string{"0002_b.up.sql"}, rep.Ran)
	require.Len(t, rep.Applied, 2)

	tdb, _ := r.cluster.Open(r.database)
	defer tdb.Close()
	var has bool
	require.NoError(t, tdb.QueryRow(`SELECT to_regclass('already_cloned') IS NOT NULL`).Scan(&has))
	require.False(t, has, "a baseline file must be recorded, not executed")
	require.NoError(t, tdb.QueryRow(`SELECT to_regclass('added_later') IS NOT NULL`).Scan(&has))
	require.True(t, has)

	// A rerun with the same baseline is fine: nothing to do.
	rep, err = r.acts.ApplyTenantMigrations(ctx, in)
	require.NoError(t, err)
	require.True(t, rep.Done)
	require.Empty(t, rep.Ran)
}

func TestTenantDatabase_ApplyFailsClosedOnDriftAndMissingApp(t *testing.T) {
	r := newSagaRig(t)
	ctx := context.Background()

	_, err := r.acts.ApplyTenantMigrations(ctx, r.in()) // no orm directory
	require.Error(t, err)
	require.True(t, isNonRetryableOf(err, "TenantDatabaseMigrationFailed"), "%v", err)

	dir := filepath.Join(r.acts.Migrations.Root, "orm")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	f := filepath.Join(dir, "0001_a.up.sql")
	require.NoError(t, os.WriteFile(f, []byte(`CREATE TABLE a (id int);`), 0o644))
	_, err = r.acts.ApplyTenantMigrations(ctx, r.in())
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(f, []byte(`CREATE TABLE a (id int, sneaky int);`), 0o644))
	_, err = r.acts.ApplyTenantMigrations(ctx, r.in())
	require.Error(t, err)
	require.True(t, isNonRetryableOf(err, "TenantDatabaseMigrationFailed"), "%v", err)
}

func TestTenantDatabase_ProbeConnectsLikeProductionAndRefusesAnotherTenant(t *testing.T) {
	r := newSagaRig(t)
	ctx := context.Background()
	in := r.bound(t)

	require.Error(t, r.acts.ProbeTenantDatabase(ctx, in), "before the role exists the probe must fail")

	require.NoError(t, r.acts.ProvisionTenantDatabaseAccess(ctx, in))
	require.NoError(t, r.acts.ProbeTenantDatabase(ctx, in), "the probe goes through tenantdb with the registered credential")

	other := in
	other.TenantID = uuid.NewString()
	require.Error(t, r.acts.ProbeTenantDatabase(ctx, other), "another tenant must not be able to probe this datasource")
}

func TestTenantDatabase_ActivateOnlyMovesAProvisioningBinding(t *testing.T) {
	r := newSagaRig(t)
	ctx := context.Background()
	in := r.provisioned(t)

	require.NoError(t, r.acts.ActivateTenantDatabase(ctx, in))
	b := r.one(t, r.tenant, `SELECT lifecycle_state, version FROM tenant_datasource_binding WHERE datasource_id = $1`, in.DatasourceID)
	require.EqualValues(t, "active", b[0])
	require.EqualValues(t, 2, b[1])

	require.NoError(t, r.acts.ActivateTenantDatabase(ctx, in), "idempotent")
	b = r.one(t, r.tenant, `SELECT version FROM tenant_datasource_binding WHERE datasource_id = $1`, in.DatasourceID)
	require.EqualValues(t, 2, b[0], "a second activation must not bump the version")

	_, err := r.admin.Exec(`UPDATE tenant_datasource_binding SET lifecycle_state = 'suspended' WHERE datasource_id = $1`, in.DatasourceID)
	require.NoError(t, err)
	require.NoError(t, r.acts.ActivateTenantDatabase(ctx, in))
	b = r.one(t, r.tenant, `SELECT lifecycle_state FROM tenant_datasource_binding WHERE datasource_id = $1`, in.DatasourceID)
	require.EqualValues(t, "suspended", b[0], "activation must not resurrect a suspended binding")
}

func TestTenantDatabase_RollbackRemovesWhatTheRunCreated(t *testing.T) {
	r := newSagaRig(t)
	ctx := context.Background()
	in := r.provisioned(t)
	path, _ := dscreds.CanonicalPath(dscreds.KindDatasource, r.tenant, in.DatasourceID)
	require.True(t, r.roleExists(t))

	require.NoError(t, r.acts.RollbackTenantDatabase(ctx, in))
	require.False(t, r.roleExists(t), "the role must be dropped")
	_, err := r.sec.GetMap(ctx, path)
	require.ErrorIs(t, err, secrets.ErrSecretNotFound, "the credential must be deleted")
	require.Nil(t, r.one(t, r.tenant, `SELECT 1 FROM tenant_datasource_binding WHERE datasource_id = $1`, in.DatasourceID))

	require.NoError(t, r.acts.RollbackTenantDatabase(ctx, in), "a second rollback is a no-op")
	require.NoError(t, r.acts.RollbackTenantDatabase(ctx, provisioning.TenantDatabaseInput{TenantID: r.tenant}),
		"nothing to undo when binding never completed")
}

func TestTenantDatabase_RollbackNeverTouchesALiveBinding(t *testing.T) {
	r := newSagaRig(t)
	ctx := context.Background()
	in := r.provisioned(t)
	require.NoError(t, r.acts.ActivateTenantDatabase(ctx, in))
	path, _ := dscreds.CanonicalPath(dscreds.KindDatasource, r.tenant, in.DatasourceID)

	require.NoError(t, r.acts.RollbackTenantDatabase(ctx, in))
	require.True(t, r.roleExists(t), "an active tenant's role must survive a rollback")
	_, err := r.sec.GetMap(ctx, path)
	require.NoError(t, err)
	require.NotNil(t, r.one(t, r.tenant, `SELECT 1 FROM tenant_datasource_binding WHERE datasource_id = $1`, in.DatasourceID))
}

func TestTenantDatabase_AdminFromEnvHasNoDefaults(t *testing.T) {
	for _, k := range []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_PASS"} {
		t.Setenv(k, "")
	}
	_, err := activities.TenantDatabaseAdminFromEnv()
	require.ErrorIs(t, err, activities.ErrTenantDatabaseNotConfigured)
	t.Setenv("DB_HOST", "h")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "u")
	_, err = activities.TenantDatabaseAdminFromEnv()
	require.ErrorIs(t, err, activities.ErrTenantDatabaseNotConfigured, "a missing password must not be defaulted")
	t.Setenv("DB_PASS", "p")
	a, err := activities.TenantDatabaseAdminFromEnv()
	require.NoError(t, err)
	require.Equal(t, 5432, a.Port)
}

func TestTenantDatabaseRole(t *testing.T) {
	r, err := activities.TenantDatabaseRole("orm_acme")
	require.NoError(t, err)
	require.Equal(t, "orm_acme_app", r)
	for _, bad := range []string{"", "Orm", "1orm", "orm-acme", "orm acme", "orm;drop", strings.Repeat("a", 60)} {
		_, err := activities.TenantDatabaseRole(bad)
		require.Error(t, err, bad)
	}
}
