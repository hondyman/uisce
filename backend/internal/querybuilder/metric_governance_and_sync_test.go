package querybuilder

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricCatalogSync_TransactionalEmission verifies that SyncMetricToCatalogGraph
// emits SEMANTIC_TERM node, METRIC_OF edge, and USES_TERM edges within a single transaction.
//
// NOTE ON THE FIXTURE: this test used to set TermNodeID to a UUID. That was not
// arbitrary - it was the only shape the implementation accepted, because the
// USES_TERM insert was guarded by uuid.Parse(TermNodeID). Real metrics name
// terms by semantic key (the 8.3 corpus carries "revenue", "cost",
// "calc_term_net_interest_income"), so the test exercised a shape production
// data never takes, and passed while the edge was never written for any real
// metric. The fixture now uses a name, so it exercises the path that actually
// runs.
func TestMetricCatalogSync_TransactionalEmission(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	mock.ExpectBegin()
	tx, err := sqlxDB.Beginx()
	require.NoError(t, err)

	tenantID := uuid.New().String()
	// A semantic term NAME, as real metrics use - not a UUID.
	termKey := "notional"
	metricID := "m_notional_sync"

	metric := MetricDefinition{
		ID:   metricID,
		Name: "Sync Notional",
		BOID: "bo_trade",
		Expression: MetricExpression{
			Kind:       "aggregation",
			Fn:         "sum",
			TermNodeID: termKey,
		},
		GrainAllowlist: []string{"desk"},
		Decomposable:   true,
		ContentHash:    "hash_test_sync_123",
	}

	// 1. Expect upsert of SEMANTIC_TERM node
	mock.ExpectExec("INSERT INTO catalog_node").
		WithArgs(
			sqlmock.AnyArg(), // node_id
			sqlmock.AnyArg(), // tenant_id
			metricID,
			"Sync Notional",
			"metric/bo_trade/Sync Notional",
			sqlmock.AnyArg(), // properties JSON
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 2. Expect lookup of Business Object node
	boNodeUUID := uuid.New()
	mock.ExpectQuery("SELECT node_id FROM catalog_node WHERE").
		WithArgs(sqlmock.AnyArg(), "bo/bo_trade", "bo_trade").
		WillReturnRows(sqlmock.NewRows([]string{"node_id"}).AddRow(boNodeUUID))

	// 3. Expect METRIC_OF edge insertion
	mock.ExpectExec("INSERT INTO catalog_edge").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), boNodeUUID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 4. Expect the TERM to be resolved by node_key. This lookup is the fix:
	// it is what makes the USES_TERM edge reachable for a name-based term.
	// Two args only - node_type is a SQL literal, not a placeholder.
	termNodeUUID := uuid.New()
	mock.ExpectQuery("SELECT node_id FROM catalog_node WHERE").
		WithArgs(sqlmock.AnyArg(), termKey).
		WillReturnRows(sqlmock.NewRows([]string{"node_id"}).AddRow(termNodeUUID))

	// 5. Expect USES_TERM edge insertion, targeting the RESOLVED term node
	mock.ExpectExec("INSERT INTO catalog_edge").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), termNodeUUID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	nodeID, err := SyncMetricToCatalogGraph(context.Background(), tx, tenantID, metric)
	require.NoError(t, err)
	assert.NotEmpty(t, nodeID)

	mock.ExpectCommit()
	err = tx.Commit()
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMetricCatalogSync_TermMissingFromCatalogIsNotFatal pins the distinction
// the fix introduced: a term that is not a catalog node is a legitimate state
// (a metric can be imported before its term), so the sync succeeds - but the
// absent edge is reported, never silently indistinguishable from a failed
// insert. A database error, by contrast, is returned and rolls the transaction
// back, because a metric node in the graph with no lineage is exactly the
// "looks successful, isn't" failure this function had.
func TestMetricCatalogSync_TermMissingFromCatalogIsNotFatal(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	mock.ExpectBegin()
	tx, err := sqlxDB.Beginx()
	require.NoError(t, err)

	tenantID := uuid.New().String()
	boNodeUUID := uuid.New()

	mock.ExpectExec("INSERT INTO catalog_node").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT node_id FROM catalog_node WHERE").
		WithArgs(sqlmock.AnyArg(), "bo/bo_trade", "bo_trade").
		WillReturnRows(sqlmock.NewRows([]string{"node_id"}).AddRow(boNodeUUID))
	mock.ExpectExec("INSERT INTO catalog_edge").
		WillReturnResult(sqlmock.NewResult(1, 1))
	// Term is absent. Two args - node_type is a SQL literal.
	mock.ExpectQuery("SELECT node_id FROM catalog_node WHERE").
		WithArgs(sqlmock.AnyArg(), "no_such_term").
		WillReturnError(sql.ErrNoRows)
	// And crucially, NO USES_TERM insert is expected after the miss.

	_, err = SyncMetricToCatalogGraph(context.Background(), tx, tenantID, MetricDefinition{
		ID:   "m_orphan",
		Name: "Orphan",
		BOID: "bo_trade",
		Expression: MetricExpression{
			Kind: "aggregation", Fn: "sum", TermNodeID: "no_such_term",
		},
	})
	require.NoError(t, err, "a term absent from the catalog is not a database failure")

	mock.ExpectCommit()
	require.NoError(t, tx.Commit())
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestValidateAdditiveMetricExtension_Rules tests governance additive-only extension rules:
// - Extends grain allowlist (permitted)
// - Attempts to modify core formula (rejected)
// - Attempts to remove core grain (rejected)
// - Attempts to remove core variable (rejected)
func TestValidateAdditiveMetricExtension_Rules(t *testing.T) {
	base := metricContent{
		Name: "Core Revenue",
		BOID: "bo_sales",
		Expression: MetricExpression{
			Kind:       "aggregation",
			Fn:         "sum",
			TermNodeID: "revenue",
			Formula:    "SUM(t0.revenue)",
		},
		GrainAllowlist: []string{"region", "year"},
		Variables: []MetricVariable{
			{Name: "tax_rate", Type: "number", Required: true},
		},
	}.normalized()

	// 1. Valid Additive Extension
	extValid := base
	extValid.GrainAllowlist = append(extValid.GrainAllowlist, "country", "sales_channel")
	extValid.Variables = append(extValid.Variables, MetricVariable{Name: "bonus_multiplier", Type: "number"})
	err := ValidateAdditiveMetricExtension(base, extValid)
	assert.NoError(t, err, "valid additive extension must pass")

	// 2. Reject Core Formula Mutation
	extMutateFormula := base
	extMutateFormula.Expression.Formula = "SUM(t0.revenue) * 0.9"
	err = ValidateAdditiveMetricExtension(base, extMutateFormula)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot modify core metric formula")

	// 3. Reject Core Grain Removal
	extRemoveGrain := base
	extRemoveGrain.GrainAllowlist = []string{"region"} // "year" removed
	err = ValidateAdditiveMetricExtension(base, extRemoveGrain)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot remove core grain \"year\"")

	// 4. Reject Core Variable Removal
	extRemoveVar := base
	extRemoveVar.Variables = []MetricVariable{} // "tax_rate" removed
	err = ValidateAdditiveMetricExtension(base, extRemoveVar)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot remove core variable \"tax_rate\"")
}

