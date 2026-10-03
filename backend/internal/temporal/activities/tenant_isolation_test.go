package activities_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	uiscedb "github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/dscreds"
	"github.com/hondyman/uisce/backend/internal/migrations"
	"github.com/hondyman/uisce/backend/internal/platform"
	"github.com/hondyman/uisce/backend/internal/provisioning"
	"github.com/hondyman/uisce/backend/internal/secrets"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/tenantdb"
)

// Isolation across the whole path: two tenants are provisioned through the real saga activities
// (bind, role and credential, migrations, probe, activate), then reached through the real tenantdb
// router with the real AlphaRegistry. Every route from one tenant to the other's database,
// credential or binding is attempted and must fail closed.
//
// Same environment as the saga activity tests (SAGA_TEST_*). The alpha here is a scratch schema
// with the real binding migrations and a stub datasource policy, and the test cluster trusts every
// connection, so what is proven is the PRIVILEGE boundary (CONNECT on each database), the router's
// checks and the registry's, not password rejection.

type tenantUnderTest struct {
	rig *sagaRig
	in  provisioning.TenantDatabaseInput
}

const isoMigration = `CREATE TABLE notes (id serial PRIMARY KEY, owner text NOT NULL, body text NOT NULL);`

func provisionFully(t *testing.T, r *sagaRig) tenantUnderTest {
	t.Helper()
	ctx := context.Background()
	dir := filepath.Join(r.acts.Migrations.Root, "orm")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "0001_notes.up.sql"), []byte(isoMigration), 0o644))

	in := r.bound(t)
	require.NoError(t, r.acts.ProvisionTenantDatabaseAccess(ctx, in))
	rep, err := r.acts.ApplyTenantMigrations(ctx, in)
	require.NoError(t, err)
	require.True(t, rep.Done)
	require.NoError(t, r.acts.ProbeTenantDatabase(ctx, in))
	require.NoError(t, r.acts.ActivateTenantDatabase(ctx, in))
	return tenantUnderTest{rig: r, in: in}
}

func routerFor(t *testing.T, r *sagaRig) *tenantdb.Router {
	t.Helper()
	router, err := tenantdb.New(tenantdb.Config{
		Registry: &tenantdb.AlphaRegistry{
			DB: r.app, Resolver: security.NewDBDatasourceResolver(sqlx.NewDb(r.app, "pgx")), Creds: r.acts.Creds,
		},
		CallerTenant: uiscedb.GetTenantIDFromCtx,
		MaxPools:     8, MaxConnsPerPool: 2, IdleTTL: time.Minute, DialTimeout: 5 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(router.Close)
	return router
}

func as(tenant string) context.Context {
	return uiscedb.WithTenantContextToCtx(context.Background(), tenant)
}

type isoRig struct {
	a, b   tenantUnderTest
	router *tenantdb.Router
}

func newIsoRig(t *testing.T) *isoRig {
	t.Helper()
	ra := newSagaRig(t)
	rb := newSagaRigOn(t, ra)
	a, b := provisionFully(t, ra), provisionFully(t, rb)
	return &isoRig{a: a, b: b, router: routerFor(t, ra)}
}

func (u tenantUnderTest) addNote(t *testing.T, router *tenantdb.Router, body string) {
	t.Helper()
	p, err := router.Resolve(as(u.in.TenantID), u.in.DatasourceID)
	require.NoError(t, err)
	require.NoError(t, p.WithTx(as(u.in.TenantID), func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), `INSERT INTO notes (owner, body) VALUES ($1, $2)`, u.in.TenantID, body)
		return err
	}))
}

func (u tenantUnderTest) notes(t *testing.T, router *tenantdb.Router) (db string, bodies []string) {
	t.Helper()
	p, err := router.Resolve(as(u.in.TenantID), u.in.DatasourceID)
	require.NoError(t, err)
	require.NoError(t, p.WithTx(as(u.in.TenantID), func(tx pgx.Tx) error {
		if err := tx.QueryRow(context.Background(), `SELECT current_database()`).Scan(&db); err != nil {
			return err
		}
		rows, err := tx.Query(context.Background(), `SELECT body FROM notes ORDER BY id`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var b string
			if err := rows.Scan(&b); err != nil {
				return err
			}
			bodies = append(bodies, b)
		}
		return rows.Err()
	}))
	return db, bodies
}

