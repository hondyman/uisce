package datapipeline

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func pipelineDB(t *testing.T) *sqlx.DB {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()
	pg, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("alpha"), postgres.WithUsername("postgres"), postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(90*time.Second)))
	if err != nil {
		t.Skipf("docker unavailable: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })
	dsn, _ := pg.ConnectionString(ctx, "sslmode=disable")
	db := sqlx.MustConnect("postgres", dsn)
	t.Cleanup(func() { db.Close() })
	// The real migrations that own these tables.
	for _, m := range []string{"20260901_001_create_data_pipelines.up.sql", "20260909_001_datapipeline_run_persistence.up.sql", "20261027_001_data_pipeline_schedule.up.sql"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "db", "migrations", m))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(b)); err != nil {
			t.Fatalf("%s: %v", m, err)
		}
	}
	return db
}

func TestStoreIntegration(t *testing.T) {
	db := pipelineDB(t)
	st := &Store{DB: db}
	ctx := context.Background()

	spec := Spec{Version: SpecVersion, Nodes: []Node{
		{ID: "in", Type: NodeBOSource, Label: "Funds", Config: cfg(BOSourceConfig{BOKey: "fund"})},
		{ID: "rc", Type: NodeRuleCheck, Label: "Rules", Config: cfg(RuleCheckConfig{RuleIDs: []string{"r"}})},
		{ID: "out", Type: NodeBOSink, Label: "Copy", Config: cfg(BOSinkConfig{BOKey: "fund_copy"})},
	}, Edges: []Edge{{From: "in", To: "rc"}, {From: "rc", To: "out"}}}
	d := &Definition{Name: "Fund copy", Spec: spec}
	if err := st.Save(ctx, tenantA, "analyst", d); err != nil {
		t.Fatal(err)
	}

	// Tenant isolation on every read and write.
	if _, err := st.Get(ctx, tenantB, d.ID); err != ErrNotFound {
		t.Fatalf("other tenant must not see it: %v", err)
	}
	if l, _ := st.List(ctx, tenantB); len(l) != 0 {
		t.Fatal("other tenant list must be empty")
	}
	other := &Definition{ID: d.ID, Name: "hijack", Spec: spec}
	if err := st.Save(ctx, tenantB, "x", other); err != ErrNotFound {
		t.Fatalf("other tenant must not update it: %v", err)
	}
	if err := st.Delete(ctx, tenantB, d.ID); err != ErrNotFound {
		t.Fatal("other tenant must not delete it")
	}

	// Update keeps identity.
	d.Name = "Fund copy v2"
	if err := st.Save(ctx, tenantA, "analyst", d); err != nil {
		t.Fatal(err)
	}
	got, err := st.Get(ctx, tenantA, d.ID)
	if err != nil || got.Name != "Fund copy v2" || len(got.Spec.Nodes) != 3 {
		t.Fatalf("get: %+v %v", got, err)
	}

	// Execute: telemetry, counts, error sample, snapshot.
	src := &fakeBO{}
	for i := 0; i < 5; i++ {
		src.store = append(src.store, map[string]any{"aum": i})
	}
	runID, err := st.CreateRun(ctx, tenantA, got)
	if err != nil {
		t.Fatal(err)
	}
	got.Spec.Nodes = nil // later edits must not affect the run
	if _, err := st.Execute(ctx, Deps{BO: src, Rules: blockOdd{}}, tenantA, runID); err != nil {
		t.Fatal(err)
	}
	run, err := st.GetRun(ctx, tenantA, runID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "completed_with_errors" || run.RecordsIn != 5 || run.RecordsOut != 3 || run.Errors != 2 {
		t.Fatalf("run: %+v", run)
	}
	if len(run.Steps) != 3 || run.Steps[1].NodeID != "rc" || run.Steps[1].Errors != 2 {
		t.Fatalf("steps: %+v", run.Steps)
	}
	if string(run.ErrorJSON) == "[]" {
		t.Error("error sample must be kept")
	}
	if _, err := st.GetRun(ctx, tenantB, runID); err != ErrNotFound {
		t.Error("other tenant must not see the run")
	}
	if runs, _ := st.ListRuns(ctx, tenantA, d.ID, 10); len(runs) != 1 {
		t.Error("run history")
	}

	// A failing run is recorded as failed with its error.
	runID2, _ := st.CreateRun(ctx, tenantA, d)
	if _, err := st.Execute(ctx, Deps{}, tenantA, runID2); err == nil {
		t.Fatal("missing deps must fail the run")
	}
	r2, _ := st.GetRun(ctx, tenantA, runID2)
	if r2.Status != "failed" {
		t.Errorf("status = %s", r2.Status)
	}

	// Schedules are stored per pipeline, tenant-scoped.
	sc := &Schedule{Cron: "0 6 * * 1-5", TimeZone: "Europe/Dublin", Enabled: true}
	if err := st.SetSchedule(ctx, tenantA, d.ID, sc); err != nil {
		t.Fatal(err)
	}
	if got, _ := st.GetSchedule(ctx, tenantA, d.ID); got == nil || *got != *sc {
		t.Errorf("schedule round trip: %+v", got)
	}
	if err := st.SetSchedule(ctx, tenantB, d.ID, sc); err != ErrNotFound {
		t.Error("other tenant must not set the schedule")
	}
	if err := st.SetSchedule(ctx, tenantA, d.ID, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := st.GetSchedule(ctx, tenantA, d.ID); got != nil {
		t.Error("cleared schedule must read as nil")
	}

	if err := st.Delete(ctx, tenantA, d.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Get(ctx, tenantA, d.ID); err != ErrNotFound {
		t.Error("deleted pipeline must be gone")
	}
}
