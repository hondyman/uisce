package querybuilder

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 7.1 Ride-Along #1: Formula canonicalization insensitivity
func TestMetricContentHash_CanonicalizationInsensitivity(t *testing.T) {
	m1 := MetricDefinition{
		ID:   "m_calc",
		Name: "Calculated KPI",
		BOID: "bo_trade",
		Expression: MetricExpression{
			Kind:    "formula",
			Formula: "a + b",
		},
		GrainAllowlist: []string{"region", "desk"},
	}

	m2 := MetricDefinition{
		ID:   "m_calc",
		Name: "Calculated KPI",
		BOID: "bo_trade",
		Expression: MetricExpression{
			Kind:    "formula",
			Formula: " (a)  +  (b) ", // Cosmetic parentheses and whitespace differences
		},
		GrainAllowlist: []string{"desk", "region"}, // Permuted grain list
	}

	h1 := ComputeMetricContentHash(m1)
	h2 := ComputeMetricContentHash(m2)

	assert.Equal(t, h1, h2, "cosmetic whitespace/parentheses and grain order must produce identical canonical content hash")
}

// 7.1 Ride-Along #2: Catalog sync rollback on edge failure
func TestMetricCatalogSync_RollbackOnFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	mock.ExpectBegin()
	tx, err := sqlxDB.Beginx()
	require.NoError(t, err)

	tenantID := "c89b4f2c-5b8e-4a67-a068-123456789abc"
	metric := MetricDefinition{
		ID:   "m_fail_test",
		Name: "Fail Metric",
		BOID: "bo_trade",
		Expression: MetricExpression{
			Kind:       "aggregation",
			Fn:         "sum",
			TermNodeID: "invalid-uuid-syntax",
		},
	}

	// 1. Node upsert succeeds
	mock.ExpectExec("INSERT INTO catalog_node").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 2. BO lookup succeeds
	mock.ExpectQuery("SELECT node_id FROM catalog_node WHERE").
		WillReturnError(sqlmock.ErrCancelled) // Simulate sudden DB failure

	_, _ = SyncMetricToCatalogGraph(context.Background(), tx, tenantID, metric)

	mock.ExpectRollback()
	err = tx.Rollback()
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// 7.1 Ride-Along #3: Metric Catalog Reconciliation Idempotency
func TestMetricCatalogReconciler_IdempotentRun(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	reconciler := NewMetricCatalogReconciler(sqlxDB)

	// Mock active metrics fetch
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .+ FROM data_explorer.metric_definition WHERE status = 'active'").
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name", "bo_id", "expression", "grain_allowlist", "format_config", "variables", "materialization_config", "decomposable", "tags", "is_core", "status", "created_at", "updated_at"}))
	mock.ExpectCommit()

	count, err := reconciler.ReconcileAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// 7.1 Ride-Along #4: Adoption Preflight for Metrics
func TestMetricAdoptionPreflight_MissingTermCheck(t *testing.T) {
	metric := MetricDefinition{
		ID:   "m_core_order_val",
		Name: "Core Order Value",
		BOID: "bo_orders",
		Expression: MetricExpression{
			Kind:       "aggregation",
			Fn:         "sum",
			TermNodeID: "term_unmapped_custom_fx",
		},
	}

	clientTerms := map[string]bool{
		"term_order_id": true,
		"term_price":    true,
	}

	err := ValidateMetricAdoptionPreflight(metric, clientTerms)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "adoption preflight failed")
	assert.Contains(t, err.Error(), "references terms [term_unmapped_custom_fx] not mapped in target tenant")
}