func TestIsolation_EachTenantReachesOnlyItsOwnDatabaseAndData(t *testing.T) {
	g := newIsoRig(t)
	g.a.addNote(t, g.router, "a-secret")
	g.b.addNote(t, g.router, "b-secret")

	dbA, notesA := g.a.notes(t, g.router)
	dbB, notesB := g.b.notes(t, g.router)
	require.Equal(t, g.a.rig.database, dbA)
	require.Equal(t, g.b.rig.database, dbB)
	require.NotEqual(t, dbA, dbB)
	require.Equal(t, []string{"a-secret"}, notesA, "tenant A must see only its own rows")
	require.Equal(t, []string{"b-secret"}, notesB, "tenant B must see only its own rows")
}

func TestIsolation_ATenantCannotResolveAnotherTenantsDatasource(t *testing.T) {
	g := newIsoRig(t)
	for name, c := range map[string]struct {
		caller string
		target tenantUnderTest
	}{"A reaching B": {g.a.in.TenantID, g.b}, "B reaching A": {g.b.in.TenantID, g.a}} {
		p, err := g.router.Resolve(as(c.caller), c.target.in.DatasourceID)
		require.ErrorIs(t, err, tenantdb.ErrTenantMismatch, name)
		require.Nil(t, p, name)
	}
	require.Zero(t, g.router.Size(), "a refused resolve must not leave a pool behind")

	_, err := g.router.Resolve(context.Background(), g.a.in.DatasourceID)
	require.ErrorIs(t, err, tenantdb.ErrNoTenant, "no tenant in the context is never a default")
	_, err = g.router.Resolve(as(""), g.a.in.DatasourceID)
	require.ErrorIs(t, err, tenantdb.ErrNoTenant)
}

// The database privilege is the last line: even holding tenant A's role and password, B's database
// refuses the connection, and B's tables are unreachable.
func TestIsolation_ATenantsRoleCannotConnectToAnotherTenantsDatabase(t *testing.T) {
	g := newIsoRig(t)
	ctx := context.Background()
	path, _ := dscreds.CanonicalPath(dscreds.KindDatasource, g.a.in.TenantID, g.a.in.DatasourceID)
	creds, err := g.a.rig.sec.GetMap(ctx, path)
	require.NoError(t, err)

	own, err := pgx.Connect(ctx, fmt.Sprintf("postgres://%s:%s@%s:%d/%s", creds[dscreds.KeyUsername], creds[dscreds.KeyPassword],
		g.a.rig.cluster.Host, g.a.rig.cluster.Port, g.a.rig.database))
	require.NoError(t, err, "control: the role must reach its own database")
	own.Close(ctx)

	for _, db := range []string{g.b.rig.database, "postgres", "template1"} {
		c, err := pgx.Connect(ctx, fmt.Sprintf("postgres://%s:%s@%s:%d/%s", creds[dscreds.KeyUsername], creds[dscreds.KeyPassword],
			g.a.rig.cluster.Host, g.a.rig.cluster.Port, db))
		if err == nil {
			c.Close(ctx)
		}
		require.Error(t, err, "tenant A's role must not be able to connect to %q", db)
		require.Contains(t, err.Error(), "permission denied", db)
	}
}

