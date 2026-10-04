package ormmove

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/migrations"
)

// The order in Tables is a claim about the orm->orm foreign-key graph. These two
// tests are what make it a checked claim instead of a comment: if someone adds a
// table, drops an edge, or reorders the slice, they fail here rather than
// producing a copy that fails on a foreign key at 3am.

func TestTablesRespectDeclaredEdges(t *testing.T) {
	placed := map[string]bool{}
	for _, tbl := range Tables {
		name := tbl.Name
		require.False(t, placed[name], "table %s appears twice in the copy order", name)
		for _, parent := range Edges[name] {
			require.True(t, placed[parent],
				"%s is copied before its parent %s; a placement row would land before its order",
				name, parent)
		}
		placed[name] = true
	}
}

func TestEdgeCoverage(t *testing.T) {
	inOrder := map[string]bool{}
	for _, tbl := range Tables {
		inOrder[tbl.Name] = true
	}
	for child, parents := range Edges {
		require.True(t, inOrder[child], "edge from %s, which is not in Tables", child)
		for _, p := range parents {
			require.True(t, inOrder[p], "edge to %s, which is not in Tables", p)
		}
	}
}

func TestTableCountMatchesTheSchema(t *testing.T) {
	require.Len(t, Tables, TableCount, "TableCount is the documented 32; update it deliberately")

	// Every table the migration creates is either moved or explicitly excluded.
	// An omission that is not in ExcludedTables is a bug, which is how `quote`
	// was caught: the list was written with 32 entries and the count said 33.
	moved := tableNames()
	require.NotContains(t, moved, "quote_default", "a partition is not a table")
	for name := range ExcludedTables {
		require.NotContains(t, moved, name, "%s is both moved and excluded", name)
	}

	// And the move must never silently skip a table that has no stated reason.
	schema := schemaTableNames(t)
	for _, name := range schema {
		if name == "quote_default" {
			continue
		}
		if contains(moved, name) {
			continue
		}
		require.Contains(t, ExcludedTables, name,
			"orm.%s exists in the migration but is neither moved nor excluded", name)
	}
}

func TestExclusionsStateAReason(t *testing.T) {
	for name, reason := range ExcludedTables {
		require.NotEmpty(t, strings.TrimSpace(reason), "excluding %s needs a reason", name)
	}
}

// schemaTableNames reads the real migration, so this test fails if the schema
// gains a table the move has not been told about.
func schemaTableNames(t *testing.T) []string {
	t.Helper()
	migration, err := os.ReadFile(ormMigration)
	require.NoError(t, err)
	re := regexp.MustCompile(`CREATE TABLE IF NOT EXISTS orm\.([a-zA-Z_"]+)`)
	var out []string
	for _, m := range re.FindAllStringSubmatch(string(migration), -1) {
		out = append(out, strings.Trim(m[1], `"`))
	}
	return out
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}

func tableNames() []string {
	out := make([]string, 0, len(Tables))
	for _, t := range Tables {
		out = append(out, t.Name)
	}
	return out
}

// --- database-backed tests -----------------------------------------------------
//
// These build the REAL tenant migration (0001_orm_schema.up.sql) in a scratch
// database, seed it, and move rows between two of them. That is the point: the
// move is only correct against the schema ADR-042 actually produces.
//
//	ORMMOVE_ADMIN_DSN  superuser on a SCRATCH database (it creates two more)

var ormMigration = "../../../db/tenant_migrations/orm/0001_orm_schema.up.sql"

const (
	tenantA    = "aaaaaaaa-0000-0000-0000-000000000001"
	tenantB    = "bbbbbbbb-0000-0000-0000-000000000002"
	notATenant = "cccccccc-0000-0000-0000-000000000003"
)

func scratchDBs(t *testing.T) (source, target *sql.DB) {
	t.Helper()
	dsn := os.Getenv("ORMMOVE_ADMIN_DSN")
	if dsn == "" {
		t.Skip("ORMMOVE_ADMIN_DSN not set")
	}
	adm, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { adm.Close() })

	var name string
	require.NoError(t, adm.QueryRow(`SELECT current_database()`).Scan(&name))
	require.NotEqual(t, "alpha", name, "refusing to run against the shared alpha database")
	require.NotEqual(t, "crims", name, "refusing to run against the shared crims database")

	migration, err := os.ReadFile(ormMigration)
	require.NoError(t, err)

	suffix := randomSuffix(t)
	src := openScratch(t, adm, dsn, "ormmove_src_"+suffix, string(migration))
	tgt := openScratch(t, adm, dsn, "ormmove_tgt_"+suffix, string(migration))
	return src, tgt
}

