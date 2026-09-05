package datapipeline

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func TestEngine_DiamondDAG_PersistsToCatalogAndRespectsOrder(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration gate")
	}

	ctx := context.Background()

	db, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("DB ping failed: %v", err)
	}

	tenantID := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	nodeTypeID := uuid.MustParse("68d6d495-0992-4d92-ad2f-7f66dc1e7d78")
	runAID := uuid.New()
	runBID := uuid.New()
	patternA := runAID.String()[:8]
	patternB := runBID.String()[:8]

	engine := NewPipelineEngine(db, nil, nil)

	dag := func(pattern string) PipelineDAG {
		return PipelineDAG{
			Nodes: []PipelineNode{
				{ID: "a", Type: "source", SubType: "raw_json", Label: "Source",
					Config: map[string]interface{}{
						"raw_data": []interface{}{
							map[string]interface{}{
								"qualified_path": "diamond/test-" + pattern + "/alpha",
								"node_name":      "alpha",
								"value":          float64(100),
							},
							map[string]interface{}{
								"qualified_path": "diamond/test-" + pattern + "/beta",
								"node_name":      "beta",
								"value":          float64(200),
							},
						},
					}},
			{ID: "b", Type: "transform", SubType: "column_mapper", Label: "BranchB",
				Config: map[string]interface{}{
					"mappings": map[string]interface{}{"node_name": "node_name", "value": "value"},
				}},
			{ID: "c", Type: "transform", SubType: "column_mapper", Label: "BranchC",
				Config: map[string]interface{}{
					"mappings": map[string]interface{}{"node_name": "node_name", "value": "value"},
				}},
				{ID: "d", Type: "loader", SubType: "catalog_graph", Label: "CatalogSink",
					Config: map[string]interface{}{
						"node_type_id": nodeTypeID.String(),
					}},
			},
			Edges: []PipelineEdge{
				{ID: "e1", Source: "a", Target: "b"},
				{ID: "e2", Source: "a", Target: "c"},
				{ID: "e3", Source: "b", Target: "d"},
				{ID: "e4", Source: "c", Target: "d"},
			},
		}
	}

	preClean := "DELETE FROM catalog_node WHERE qualified_path LIKE 'diamond/test-%'"
	if _, err := db.ExecContext(ctx, preClean); err != nil {
		t.Fatalf("pre-cleanup failed: %v", err)
	}

	runA, err := engine.ExecuteRun(ctx, tenantID, PipelineDefinition{
		ID:          runAID,
		TenantID:    tenantID,
		Name:        "Diamond Forward",
		DAGJSON:     mustMarshal(dag(patternA)),
		Concurrency: 4,
		BatchSize:   100,
		ErrorPolicy: "fail_fast",
	}, nil, false, nil)
	if err != nil {
		t.Fatalf("forward run failed: %v", err)
	}

	runB, err := engine.ExecuteRun(ctx, tenantID, PipelineDefinition{
		ID:          runBID,
		TenantID:    tenantID,
		Name:        "Diamond Reversed",
		DAGJSON:     mustMarshal(dag(patternB)),
		Concurrency: 4,
		BatchSize:   100,
		ErrorPolicy: "fail_fast",
	}, nil, false, nil)
	if err != nil {
		t.Fatalf("reversed run failed: %v", err)
	}

	if len(runA.StepOrder) == 0 || len(runB.StepOrder) == 0 {
		t.Fatalf("StepOrder not populated: runA=%v runB=%v", runA.StepOrder, runB.StepOrder)
	}

	forwardOrderCheck := func(label string, order []string) {
		a := indexOf(order, "a")
		b := indexOf(order, "b")
		c := indexOf(order, "c")
		d := indexOf(order, "d")
		if a < 0 || b < 0 || c < 0 || d < 0 {
			t.Errorf("%s: StepOrder missing nodes: %v", label, order)
			return
		}
		if a >= b || a >= c {
			t.Errorf("%s: a must precede b and c; StepOrder=%v", label, order)
		}
		if b >= d || c >= d {
			t.Errorf("%s: b and c must precede d; StepOrder=%v", label, order)
		}
	}

	forwardOrderCheck("forward", runA.StepOrder)
	forwardOrderCheck("reversed", runB.StepOrder)

	if runA.TotalRecordsOut != runB.TotalRecordsOut {
		t.Errorf("record counts differ: forward=%d reversed=%d",
			runA.TotalRecordsOut, runB.TotalRecordsOut)
	}

	if runA.TotalRecordsOut == 0 {
		t.Errorf("forward run produced 0 records")
	}

	var countA int
	err = db.GetContext(ctx, &countA,
		"SELECT COUNT(*) FROM catalog_node WHERE tenant_id = $1 AND qualified_path LIKE 'diamond/test-"+patternA+"%'",
		tenantID)
	if err != nil {
		t.Fatalf("catalog query A failed: %v", err)
	}
	if countA != 2 {
		t.Errorf("forward run: expected 2 rows in catalog_node, got %d (qualified_path LIKE 'diamond/test-%s%%')", countA, patternA)
	}

	var countB int
	err = db.GetContext(ctx, &countB,
		"SELECT COUNT(*) FROM catalog_node WHERE tenant_id = $1 AND qualified_path LIKE 'diamond/test-"+patternB+"%'",
		tenantID)
	if err != nil {
		t.Fatalf("catalog query B failed: %v", err)
	}
	if countB != 2 {
		t.Errorf("reversed run: expected 2 rows in catalog_node, got %d (qualified_path LIKE 'diamond/test-%s%%')", countB, patternB)
	}

	var nodeNameA string
	err = db.GetContext(ctx, &nodeNameA,
		"SELECT node_name FROM catalog_node WHERE tenant_id = $1 AND qualified_path = 'diamond/test-"+patternA+"/alpha' LIMIT 1",
		tenantID)
	if err != nil {
		t.Fatalf("node_name query failed: %v", err)
	}
	if nodeNameA != "alpha" {
		t.Errorf("node_name mismatch: expected 'alpha', got %q", nodeNameA)
	}

	postClean := "DELETE FROM catalog_node WHERE qualified_path LIKE 'diamond/test-%'"
	if _, err := db.ExecContext(ctx, postClean); err != nil {
		t.Logf("post-cleanup failed (non-fatal): %v", err)
	}
}