// An operator mistake or a compromised registry row must still fail closed.
func TestIsolation_AMiswiredRegistryFailsClosed(t *testing.T) {
	ctx := context.Background()

	t.Run("A's datasource repointed at B's database", func(t *testing.T) {
		g := newIsoRig(t)
		_, err := g.a.rig.admin.Exec(`UPDATE tenant_product_datasource SET config = config || jsonb_build_object('database', $2::text) WHERE id = $1`,
			g.a.in.DatasourceID, g.b.rig.database)
		require.NoError(t, err)
		p, err := g.router.Resolve(as(g.a.in.TenantID), g.a.in.DatasourceID)
		require.Error(t, err, "A's credential must not open B's database")
		require.Nil(t, p)
		require.Zero(t, g.router.Size())
	})

	t.Run("A's datasource points at B's secret path", func(t *testing.T) {
		g := newIsoRig(t)
		bPath, _ := dscreds.CanonicalPath(dscreds.KindDatasource, g.b.in.TenantID, g.b.in.DatasourceID)
		_, err := g.a.rig.admin.Exec(`UPDATE tenant_product_datasource SET config = config || jsonb_build_object('secret_path', $2::text) WHERE id = $1`,
			g.a.in.DatasourceID, bPath)
		require.NoError(t, err)
		_, err = g.router.Resolve(as(g.a.in.TenantID), g.a.in.DatasourceID)
		require.ErrorIs(t, err, dscreds.ErrRefMismatch, "a datasource may only read its own canonical secret path")
	})

	t.Run("B's credential filed under A's secret", func(t *testing.T) {
		g := newIsoRig(t)
		aPath, _ := dscreds.CanonicalPath(dscreds.KindDatasource, g.a.in.TenantID, g.a.in.DatasourceID)
		bPath, _ := dscreds.CanonicalPath(dscreds.KindDatasource, g.b.in.TenantID, g.b.in.DatasourceID)
		bCreds, err := g.b.rig.sec.GetMap(ctx, bPath)
		require.NoError(t, err)
		require.NoError(t, g.a.rig.sec.PutMap(ctx, aPath, bCreds))
		p, err := g.router.Resolve(as(g.a.in.TenantID), g.a.in.DatasourceID)
		require.Error(t, err, "B's role must not be able to open A's database")
		require.Nil(t, p)
	})

	t.Run("a cloned datasource that still named the gold copy resolves to nothing", func(t *testing.T) {
		g := newIsoRig(t)
		_, err := g.router.Resolve(as(g.a.in.TenantID), g.a.rig.dsOther)
		require.ErrorIs(t, err, tenantdb.ErrIncomplete, "Bind cleared its database, so it cannot resolve to the gold copy's")
	})

	t.Run("a suspended binding stops only its own tenant", func(t *testing.T) {
		g := newIsoRig(t)
		_, err := g.a.rig.admin.Exec(`UPDATE tenant_datasource_binding SET lifecycle_state = 'suspended' WHERE datasource_id = $1`, g.a.in.DatasourceID)
		require.NoError(t, err)
		_, err = g.router.Resolve(as(g.a.in.TenantID), g.a.in.DatasourceID)
		require.ErrorIs(t, err, tenantdb.ErrUnbound)
		_, err = g.router.Resolve(as(g.b.in.TenantID), g.b.in.DatasourceID)
		require.NoError(t, err, "B must be unaffected")
	})

	t.Run("a binding filed under the wrong tenant authorizes nothing", func(t *testing.T) {
		g := newIsoRig(t)
		_, err := g.a.rig.admin.Exec(`UPDATE tenant_datasource_binding SET tenant_id = $2 WHERE datasource_id = $1`, g.a.in.DatasourceID, g.b.in.TenantID)
		require.NoError(t, err)
		_, err = g.router.Resolve(as(g.a.in.TenantID), g.a.in.DatasourceID)
		require.ErrorIs(t, err, tenantdb.ErrUnbound, "row-level security hides a binding that is not the owner's")
	})
}

// The cutover: the registry flips and the next resolve follows it, per tenant.
func TestIsolation_ACutoverMovesOnlyItsOwnTenant(t *testing.T) {
	g := newIsoRig(t)
	g.a.addNote(t, g.router, "before")

	// Tenant A's datasource is repointed at a new database that holds its data after a move.
	next := g.a.rig.database + "_v2"
	cl, err := g.a.rig.cluster.Open("postgres")
	require.NoError(t, err)
	_, err = cl.Exec(`CREATE DATABASE ` + next)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = cl.Exec(`DROP DATABASE IF EXISTS ` + next + ` WITH (FORCE)`); cl.Close() })
	nd, err := g.a.rig.cluster.Open(next)
	require.NoError(t, err)
	_, err = nd.Exec(isoMigration)
	require.NoError(t, err)
	_, err = nd.Exec(fmt.Sprintf(`GRANT CONNECT ON DATABASE %s TO %s_app; GRANT USAGE ON SCHEMA public TO %s_app;
		GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO %s_app; GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO %s_app`,
		next, g.a.rig.database, g.a.rig.database, g.a.rig.database, g.a.rig.database))
	require.NoError(t, err)
	nd.Close()

	_, err = g.a.rig.admin.Exec(`UPDATE tenant_product_datasource SET config = config || jsonb_build_object('database', $2::text) WHERE id = $1`, g.a.in.DatasourceID, next)
	require.NoError(t, err)
	_, err = g.a.rig.admin.Exec(`UPDATE tenant_datasource_binding SET version = version + 1 WHERE datasource_id = $1`, g.a.in.DatasourceID)
	require.NoError(t, err)

	dbA, notesA := g.a.notes(t, g.router)
	require.Equal(t, next, dbA, "after the binding version bump A's next resolve reaches the new database")
	require.Empty(t, notesA)
	dbB, notesB := g.b.notes(t, g.router)
	require.Equal(t, g.b.rig.database, dbB, "B must not move")
	require.Empty(t, notesB)
}