func randomSuffix(t *testing.T) string {
	t.Helper()
	// Keeps concurrent runs apart without pulling in a uuid dependency here.
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

// openScratch creates a database, applies the migration and returns a handle.
func openScratch(t *testing.T, adm *sql.DB, adminDSN, name, migration string) *sql.DB {
	t.Helper()
	_, err := adm.Exec(`DROP DATABASE IF EXISTS ` + pgQuote(name))
	require.NoError(t, err)
	_, err = adm.Exec(`CREATE DATABASE ` + pgQuote(name))
	require.NoError(t, err)
	t.Cleanup(func() { adm.Exec(`DROP DATABASE IF EXISTS ` + pgQuote(name) + ` WITH (FORCE)`) })

	db, err := sql.Open("pgx", swapDatabase(adminDSN, name))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(migration)
	require.NoError(t, err)
	return db
}

func swapDatabase(dsn, name string) string {
	if u, err := url.Parse(dsn); err == nil && u.Scheme != "" {
		u.Path = "/" + name
		return u.String()
	}
	// keyword/value DSN: postgres://... is handled above; this is "host=... dbname=x"
	re := regexp.MustCompile(`(?i)dbname\s*=\s*\S+`)
	if re.MatchString(dsn) {
		return re.ReplaceAllString(dsn, "dbname="+name)
	}
	return dsn + " dbname=" + name
}

func pgQuote(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

// seedOrderAndPlacement inserts the order chain for a tenant, which is what the
// FK order is actually about: an order, its placement, an execution and an
// allocation. Ids are derived from the tenant so seeding two tenants does not
// collide on a primary key.
func seedOrderAndPlacement(t *testing.T, db *sql.DB, tenant string, n int) {
	t.Helper()
	seedID := func(group int, i int) string {
		// The tenant's own first group, so two tenants never share an id.
		// 8-4-4-4-12, as a uuid must be.
		return fmt.Sprintf("%s-%04x-0000-0000-%012d", strings.Split(tenant, "-")[0], group, i+1)
	}
	for i := 0; i < n; i++ {
		orderID := seedID(1, i)
		placementID := seedID(2, i)
		execID := seedID(3, i)
		_, err := db.Exec(`INSERT INTO orm."order"
			(id, sec_id, side, order_type, status, target_qty, leaves_qty, trade_date, tenant_id)
			VALUES ($1, 123, 'BUY', 'MKT', 'NEW', 100, 100, CURRENT_DATE, $2)`, orderID, tenant)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO orm.placement
			(id, order_id, broker_id, routed_qty, leaves_qty, status, fix_clordid, tenant_id)
			VALUES ($1, $2, 'GSCO', 100, 100, 'ROUTED', $3, $4)`,
			placementID, orderID, fmt.Sprintf("C-%d", i), tenant)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO orm.execution
			(id, placement_id, order_id, exec_qty, exec_price, exec_time, transact_time, status, tenant_id)
			VALUES ($1, $2, $3, 100, 1.5, NOW(), NOW(), 'FILLED', $4)`,
			execID, placementID, orderID, tenant)
		require.NoError(t, err)
	}
}

// seedReferenceRows inserts rows owned by the shared reference sentinel, which the
// RLS policies this move replaces let every tenant read. `orm.broker` is keyed by a
// VARCHAR broker_id, not a uuid.
func seedReferenceRows(t *testing.T, db *sql.DB, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		_, err := db.Exec(`INSERT INTO orm.broker (broker_id, status, tenant_id)
			VALUES ($1, 'ACTIVE', $2)`,
			fmt.Sprintf("BRK%03d", i), ReferenceTenant)
		require.NoError(t, err)
	}
}

func countRows(t *testing.T, db *sql.DB, table, tenant string) int64 {
	t.Helper()
	var n int64
	err := db.QueryRow(fmt.Sprintf(
		`SELECT count(*) FROM orm.%q WHERE tenant_id = $1::uuid OR tenant_id = $2::uuid`,
		table), tenant, ReferenceTenant).Scan(&n)
	require.NoError(t, err)
	return n
}

func TestMove_CopiesTheTenantAndTheReferenceRows(t *testing.T) {
	src, tgt := scratchDBs(t)
	seedOrderAndPlacement(t, src, tenantA, 3)
	seedOrderAndPlacement(t, src, tenantB, 2)
	seedReferenceRows(t, src, 4)

	m := &Mover{Source: src}
	rep, err := m.Move(context.Background(), tgt, tenantA)
	require.NoError(t, err)
	require.True(t, rep.Done, "move reported not done: %+v", rep.Counts)
	require.Zero(t, rep.Mismatched)

	// Tenant A's own rows moved, and the shared reference rows came with them.
	require.EqualValues(t, 3, countRows(t, tgt, "order", tenantA))
	require.EqualValues(t, 3, countRows(t, tgt, "placement", tenantA))
	require.EqualValues(t, 3, countRows(t, tgt, "execution", tenantA))
	require.EqualValues(t, 4, countRows(t, tgt, "broker", tenantA))

	// Tenant B's rows did not: the predicate is the tenant, not "everything".
	require.EqualValues(t, 0, countRows(t, tgt, "order", tenantB))
	require.EqualValues(t, 0, countRows(t, tgt, "placement", tenantB))
}

