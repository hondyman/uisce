package datapipeline

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func TestEngine_DiamondDAG_OrderInvariantUnderNodeShuffle(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration gate (CI portability)")
	}

	ctx := context.Background()

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("DB ping failed: %v", err)
	}

	tenantID := uuid.New()
	runSuffix := uuid.New().String()[:8]

	setupSQL := fmt.Sprintf(`
		DROP TABLE IF EXISTS diamond_sink_%s CASCADE;
		DROP TABLE IF EXISTS diamond_source_%s CASCADE;

		CREATE TABLE diamond_source_%s (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			pipeline_run_id UUID NOT NULL,
			tenant_id UUID NOT NULL,
			source_col TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE TABLE diamond_sink_%s (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			pipeline_run_id UUID NOT NULL,
			tenant_id UUID NOT NULL,
			sink_col TEXT,
			sink_marker TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);

		INSERT INTO diamond_source_%s (pipeline_run_id, tenant_id, source_col)
		VALUES ('00000000-0000-0000-0000-000000000001', '%s', 'source-value-1'),
		       ('00000000-0000-0000-0000-000000000001', '%s', 'source-value-2');
	`, runSuffix, runSuffix, runSuffix, runSuffix, runSuffix, tenantID.String(), tenantID.String())

	if _, err := db.ExecContext(ctx, setupSQL); err != nil {
		t.Fatalf("failed to setup fixture tables: %v", err)
	}

	cleanup := func() {
		db.ExecContext(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS diamond_sink_%s CASCADE; DROP TABLE IF EXISTS diamond_source_%s CASCADE;`, runSuffix, runSuffix))
	}
	defer cleanup()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("DB ping after fixture setup: %v", err)
	}

	engine := NewPipelineEngine(nil, nil, nil)

	dagForward := PipelineDAG{
		Nodes: []PipelineNode{
			{ID: "a", Type: "source", SubType: "raw_json", Label: "Source",
				Config: map[string]interface{}{
					"raw_data": []interface{}{
						map[string]interface{}{"src": "a-first"},
						map[string]interface{}{"src": "a-second"},
					},
				}},
			{ID: "b", Type: "transform", SubType: "column_mapper", Label: "Branch B",
				Config: map[string]interface{}{
					"mappings": map[string]interface{}{"out_col": "src"},
				}},
			{ID: "c", Type: "transform", SubType: "column_mapper", Label: "Branch C",
				Config: map[string]interface{}{
					"mappings": map[string]interface{}{"out_col": "src"},
				}},
			{ID: "d", Type: "transform", SubType: "column_mapper", Label: "Sink",
				Config: map[string]interface{}{
					"mappings": map[string]interface{}{"sink_col": "out_col"},
				}},
		},
		Edges: []PipelineEdge{
			{ID: "e1", Source: "a", Target: "b"},
			{ID: "e2", Source: "a", Target: "c"},
			{ID: "e3", Source: "b", Target: "d"},
			{ID: "e4", Source: "c", Target: "d"},
		},
	}

	dagReversed := PipelineDAG{
		Nodes: []PipelineNode{
			{ID: "d", Type: "transform", SubType: "column_mapper", Label: "Sink",
				Config: map[string]interface{}{
					"mappings": map[string]interface{}{"sink_col": "out_col"},
				}},
			{ID: "b", Type: "transform", SubType: "column_mapper", Label: "Branch B",
				Config: map[string]interface{}{
					"mappings": map[string]interface{}{"out_col": "src"},
				}},
			{ID: "c", Type: "transform", SubType: "column_mapper", Label: "Branch C",
				Config: map[string]interface{}{
					"mappings": map[string]interface{}{"out_col": "src"},
				}},
			{ID: "a", Type: "source", SubType: "raw_json", Label: "Source",
				Config: map[string]interface{}{
					"raw_data": []interface{}{
						map[string]interface{}{"src": "a-first"},
						map[string]interface{}{"src": "a-second"},
					},
				}},
		},
		Edges: []PipelineEdge{
			{ID: "e1", Source: "a", Target: "b"},
			{ID: "e2", Source: "a", Target: "c"},
			{ID: "e3", Source: "b", Target: "d"},
			{ID: "e4", Source: "c", Target: "d"},
		},
	}

	def1 := PipelineDefinition{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        "Diamond Forward",
		DAGJSON:     mustMarshal(dagForward),
		Concurrency: 4,
		BatchSize:   100,
		ErrorPolicy: "skip_and_log",
	}

	def2 := PipelineDefinition{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        "Diamond Reversed",
		DAGJSON:     mustMarshal(dagReversed),
		Concurrency: 4,
		BatchSize:   100,
		ErrorPolicy: "skip_and_log",
	}

	run1, err := engine.ExecuteRun(ctx, tenantID, def1, nil, true, nil)
	if err != nil {
		t.Fatalf("ExecuteRun forward failed: %v", err)
	}

	run2, err := engine.ExecuteRun(ctx, tenantID, def2, nil, true, nil)
	if err != nil {
		t.Fatalf("ExecuteRun reversed failed: %v", err)
	}

	if len(run1.StepOrder) == 0 {
		t.Errorf("StepOrder is empty after forward run — topologicalOrder not populating StepOrder")
	}

	if len(run2.StepOrder) == 0 {
		t.Errorf("StepOrder is empty after reversed run — topologicalOrder not populating StepOrder")
	}

	if run1.Status != "simulated" {
		t.Errorf("forward run: expected status 'simulated', got %s", run1.Status)
	}
	if run2.Status != "simulated" {
		t.Errorf("reversed run: expected status 'simulated', got %s", run2.Status)
	}

	gotA1 := indexOf(run1.StepOrder, "a")
	gotB1 := indexOf(run1.StepOrder, "b")
	gotC1 := indexOf(run1.StepOrder, "c")
	gotD1 := indexOf(run1.StepOrder, "d")

	gotA2 := indexOf(run2.StepOrder, "a")
	gotB2 := indexOf(run2.StepOrder, "b")
	gotC2 := indexOf(run2.StepOrder, "c")
	gotD2 := indexOf(run2.StepOrder, "d")

	if gotA1 < 0 || gotB1 < 0 || gotC1 < 0 || gotD1 < 0 {
		t.Errorf("forward run StepOrder missing nodes: %v", run1.StepOrder)
	}
	if gotA2 < 0 || gotB2 < 0 || gotC2 < 0 || gotD2 < 0 {
		t.Errorf("reversed run StepOrder missing nodes: %v", run2.StepOrder)
	}

	if gotA1 >= gotB1 || gotA1 >= gotC1 {
		t.Errorf("forward: a must precede b and c; StepOrder=%v", run1.StepOrder)
	}
	if gotB1 >= gotD1 || gotC1 >= gotD1 {
		t.Errorf("forward: b and c must precede d; StepOrder=%v", run1.StepOrder)
	}

	if gotA2 >= gotB2 || gotA2 >= gotC2 {
		t.Errorf("reversed: a must precede b and c; StepOrder=%v", run2.StepOrder)
	}
	if gotB2 >= gotD2 || gotC2 >= gotD2 {
		t.Errorf("reversed: b and c must precede d; StepOrder=%v", run2.StepOrder)
	}

	if gotB1 == gotC1 {
		t.Errorf("forward: b and c must have distinct positions; StepOrder=%v", run1.StepOrder)
	}
	if gotB2 == gotC2 {
		t.Errorf("reversed: b and c must have distinct positions; StepOrder=%v", run2.StepOrder)
	}

	if run1.TotalRecordsOut != run2.TotalRecordsOut {
		t.Errorf("record counts differ between forward (%d) and reversed (%d)",
			run1.TotalRecordsOut, run2.TotalRecordsOut)
	}

	if run1.TotalRecordsOut == 0 {
		t.Errorf("forward run produced 0 records — diamond path broken; StepOrder=%v", run1.StepOrder)
	}
	if run2.TotalRecordsOut == 0 {
		t.Errorf("reversed run produced 0 records — diamond path broken; StepOrder=%v", run2.StepOrder)
	}
}

func indexOf(slice []string, val string) int {
	for i, v := range slice {
		if v == val {
			return i
		}
	}
	return -1
}

func mustMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