var _ = sql.ErrNoRows
var _ = migrations.ErrDrift

// A tenant's role must reach exactly one database. The saga proves it before activating the
// binding, by trying every other connectable database as the role.
func TestIsolation_ProvisioningRefusesARoleThatCanReachAnotherDatabase(t *testing.T) {
	ctx := context.Background()

	probe := func(t *testing.T, r *sagaRig) error {
		t.Helper()
		in := r.provisioned(t)
		return r.acts.ProbeTenantDatabase(ctx, in)
	}

	t.Run("a hardened cluster passes", func(t *testing.T) {
		require.NoError(t, probe(t, newSagaRig(t)))
	})

	t.Run("an open sibling database fails, naming it and the fix", func(t *testing.T) {
		r := newSagaRig(t)
		open := "tdb_open_" + r.database[len("tdb_saga_"):]
		adm, err := r.cluster.Open("postgres")
		require.NoError(t, err)
		_, err = adm.Exec(`CREATE DATABASE ` + open) // default: PUBLIC may connect
		require.NoError(t, err)
		t.Cleanup(func() { _, _ = adm.Exec(`DROP DATABASE IF EXISTS ` + open + ` WITH (FORCE)`); adm.Close() })

		err = probe(t, r)
		require.Error(t, err)
		require.True(t, isNonRetryableOf(err, "TenantDatabaseNotIsolated"), "%v", err)
		require.ErrorContains(t, err, open)
		require.ErrorContains(t, err, `REVOKE CONNECT ON DATABASE "`+open+`" FROM PUBLIC`)

		// Once the operator closes it, provisioning goes through.
		_, err = adm.Exec(`REVOKE CONNECT ON DATABASE ` + open + ` FROM PUBLIC`)
		require.NoError(t, err)
		in := r.in()
		in.DatasourceID = r.dsOrm
		require.NoError(t, r.acts.ProbeTenantDatabase(ctx, in))
	})

	t.Run("an open control-plane database fails", func(t *testing.T) {
		r := newSagaRig(t)
		var alpha string
		require.NoError(t, r.admin.QueryRow(`SELECT current_database()`).Scan(&alpha))
		_, err := r.admin.Exec(fmt.Sprintf(`GRANT CONNECT ON DATABASE "%s" TO PUBLIC`, alpha))
		require.NoError(t, err)
		err = probe(t, r)
		require.True(t, isNonRetryableOf(err, "TenantDatabaseNotIsolated"), "%v", err)
		require.ErrorContains(t, err, alpha, "the control plane must not be reachable by a tenant's role")
	})

	t.Run("a connection failure that is not a denied privilege is not a pass", func(t *testing.T) {
		r := newSagaRig(t)
		odd := "tdb_odd_" + r.database[len("tdb_saga_"):]
		adm, err := r.cluster.Open("postgres")
		require.NoError(t, err)
		// PUBLIC may connect (the default) but the database accepts no connections: the role is
		// refused with "too many connections", which says nothing about its privileges.
		_, err = adm.Exec(`CREATE DATABASE ` + odd + ` CONNECTION LIMIT 0`)
		require.NoError(t, err)
		t.Cleanup(func() { _, _ = adm.Exec(`DROP DATABASE IF EXISTS ` + odd + ` WITH (FORCE)`); adm.Close() })

		err = probe(t, r)
		require.Error(t, err, "an unexpected error must not be read as isolation")
		require.ErrorContains(t, err, "cannot verify isolation")
		require.False(t, isNonRetryableOf(err, "TenantDatabaseNotIsolated"), "it is a retryable inability to verify, not a proven hole")
	})

	t.Run("not being able to tell is not a pass", func(t *testing.T) {
		r := newSagaRig(t)
		in := r.provisioned(t)
		// The router's own credential read (through Creds) still works, so the probe gets as far as
		// the isolation check; only the check's read of the credential fails.
		r.acts.Secrets = unreadable{r.sec}
		err := r.acts.ProbeTenantDatabase(ctx, in)
		require.Error(t, err)
		require.ErrorContains(t, err, "cannot verify isolation")
	})
}

// unreadable is a secrets provider whose GetMap always fails.
type unreadable struct{ secrets.Provider }

func (unreadable) GetMap(context.Context, string) (map[string]string, error) {
	return nil, secrets.ErrSecretNotFound
}