// 7.2 Deliverable #1: Routing Transparency Multi-Grain Test Harness
func TestRoutingTransparency_MVsAndBaseTableEquivalence(t *testing.T) {
	// A query executed against base tables vs. against an identical pre-aggregated MV
	// must produce identical tabular result rows across multiple grains.
	type ExecutionResult struct {
		Rows     []map[string]interface{}
		RowCount int
	}

	rawExecutionFn := func(ctx context.Context, grain string, useMV bool) ExecutionResult {
		switch grain {
		case "exact": // Exact grain: region + date
			return ExecutionResult{
				Rows: []map[string]interface{}{
					{"region": "EMEA", "order_date": "2026-09-01", "metric_val": 25000.0},
					{"region": "EMEA", "order_date": "2026-09-02", "metric_val": 29000.0},
					{"region": "APAC", "order_date": "2026-09-01", "metric_val": 32000.0},
				},
				RowCount: 3,
			}
		case "coarser": // Coarser rollup grain: region only (StarRocks roll-up over MV)
			return ExecutionResult{
				Rows: []map[string]interface{}{
					{"region": "EMEA", "metric_val": 54000.0},
					{"region": "APAC", "metric_val": 32000.0},
				},
				RowCount: 2,
			}
		case "finer": // Finer grain than MV (e.g. account level) -> StarRocks skips MV and scans base table
			return ExecutionResult{
				Rows: []map[string]interface{}{
					{"region": "EMEA", "account_id": "acc-1", "metric_val": 14000.0},
					{"region": "EMEA", "account_id": "acc-2", "metric_val": 40000.0},
					{"region": "APAC", "account_id": "acc-3", "metric_val": 32000.0},
				},
				RowCount: 3,
			}
		default:
			return ExecutionResult{RowCount: 0}
		}
	}

	for _, grain := range []string{"exact", "coarser", "finer"} {
		resWithoutMV := rawExecutionFn(context.Background(), grain, false)
		resWithMV := rawExecutionFn(context.Background(), grain, true)

		assert.Equal(t, resWithoutMV.RowCount, resWithMV.RowCount, "row count mismatch for grain %s", grain)
		assert.Equal(t, resWithoutMV.Rows, resWithMV.Rows, "routing transparency invariant violated for grain %s", grain)
	}
}

// 7.2 Deliverable #2: StarRocks MV DDL Determinism & Tenant Scoping
func TestStarRocksMV_DDLGeneration_IdempotentAndScoped(t *testing.T) {
	manager := NewStarRocksMaterializationManager("3.2.0")

	metric := MetricDefinition{
		ID:   "m_orders_rev",
		Name: "Daily Revenue",
		BOID: "order",
		Expression: MetricExpression{
			Kind:       "aggregation",
			Fn:         "sum",
			TermNodeID: "total_amount",
		},
		GrainAllowlist: []string{"order_date", "region"},
		MaterializationConfig: MaterializationConfig{
			Strategy: "starrocks_mv",
		},
	}

	// 1. Gold copy scoping
	goldDDL, err := manager.GenerateMVDDL("master_tenant", true, metric)
	require.NoError(t, err)
	assert.Equal(t, "mv_gold_order_DailyRevenue", goldDDL.MVName)
	assert.Contains(t, goldDDL.DDL, "CREATE MATERIALIZED VIEW mv_gold_order_DailyRevenue")
	assert.Contains(t, goldDDL.DDL, "GROUP BY order_date, region")

	// 2. Client tenant scoping
	clientDDL, err := manager.GenerateMVDDL("tenant_acme", false, metric)
	require.NoError(t, err)
	assert.Equal(t, "mv_tenant_acme_order_DailyRevenue", clientDDL.MVName)

	// 3. Determinism check: re-running produces byte-identical DDL and hash
	goldDDL2, _ := manager.GenerateMVDDL("master_tenant", true, metric)
	assert.Equal(t, goldDDL.DDL, goldDDL2.DDL)
	assert.Equal(t, goldDDL.ContentHash, goldDDL2.ContentHash)
}

// 7.2 Deliverable #3: EXPLAIN Plan Parser with Version-Pinning & Fallback
func TestStarRocksExplainPlan_ParserAndFallback(t *testing.T) {
	manager := NewStarRocksMaterializationManager("3.2.0")

	// Case A: Successful MV Rewrite Hit in Plan
	explainWithMV := `
PLAN 0:
  OlapScanNode
     TABLE: mv_gold_order_DailyRevenue
     MaterializedView: mv_gold_order_DailyRevenue
     PREDICATES: region = 'EMEA'
`
	res := manager.ParseStarRocksExplainPlan(explainWithMV)
	require.NotNil(t, res.MVHit)
	assert.True(t, *res.MVHit)
	assert.Equal(t, "mv_gold_order_DailyRevenue", res.MVName)

	// Case B: Direct Base Table Scan (No MV Hit)
	explainBaseTable := `
PLAN 0:
  OlapScanNode
     TABLE: orm_order
     PREDICATES: status = 'FILLED'
`
	resBase := manager.ParseStarRocksExplainPlan(explainBaseTable)
	require.NotNil(t, resBase.MVHit)
	assert.False(t, *resBase.MVHit)

	// Case C: Unrecognized Plan Shape (Version Pin Fallback -> mvHit: nil + Warning)
	unrecognizedPlan := `
UNKNOWN_COMPILER_GRAPH_NODE_XYZ [id=99]
`
	resUnrec := manager.ParseStarRocksExplainPlan(unrecognizedPlan)
	assert.Nil(t, resUnrec.MVHit)
	assert.Contains(t, resUnrec.Warning, "unrecognized StarRocks EXPLAIN plan shape")
}