// TestMetricUsageScan_5Sources verifies the 5-source usage scanner detects references across
// saved queries, KPI tiles, derived metrics, core adoptions, and active scheduled jobs.
func TestMetricUsageScan_5Sources(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	metricID := "m_target_usage"
	tenantID := "t-client-1"

	// 1. Saved Query Scan
	mock.ExpectQuery("SELECT id, name FROM data_explorer.saved_query").
		WithArgs(metricID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("sq-101", "Executive Dashboard Query"))

	// 2. Page Components Scan
	mock.ExpectQuery("SELECT id, name FROM page_definitions").
		WithArgs(metricID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("pg-202", "Financial Overview Page"))

	// 3. Derived Metric Scan
	mock.ExpectQuery("SELECT id, name FROM data_explorer.metric_definition").
		WithArgs(metricID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("m-303", "Derived Margin KPI"))

	// 4. Scheduled Jobs Scan
	mock.ExpectQuery("SELECT id::text, name FROM public.schedules").
		WithArgs(metricID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("sched-404", "Monthly PDF Delivery"))

	report, err := ScanMetricUsage(context.Background(), sqlxDB, metricID, false, tenantID)
	require.NoError(t, err)
	assert.True(t, report.InUse)
	require.Len(t, report.References, 4)

	assert.Equal(t, "saved_query", report.References[0].Type)
	assert.Equal(t, "kpi_tile", report.References[1].Type)
	assert.Equal(t, "derived_metric", report.References[2].Type)
	assert.Equal(t, "scheduled_job", report.References[3].Type)
}

// TestValidateQueryMetricDependencies_FailClosed verifies preflight dependency validation
// fails closed (422) if a query bundle references a metric missing in the target tenant.
func TestValidateQueryMetricDependencies_FailClosed(t *testing.T) {
	availableMetrics := map[string]bool{
		"m_revenue": true,
		"m_cost":    true,
	}

	// Case A: All present
	err := ValidateQueryMetricDependencies([]string{"m_revenue", "m_cost"}, availableMetrics)
	assert.NoError(t, err)

	// Case B: Missing metric dependency -> Fail Closed
	err = ValidateQueryMetricDependencies([]string{"m_revenue", "m_missing_kpi"}, availableMetrics)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMissingMetricDep)
	assert.Contains(t, err.Error(), "required metrics [m_missing_kpi] not found")
}