// A new tenant database is closed to every role the moment it exists, and a retry closes it too.
func TestIsolation_CreateTenantDatabaseClosesPublicConnect(t *testing.T) {
	r := newSagaRig(t) // the rig created r.database through the real activity
	ctx := context.Background()

	canConnect := func() bool {
		var ok bool
		require.NoError(t, r.admin.QueryRow(`SELECT has_database_privilege('public_probe_role', $1, 'CONNECT')`, r.database).Scan(&ok))
		return ok
	}
	_, err := r.admin.Exec(`DROP ROLE IF EXISTS public_probe_role; CREATE ROLE public_probe_role NOLOGIN`)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = r.admin.Exec(`DROP ROLE IF EXISTS public_probe_role`) })

	require.False(t, canConnect(), "a freshly created tenant database must not be connectable by an arbitrary role")

	// A failure after CREATE, then a retry ("already exists"), must still end closed.
	adm, err := r.cluster.Open("postgres")
	require.NoError(t, err)
	defer adm.Close()
	_, err = adm.Exec(fmt.Sprintf(`GRANT CONNECT ON DATABASE "%s" TO PUBLIC`, r.database))
	require.NoError(t, err)
	require.True(t, canConnect())
	require.NoError(t, r.acts.CreateTenantDatabase(ctx, r.database), "the retry path")
	require.False(t, canConnect(), "the already-exists path must close PUBLIC's CONNECT too")
}

// The manager used by the wealth service and business-object instance operations serves each tenant
// through the router: its own database, no second pool set, and a closed door for anyone else.
func TestIsolation_TheTenantDBManagerServesEachTenantThroughTheRouter(t *testing.T) {
	g := newIsoRig(t)
	m := platform.NewTenantDBManagerWithRouter(g.router, "orm")

	dbOf := func(tenant string) (string, *sql.DB) {
		conn, err := m.GetConnection(tenant)
		require.NoError(t, err)
		var name string
		require.NoError(t, conn.QueryRow(`SELECT current_database()`).Scan(&name))
		return name, conn
	}
	nameA, connA := dbOf(g.a.in.TenantID)
	nameB, connB := dbOf(g.b.in.TenantID)
	require.Equal(t, g.a.rig.database, nameA)
	require.Equal(t, g.b.rig.database, nameB)

	_, err := connA.Exec(`INSERT INTO notes (owner, body) VALUES ('a', 'only-a')`)
	require.NoError(t, err)
	var n int
	require.NoError(t, connB.QueryRow(`SELECT count(*) FROM notes`).Scan(&n))
	require.Zero(t, n, "tenant B's connection must not see tenant A's row")

	for i := 0; i < 5; i++ {
		dbOf(g.a.in.TenantID)
		dbOf(g.b.in.TenantID)
	}
	require.Equal(t, 2, g.router.Size(), "the manager must not hold a second pool set: one pool per tenant, owned by the router")

	t.Run("a tenant with no datasource for the app gets nothing", func(t *testing.T) {
		conn, err := m.GetConnection(uuid.NewString())
		require.Error(t, err)
		require.Nil(t, conn)
	})
	t.Run("a different app code finds nothing", func(t *testing.T) {
		conn, err := platform.NewTenantDBManagerWithRouter(g.router, "wealth").GetConnection(g.a.in.TenantID)
		require.Error(t, err)
		require.Nil(t, conn)
	})
	t.Run("a suspended tenant is refused and the other is unaffected", func(t *testing.T) {
		_, err := g.a.rig.admin.Exec(`UPDATE tenant_datasource_binding SET lifecycle_state = 'suspended' WHERE datasource_id = $1`, g.a.in.DatasourceID)
		require.NoError(t, err)
		_, err = m.GetConnection(g.a.in.TenantID)
		require.ErrorIs(t, err, tenantdb.ErrUnbound)
		_, err = m.GetConnection(g.b.in.TenantID)
		require.NoError(t, err)
	})
}

