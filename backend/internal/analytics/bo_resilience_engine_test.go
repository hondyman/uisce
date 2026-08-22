package analytics

import (
	"strings"
	"testing"
)

func TestBOResilienceEngine_CircularDependencyDetection(t *testing.T) {
	engine := NewBOResilienceEngine(nil)

	// Test 1: Acyclic Calculation DAG
	dagAcyclic := map[string][]string{
		"order_total": {"line_amount", "tax_amount"},
		"line_amount": {"unit_price", "quantity", "discount_rate"},
		"tax_amount":  {"line_amount", "tax_rate"},
		"unit_price":  {},
		"quantity":    {},
		"discount_rate": {},
		"tax_rate":    {},
	}
	cycle, err := engine.DetectCircularCalculations(dagAcyclic)
	if err != nil || cycle != nil {
		t.Fatalf("expected acyclic graph to pass, got err=%v, cycle=%v", err, cycle)
	}

	// Test 2: Cyclic Calculation Loop: line_amount -> discount_rate -> customer_discount -> line_amount
	dagCyclic := map[string][]string{
		"order_total":       {"line_amount"},
		"line_amount":       {"discount_rate"},
		"discount_rate":     {"customer_discount"},
		"customer_discount": {"line_amount"},
	}
	cycle, err = engine.DetectCircularCalculations(dagCyclic)
	if err == nil {
		t.Fatalf("expected circular dependency error, got nil")
	}
	if len(cycle) < 3 {
		t.Fatalf("expected cycle path with at least 3 nodes, got %v", cycle)
	}
	t.Logf("Detected cycle successfully: %v (error: %v)", cycle, err)
}

func TestBOResilienceEngine_TwoStageCTEAndFanOutDefense(t *testing.T) {
	engine := NewBOResilienceEngine(nil)

	spec := QueryPlanSpec{
		TenantID:     "tenant-100-alpha",
		DrivingTable: "public.orders",
		BaseFilterPredicate: map[string]interface{}{
			"status": "COMPLETED",
		},
		Fields: []FieldSpec{
			{Name: "order_id", Role: "DIMENSION"},
			{Name: "freight_amount", Role: "MEASURE", AdditivityScope: AdditivityFullyAdditive},
			{Name: "quantity", Role: "MEASURE", AdditivityScope: AdditivityFullyAdditive},
		},
		Joins: []JoinSpec{
			{
				ParentTable: "public.orders",
				ParentKey:   "order_id",
				ChildTable:  "public.order_details",
				ChildKey:    "order_id",
				Cardinality: CardinalityOneToMany,
				Measures:    []string{"quantity"},
			},
		},
		TwoStageCTEEnabled: true,
	}

	sql, err := engine.CompileResilientQuery(spec)
	if err != nil {
		t.Fatalf("failed compiling resilient query: %v", err)
	}

	// Assertions:
	// 1. Two-Stage CTE generated for order_details to prevent freight_amount multiplication
	if !strings.Contains(sql, "layer_0_public_order_details_agg AS (") {
		t.Errorf("expected Two-Stage CTE aggregation for 1:N child, got SQL:\n%s", sql)
	}
	if !strings.Contains(sql, "COALESCE(SUM(quantity), 0) AS quantity") {
		t.Errorf("expected quantity aggregation in CTE, got SQL:\n%s", sql)
	}
	// 2. Tenant and soft-delete invariants injected
	if !strings.Contains(sql, "WHERE base.tenant_id = 'tenant-100-alpha' AND base.is_deleted = FALSE") {
		t.Errorf("expected tenant and is_deleted invariants, got SQL:\n%s", sql)
	}
	if !strings.Contains(sql, "AND base.status = 'COMPLETED'") {
		t.Errorf("expected custom base filter predicate, got SQL:\n%s", sql)
	}
	t.Logf("Generated resilient SQL successfully:\n%s", sql)
}

func TestBOResilienceEngine_SemiAdditiveAndTieredStorage(t *testing.T) {
	engine := NewBOResilienceEngine(nil)

	spec := QueryPlanSpec{
		TenantID: "tenant-200-beta",
		Fields: []FieldSpec{
			{Name: "account_id", Role: "DIMENSION"},
			{
				Name:            "portfolio_balance",
				Role:            "MEASURE",
				AdditivityScope: AdditivitySemiAdditiveAcrossTime,
			},
			{
				Name:            "yield_rate",
				Role:            "MEASURE",
				AdditivityScope: AdditivityNonAdditive,
				ASTExpression:   "SUM(yield_rate)", // Naive sum that must be rewritten to AVG
			},
		},
		TieredStorage: &TieredStoragePair{
			HotTableName:            "starrocks.trades_realtime",
			ColdTableName:           "iceberg.trades_historical",
			TemporalWatermarkColumn: "trade_date",
			WatermarkCutoff:         "2026-01-01",
		},
	}

	sql, err := engine.CompileResilientQuery(spec)
	if err != nil {
		t.Fatalf("failed compiling query: %v", err)
	}

	// 1. Check Hot/Cold Seam UNION ALL
	if !strings.Contains(sql, "tiered_storage_seam AS (") || !strings.Contains(sql, "UNION ALL") {
		t.Errorf("expected Hot/Cold seam CTE, got:\n%s", sql)
	}
	if !strings.Contains(sql, "WHERE trade_date >= '2026-01-01'") || !strings.Contains(sql, "WHERE trade_date < '2026-01-01'") {
		t.Errorf("expected watermark temporal bounds, got:\n%s", sql)
	}

	// 2. Check Semi-additive snapshot rule
	if !strings.Contains(sql, "LAST_VALUE(portfolio_balance) OVER") {
		t.Errorf("expected LAST_VALUE snapshot rule for semi-additive metric, got:\n%s", sql)
	}

	// 3. Check Non-additive rewrite
	if !strings.Contains(sql, "AVG(yield_rate)") {
		t.Errorf("expected non-additive SUM to be converted to AVG, got:\n%s", sql)
	}
}

func TestBOResilienceEngine_MakerCheckerLifecycle(t *testing.T) {
	engine := NewBOResilienceEngine(nil)

	// 1. Maker submits DRAFT
	status, ver, err := engine.TransitionLifecycle(StatusDraft, "SUBMIT_FOR_APPROVAL", false)
	if err != nil || status != StatusPendingApproval {
		t.Fatalf("expected PENDING_APPROVAL, got status=%s, err=%v", status, err)
	}

	// 2. Maker attempts to approve own submission (Violation of Maker-Checker)
	_, _, err = engine.TransitionLifecycle(StatusPendingApproval, "APPROVE", false)
	if err == nil {
		t.Fatalf("expected maker-checker rejection when maker tries to approve own submission")
	}

	// 3. Checker approves submission
	status, ver, err = engine.TransitionLifecycle(StatusPendingApproval, "APPROVE", true)
	if err != nil || status != StatusPublished || ver != 1 {
		t.Fatalf("expected PUBLISHED with version 1, got status=%s, ver=%d, err=%v", status, ver, err)
	}

	// 4. Draft new version from Published
	status, ver, err = engine.TransitionLifecycle(StatusPublished, "DRAFT_NEW_VERSION", false)
	if err != nil || status != StatusDraft || ver != 2 {
		t.Fatalf("expected DRAFT with version 2, got status=%s, ver=%d, err=%v", status, ver, err)
	}
}
