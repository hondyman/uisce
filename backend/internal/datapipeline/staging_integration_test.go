package datapipeline

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// The staging schema's load-tracking and RLS shape, as in
// db/manual_fixes/008_staging_product.sql.
const stagingSchemaSQL = `
CREATE SCHEMA staging;
CREATE TABLE staging._load_run (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_system_cd varchar(30) NOT NULL, domain varchar(30) NOT NULL, run_ref varchar(100) NOT NULL,
    file_name text, file_hash varchar(64), expected_rows int, received_rows int, accepted_rows int, rejected_rows int,
    started_at timestamptz NOT NULL DEFAULT now(), completed_at timestamptz,
    status varchar(20) NOT NULL DEFAULT 'RUNNING', error_summary text, tenant_id uuid NOT NULL,
    CONSTRAINT uq_lr UNIQUE (tenant_id, source_system_cd, domain, run_ref));
CREATE TABLE staging.ff_fund (
    _load_run_id uuid NOT NULL REFERENCES staging._load_run(id) ON DELETE CASCADE,
    _source_row_num int NOT NULL, tenant_id uuid NOT NULL,
    fsym_id text, fund_name text, aum numeric,
    PRIMARY KEY (_load_run_id, _source_row_num));
ALTER TABLE staging._load_run ENABLE ROW LEVEL SECURITY; ALTER TABLE staging._load_run FORCE ROW LEVEL SECURITY;
ALTER TABLE staging.ff_fund ENABLE ROW LEVEL SECURITY; ALTER TABLE staging.ff_fund FORCE ROW LEVEL SECURITY;
CREATE POLICY lr ON staging._load_run FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::uuid);
CREATE POLICY ff ON staging.ff_fund FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::uuid);
CREATE ROLE loader LOGIN PASSWORD 'loader';
GRANT USAGE ON SCHEMA staging TO loader;
GRANT TEMPORARY ON DATABASE crims TO loader;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA staging TO loader;
`

const (
	tenantA = "11111111-1111-1111-1111-111111111111"
	tenantB = "22222222-2222-2222-2222-222222222222"
)

func stagingDB(t *testing.T) *sql.DB {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()
	pg, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("crims"), postgres.WithUsername("postgres"), postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(90*time.Second)))
	if err != nil {
		t.Skipf("docker unavailable: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })
	admin, _ := pg.ConnectionString(ctx, "sslmode=disable")
	adb, err := sql.Open("postgres", admin)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adb.Exec(stagingSchemaSQL); err != nil {
		t.Fatal(err)
	}
	adb.Close()
	// RLS is bypassed by superusers; load as an ordinary role.
	db, err := sql.Open("postgres", strings.Replace(admin, "postgres:postgres@", "loader:loader@", 1))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func runStaging(t *testing.T, db *sql.DB, tenant, runRef string, rows []Row, runErr error) error {
	n := Node{ID: "s", Type: NodeStagingSink, Config: cfg(StagingSinkConfig{
		Table: "staging.ff_fund", SourceCd: "FACTSET", Domain: "FUND", RunRef: runRef,
		Columns: map[string]string{"FSYM_ID": "fsym_id", "NAME": "fund_name", "AUM": "aum"},
	})}
	p, err := newStagingSink(n, db)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if err := p.Open(ctx, &RunContext{TenantID: tenant, RunID: "run-x"}); err != nil {
		return err
	}
	if _, err := p.Process(ctx, rows); err != nil {
		_ = p.Close(ctx, err)
		return err
	}
	return p.Close(ctx, runErr)
}

func count(t *testing.T, db *sql.DB, tenant, q string) int {
	tx, _ := db.Begin()
	defer tx.Rollback()
	tx.Exec(`SELECT set_config('app.current_tenant', $1, true)`, tenant)
	var n int
	if err := tx.QueryRow(q).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestStagingSinkIntegration(t *testing.T) {
	db := stagingDB(t)
	rows := []Row{
		{Num: 1, Data: map[string]any{"FSYM_ID": "AB12CD-R", "NAME": "Alpha", "AUM": "1000.50"}},
		{Num: 3, Data: map[string]any{"FSYM_ID": "ZZ99YY-R", "NAME": "Beta", "AUM": nil}},
	}

	// Load, then re-run the same run_ref: idempotent no-op.
	if err := runStaging(t, db, tenantA, "20260924-01", rows, nil); err != nil {
		t.Fatal(err)
	}
	if err := runStaging(t, db, tenantA, "20260924-01", rows, nil); err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, tenantA, `SELECT count(*) FROM staging.ff_fund`); n != 2 {
		t.Fatalf("tenant A rows = %d, want 2 (re-run must not duplicate)", n)
	}
	if n := count(t, db, tenantA, `SELECT count(*) FROM staging.ff_fund WHERE _source_row_num = 3 AND aum IS NULL`); n != 1 {
		t.Error("source row numbers and nulls must be preserved")
	}
	if n := count(t, db, tenantA, `SELECT count(*) FROM staging._load_run WHERE status = 'COMPLETED' AND accepted_rows = 2`); n != 1 {
		t.Error("load run must be COMPLETED with accepted_rows")
	}

	// Same run_ref for another tenant is a separate load, invisible to A.
	if err := runStaging(t, db, tenantB, "20260924-01", rows[:1], nil); err != nil {
		t.Fatal(err)
	}
	if count(t, db, tenantA, `SELECT count(*) FROM staging.ff_fund`) != 2 || count(t, db, tenantB, `SELECT count(*) FROM staging.ff_fund`) != 1 {
		t.Error("tenants must be isolated")
	}

	// A failed run is marked FAILED; the retry clears its rows and completes.
	if err := runStaging(t, db, tenantA, "20260925-01", rows, errors.New("upstream failed")); err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, tenantA, `SELECT count(*) FROM staging._load_run WHERE run_ref = '20260925-01' AND status = 'FAILED'`); n != 1 {
		t.Fatal("run should be FAILED")
	}
	if err := runStaging(t, db, tenantA, "20260925-01", rows[:1], nil); err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, tenantA, `SELECT count(*) FROM staging.ff_fund f JOIN staging._load_run r ON r.id = f._load_run_id WHERE r.run_ref = '20260925-01'`); n != 1 {
		t.Errorf("retry must replace the failed run's rows, got %d", n)
	}

	// Mapping to a load-tracking or unknown column is refused.
	bad := Node{ID: "s", Type: NodeStagingSink, Config: cfg(StagingSinkConfig{Table: "staging.ff_fund", SourceCd: "X", Domain: "Y",
		Columns: map[string]string{"T": "tenant_id"}})}
	p, _ := newStagingSink(bad, db)
	if err := p.Open(context.Background(), &RunContext{TenantID: tenantA, RunID: "r"}); err == nil {
		t.Error("mapping onto tenant_id must be refused")
	}
}