// The operator's fix for what the probe refuses is scripts/harden-tenant-cluster.sh. This ties the
// two together: provisioning refuses a cluster with an open database, the script closes it
// (dry run first, idempotent), and provisioning then goes through. It also proves the script keeps
// its safety promises on a real cluster.
func TestIsolation_TheHardeningScriptClosesWhatTheProbeRefuses(t *testing.T) {
	if _, err := exec.LookPath("psql"); err != nil {
		t.Skip("psql not on PATH")
	}
	r := newSagaRig(t)
	ctx := context.Background()
	in := r.provisioned(t)

	script, err := filepath.Abs("../../../../scripts/harden-tenant-cluster.sh")
	require.NoError(t, err)
	runScript := func(args ...string) (string, int) {
		cmd := exec.Command(script, args...)
		cmd.Env = append(os.Environ(), "PGHOST="+r.cluster.Host, "PGPORT="+strconv.Itoa(r.cluster.Port),
			"PGUSER="+r.cluster.User, "PGPASSWORD="+r.cluster.Password, "PGDATABASE=postgres")
		out, err := cmd.CombinedOutput()
		code := 0
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
		} else {
			require.NoError(t, err)
		}
		return string(out), code
	}

	open := "tdb_legacy_" + r.database[len("tdb_saga_"):]
	adm, err := r.cluster.Open("postgres")
	require.NoError(t, err)
	_, err = adm.Exec(`CREATE DATABASE ` + open) // a legacy path that never revoked PUBLIC
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = adm.Exec(`DROP DATABASE IF EXISTS ` + open + ` WITH (FORCE)`); adm.Close() })

	require.True(t, isNonRetryableOf(r.acts.ProbeTenantDatabase(ctx, in), "TenantDatabaseNotIsolated"),
		"provisioning must refuse while a database is open")

	out, code := runScript()
	require.Equal(t, 3, code, "a dry run with work to do exits 3:\n%s", out)
	require.Contains(t, out, `PLAN REVOKE CONNECT ON DATABASE `+open+` FROM PUBLIC;`)
	require.True(t, isNonRetryableOf(r.acts.ProbeTenantDatabase(ctx, in), "TenantDatabaseNotIsolated"), "a dry run must change nothing")

	out, code = runScript("--apply")
	require.Equal(t, 0, code, out)
	require.Contains(t, out, "verified")
	require.NoError(t, r.acts.ProbeTenantDatabase(ctx, in), "after the script, provisioning goes through")

	out, code = runScript("--apply")
	require.Equal(t, 0, code, out)
	require.Contains(t, out, "nothing to do", "a second run must change nothing")
}

// The script must refuse to lock out an ordinary role that only had access through PUBLIC.
func TestIsolation_TheHardeningScriptRefusesToLockOutAnOrdinaryRole(t *testing.T) {
	if _, err := exec.LookPath("psql"); err != nil {
		t.Skip("psql not on PATH")
	}
	r := newSagaRig(t)
	script, _ := filepath.Abs("../../../../scripts/harden-tenant-cluster.sh")
	adm, err := r.cluster.Open("postgres")
	require.NoError(t, err)
	t.Cleanup(func() { adm.Close() })

	open, who := "tdb_lockout_"+r.database[len("tdb_saga_"):], "reporting_reader_"+r.database[len("tdb_saga_"):]
	_, err = adm.Exec(`CREATE DATABASE ` + open)
	require.NoError(t, err)
	_, err = adm.Exec(`CREATE ROLE ` + who + ` LOGIN`) // not a tenant role: it relies on PUBLIC
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = adm.Exec(`DROP DATABASE IF EXISTS ` + open + ` WITH (FORCE)`)
		_, _ = adm.Exec(`DROP ROLE IF EXISTS ` + who)
	})

	run := func(args ...string) (string, int) {
		cmd := exec.Command(script, args...)
		cmd.Env = append(os.Environ(), "PGHOST="+r.cluster.Host, "PGPORT="+strconv.Itoa(r.cluster.Port),
			"PGUSER="+r.cluster.User, "PGPASSWORD="+r.cluster.Password, "PGDATABASE=postgres")
		out, err := cmd.CombinedOutput()
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return string(out), ee.ExitCode()
		}
		require.NoError(t, err)
		return string(out), 0
	}

	out, code := run("--apply")
	require.Equal(t, 4, code, "must refuse, not lock the role out:\n%s", out)
	require.Contains(t, out, "LOSES-ACCESS role "+who+" on database "+open)
	var acl *string
	require.NoError(t, adm.QueryRow(`SELECT datacl::text FROM pg_database WHERE datname = $1`, open).Scan(&acl))
	require.Nil(t, acl, "a refused apply must change nothing")

	out, code = run("--apply", "--grant", open+"="+who)
	require.Equal(t, 0, code, out)
	var ok bool
	require.NoError(t, adm.QueryRow(`SELECT has_database_privilege($1, $2, 'CONNECT')`, who, open).Scan(&ok))
	require.True(t, ok, "the granted role keeps its access")
}