func TestMove_NeverCopiesAnotherTenantsRows(t *testing.T) {
	src, tgt := scratchDBs(t)
	seedOrderAndPlacement(t, src, tenantA, 2)
	seedOrderAndPlacement(t, src, notATenant, 5)

	m := &Mover{Source: src}
	rep, err := m.Move(context.Background(), tgt, tenantA)
	require.NoError(t, err)
	require.True(t, rep.Done)

	// The other tenant's rows exist in the source...
	require.EqualValues(t, 5, countRows(t, src, "order", notATenant))
	// ...and did not follow tenant A into its own database. This is the
	// isolation property: the predicate is the tenant, not "everything in orm".
	require.EqualValues(t, 0, countRows(t, tgt, "order", notATenant))
	require.EqualValues(t, 0, countRows(t, tgt, "placement", notATenant))
	require.EqualValues(t, 2, countRows(t, tgt, "order", tenantA))
	require.EqualValues(t, 2, countRows(t, tgt, "placement", tenantA))
}

func TestMove_IsIdempotent(t *testing.T) {
	src, tgt := scratchDBs(t)
	seedOrderAndPlacement(t, src, tenantA, 3)
	seedReferenceRows(t, src, 2)

	m := &Mover{Source: src}
	first, err := m.Move(context.Background(), tgt, tenantA)
	require.NoError(t, err)
	require.True(t, first.Done)

	second, err := m.Move(context.Background(), tgt, tenantA)
	require.NoError(t, err, "a rerun must not fail on a duplicate key")
	require.True(t, second.Done)
	require.Zero(t, second.Mismatched)

	var inserted int64
	for _, c := range second.Counts {
		inserted += c.Inserted
	}
	require.Zero(t, inserted, "a rerun copied rows again: %+v", second.Counts)
	require.EqualValues(t, 3, countRows(t, tgt, "order", tenantA))
}

func TestMove_RepairsRowsLostFromTheTarget(t *testing.T) {
	src, tgt := scratchDBs(t)
	seedOrderAndPlacement(t, src, tenantA, 3)

	m := &Mover{Source: src}
	rep, err := m.Move(context.Background(), tgt, tenantA)
	require.NoError(t, err)
	require.True(t, rep.Done)

	// Something lost rows behind the mover's back. Re-running is the repair
	// path, and it must end Done again rather than leaving the tenant short.
	_, err = tgt.Exec(`DELETE FROM orm.execution`)
	require.NoError(t, err)
	require.EqualValues(t, 0, countRows(t, tgt, "execution", tenantA))

	rep, err = m.Move(context.Background(), tgt, tenantA)
	require.NoError(t, err)
	require.True(t, rep.Done, "a re-run must restore what was lost: %+v", rep.Counts)
	require.EqualValues(t, 3, countRows(t, tgt, "execution", tenantA))
}

func TestMove_AMismatchIsNotDone(t *testing.T) {
	src, tgt := scratchDBs(t)
	seedOrderAndPlacement(t, src, tenantA, 3)

	// A row the source does not have: the move cannot remove it, so the counts
	// can never agree. This is the case that must NOT be reported done, because
	// the cutover is a one-way door for writes.
	_, err := tgt.Exec(`INSERT INTO orm."order"
		(id, sec_id, side, order_type, status, target_qty, leaves_qty, trade_date, tenant_id)
		VALUES ('ffffffff-ffff-4fff-8000-000000000001', 1, 'BUY', 'MKT', 'NEW', 1, 1, CURRENT_DATE, $1)`,
		tenantA)
	require.NoError(t, err)

	m := &Mover{Source: src}
	rep, err := m.Move(context.Background(), tgt, tenantA)
	require.NoError(t, err, "a count disagreement is reported, not raised: the rest of the run still happens")
	require.False(t, rep.Done, "a target holding a row the source lacks must not be reported done")
	require.NotZero(t, rep.Mismatched)
	require.Contains(t, rep.Error, "do not match the source")

	// Every table is still checked, so one pass says everything that is wrong.
	require.Len(t, rep.Counts, len(Tables))
}