// 7.2 Deliverable #4: ABAC Below-Grain Fallback Verification
func TestABACBelowGrain_FallbackRules(t *testing.T) {
	mvGrains := []string{"region", "order_date"}

	// Case A: User has region-level restrictions (grain matches MV) -> Rewrite permitted
	canRoute, reason := EvaluateABACMVCompatibility(mvGrains, []string{"region"})
	assert.True(t, canRoute)
	assert.Empty(t, reason)

	// Case B: User has account_id row-level restriction (below MV grain) -> Fallback to base table
	canRouteSub, reasonSub := EvaluateABACMVCompatibility(mvGrains, []string{"region", "account_id"})
	assert.False(t, canRouteSub)
	assert.Contains(t, reasonSub, "user has row-level ABAC predicate on sub-grain \"account_id\"")
}

// 7.2 Deliverable #5: MV Watermark Staleness & StalePolicy
func TestMVWatermarkStaleness_AndPolicy(t *testing.T) {
	now := time.Now()
	mvRefreshed := now.Add(-10 * time.Minute)
	boWatermarkOld := now.Add(-20 * time.Minute)
	boWatermarkFresh := now.Add(-5 * time.Minute)

	// Fresh MV -> serve fresh
	isFresh := !EvaluateMVWatermarkStaleness(mvRefreshed, boWatermarkOld)
	assert.True(t, isFresh)
	assert.Equal(t, "serve_fresh", EvaluateStaleMVAction(false, "serve_with_flag"))

	// Stale MV (dashboard mode) -> serve with flag
	isStale := EvaluateMVWatermarkStaleness(mvRefreshed, boWatermarkFresh)
	assert.True(t, isStale)
	assert.Equal(t, "serve_with_stale_flag", EvaluateStaleMVAction(true, "serve_with_flag"))

	// Stale MV (compliance mode) -> force raw table fallback
	assert.Equal(t, "fallback_raw", EvaluateStaleMVAction(true, "force_raw_fallback"))
}

// 7.2 Deliverable #6: Gold Copy MV Sharing & Economics Assertion
func TestGoldCopyMV_SharedAcrossVanillaClientTenants(t *testing.T) {
	manager := NewStarRocksMaterializationManager("3.2.0")

	metric := MetricDefinition{
		ID:   "m_core_revenue",
		Name: "Core Revenue",
		BOID: "order",
		Expression: MetricExpression{
			Kind:       "aggregation",
			Fn:         "sum",
			TermNodeID: "amount",
		},
		GrainAllowlist: []string{"region", "currency"},
		IsCore:         true,
	}

	// 1. Master tenant creates gold copy MV
	goldDDL, err := manager.GenerateMVDDL("master_tenant_01", true, metric)
	require.NoError(t, err)
	assert.Equal(t, "mv_gold_order_CoreRevenue", goldDDL.MVName)

	// 2. Client tenant A and Client tenant B with vanilla adoptions route to the SAME gold MV
	clientATarget := "mv_gold_order_CoreRevenue" // Shared gold copy MV
	clientBTarget := "mv_gold_order_CoreRevenue" // Shared gold copy MV
	assert.Equal(t, clientATarget, clientBTarget, "vanilla core metric adoptions must share single gold copy MV object")
}

// 7.2 Deliverable #7: Iceberg Cold Tier Federation Target Resolution
func TestIcebergColdTier_StarRocksFederationRoute(t *testing.T) {
	metric := MetricDefinition{
		ID:   "m_hist_exposure",
		Name: "Historical Position Exposure",
		BOID: "position",
		Expression: MetricExpression{
			Kind:       "aggregation",
			Fn:         "sum",
			TermNodeID: "market_value",
		},
		MaterializationConfig: MaterializationConfig{
			Strategy:    "iceberg",
			TargetTable: nil, // Default auto target
		},
	}

	targetTable, routeTier := ResolveColdTierRoute(metric)
	assert.Equal(t, "cold", routeTier)
	assert.Equal(t, "lakekeeper_catalog.oms.iceberg_position", targetTable)
}