func TestMove_AnEmptyTableIsAnErrorNotASilentZero(t *testing.T) {
	// A source with no orm schema at all: every table reports 0 and the move
	// would otherwise "verify" 0 == 0 and claim success.
	_, tgt := scratchDBs(t)
	empty, err := sql.Open("pgx", swapDatabase(mustCurrentDSN(t), "postgres"))
	require.NoError(t, err)
	t.Cleanup(func() { empty.Close() })

	m := &Mover{Source: empty}
	rep, _ := m.Move(context.Background(), tgt, tenantA)
	require.False(t, rep.Done)
	require.NotEmpty(t, rep.Error, "copying from a database with no orm schema must fail, not succeed")
}

func mustCurrentDSN(t *testing.T) string { return os.Getenv("ORMMOVE_ADMIN_DSN") }

func TestFleet_MovesEveryTenantAndReportsEach(t *testing.T) {
	src, tgt := scratchDBs(t)
	seedOrderAndPlacement(t, src, tenantA, 2)
	seedOrderAndPlacement(t, src, tenantB, 3)

	// Both tenants resolve to the same target here; the point is the reporting.
	conn := func(ctx context.Context, tg migrations.Target) (*sql.DB, func(), error) {
		return tgt, func() {}, nil
	}
	f := &Fleet{Mover: &Mover{Source: src}, Connect: conn, WaveSize: 10}
	rep, err := f.Apply(context.Background(), []migrations.Target{
		{TenantID: tenantA, App: "orm"},
		{TenantID: tenantB, App: "orm"},
	})
	require.NoError(t, err)
	require.True(t, rep.Done, "%+v", rep)
	require.Equal(t, 2, rep.Moved)
	require.Zero(t, rep.Failed)
	require.Zero(t, rep.Skipped)
}

func TestFleet_ABadTenantIsReportedAndTheRestStillMove(t *testing.T) {
	src, tgt := scratchDBs(t)
	seedOrderAndPlacement(t, src, tenantA, 2)

	// A connector that refuses one tenant, so the failure is at the boundary a
	// real deployment would hit (a suspended binding, an unreachable database).
	conn := func(ctx context.Context, tg migrations.Target) (*sql.DB, func(), error) {
		if tg.TenantID == tenantB {
			return nil, nil, fmt.Errorf("tenant %s has no active orm binding", tg.TenantID)
		}
		return tgt, func() {}, nil
	}
	f := &Fleet{Mover: &Mover{Source: src}, Connect: conn, WaveSize: 10}
	rep, err := f.Apply(context.Background(), []migrations.Target{
		{TenantID: tenantA, App: "orm"},
		{TenantID: tenantB, App: "orm"},
	})
	require.NoError(t, err)
	require.False(t, rep.Done, "a fleet with a failed tenant is not done")
	require.Equal(t, 1, rep.Moved)
	require.Equal(t, 1, rep.Failed)
	require.Contains(t, rep.Waves[0][1].Report.Error, "no active orm binding")
}

func TestFleet_StopsAfterTooManyFailuresInAWave(t *testing.T) {
	// Every connect fails, so wave one exhausts the tolerance and wave two is
	// never attempted. scratchDBs is still needed for the admin handle.
	scratchDBs(t)
	conn := func(ctx context.Context, tg migrations.Target) (*sql.DB, func(), error) {
		return nil, nil, fmt.Errorf("down")
	}
	f := &Fleet{Mover: &Mover{}, Connect: conn, WaveSize: 2, MaxWaveFailures: 1}
	rep, err := f.Apply(context.Background(), []migrations.Target{
		{TenantID: tenantA, App: "orm"},
		{TenantID: tenantB, App: "orm"},
		{TenantID: notATenant, App: "orm"},
		{TenantID: "dddddddd-0000-0000-0000-000000000004", App: "orm"},
	})
	require.NoError(t, err)
	require.False(t, rep.Done)
	require.Equal(t, 2, rep.Failed, "the second wave stops after the first wave fails")
	require.Equal(t, 2, rep.Skipped)
}

func TestFleet_RejectsADuplicateTarget(t *testing.T) {
	f := &Fleet{Mover: &Mover{}, Connect: func(context.Context, migrations.Target) (*sql.DB, func(), error) {
		return nil, nil, nil
	}, WaveSize: 1}
	_, err := f.Apply(context.Background(), []migrations.Target{
		{TenantID: tenantA, App: "orm"},
		{TenantID: tenantA, App: "orm"},
	})
	require.Error(t, err, "a tenant listed twice would be moved twice; refuse it")
}

func TestFleet_RejectsAnInvalidTarget(t *testing.T) {
	f := &Fleet{Mover: &Mover{}, Connect: func(context.Context, migrations.Target) (*sql.DB, func(), error) {
		return nil, nil, nil
	}, WaveSize: 1}
	_, err := f.Apply(context.Background(), []migrations.Target{{TenantID: "not-a-uuid", App: "orm"}})
	require.Error(t, err)
}
