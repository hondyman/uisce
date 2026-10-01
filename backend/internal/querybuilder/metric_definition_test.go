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

func TestMetricDefinition_ValidateGrains(t *testing.T) {
	allowlist := []string{"Region", "TradingDesk", "AssetClass"}

	// 1. Valid grains succeed
	err := ValidateGrains([]string{"Region", "TradingDesk"}, allowlist)
	assert.NoError(t, err)

	// Case-insensitive matching
	err = ValidateGrains([]string{"region", "assetclass"}, allowlist)
	assert.NoError(t, err)

	// 2. Unlisted grain fails with ErrGrainNotAllowed
	err = ValidateGrains([]string{"Region", "AccountID"}, allowlist)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGrainNotAllowed)
	assert.Contains(t, err.Error(), "AccountID")

	// 3. Empty allowlist allows all grains
	err = ValidateGrains([]string{"Anything", "AccountID"}, []string{})
	assert.NoError(t, err)
}

func TestMetricDefinition_ScanMetricUsage(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	metricID := "m-revenue-ytd"
	tenantID := "tenant-alpha"

	// 1. Mock saved query reference scan
	mock.ExpectQuery(`SELECT id, name FROM data_explorer\.saved_query WHERE query_state::text LIKE .* AND archived_at IS NULL AND tenant_id = \$2`).
		WithArgs(metricID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow("sq-exec-1", "Executive Monthly Dashboard Query"))

	// 2. Mock page component reference scan
	mock.ExpectQuery(`SELECT id, name FROM page_definitions WHERE components::text LIKE .* AND tenant_id = \$2`).
		WithArgs(metricID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow("pg-kpi-1", "Live Portfolio Overview"))

	// 3. Mock derived metric reference scan
	mock.ExpectQuery(`SELECT id, name FROM data_explorer\.metric_definition WHERE id != \$1 AND expression::text LIKE .* AND archived_at IS NULL AND tenant_id = \$2`).
		WithArgs(metricID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow("m-net-margin", "Net Profit Margin Metric"))

	report, err := ScanMetricUsage(context.Background(), sqlxDB, metricID, false, tenantID)
	require.NoError(t, err)

	assert.True(t, report.InUse)
	require.Len(t, report.References, 3)

	refTypes := map[string]bool{}
	for _, r := range report.References {
		refTypes[r.Type] = true
	}

	assert.True(t, refTypes["saved_query"], "must find saved query reference")
	assert.True(t, refTypes["kpi_tile"], "must find kpi tile component reference")
	assert.True(t, refTypes["derived_metric"], "must find derived metric dependency reference")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMetricDefinition_CoreAdoptionScan(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	coreMetricID := "core-m-aum"

	// 1. Saved query scan
	mock.ExpectQuery(`SELECT id, name FROM data_explorer\.saved_query WHERE query_state::text LIKE .* AND archived_at IS NULL`).
		WithArgs(coreMetricID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))

	// 2. Page component scan
	mock.ExpectQuery(`SELECT id, name FROM page_definitions WHERE components::text LIKE .*`).
		WithArgs(coreMetricID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))

	// 3. Derived metric scan
	mock.ExpectQuery(`SELECT id, name FROM data_explorer\.metric_definition WHERE id != \$1 AND expression::text LIKE .* AND archived_at IS NULL`).
		WithArgs(coreMetricID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))

	// 4. Core adoption scan with object_type = 'metric'
	mock.ExpectQuery(`SELECT tenant_id, COUNT\(\*\) as count FROM public\.core_object_adoption WHERE object_type = 'metric' AND core_object_id = \$1 GROUP BY tenant_id`).
		WithArgs(coreMetricID).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "count"}).
			AddRow("client-tenant-x", 4))

	report, err := ScanMetricUsage(context.Background(), sqlxDB, coreMetricID, true, "")
	require.NoError(t, err)

	assert.True(t, report.InUse)
	require.Len(t, report.References, 1)
	assert.Equal(t, "core_adoption", report.References[0].Type)
	assert.Equal(t, "client-tenant-x", report.References[0].ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMetricDefinition_ExportBundle(t *testing.T) {
	precision := 2
	symbol := "$"
	metrics := []MetricDefinition{
		{
			ID:          "m-rev-01",
			TenantID:    "tenant-1",
			Name:        "Total Revenue",
			Description: "Aggregated gross sales revenue",
			BOID:        "bo-account",
			Expression: MetricExpression{
				Kind:       "aggregation",
				Fn:         "sum",
				TermNodeID: "meas-sales-amount",
			},
			GrainAllowlist: []string{"region", "product_category"},
			FormatConfig: MetricFormatConfig{
				Type:           "currency",
				Precision:      &precision,
				CurrencySymbol: &symbol,
			},
			Tags:   []string{"Finance", "Sales"},
			IsCore: true,
			Status: "active",
		},
	}

	bundle, err := ExportMetricBundle(metrics)
	require.NoError(t, err)

	assert.Equal(t, MetricBundleSchemaVersion, bundle.SchemaVersion)
	assert.WithinDuration(t, time.Now().UTC(), bundle.ExportedAt, 2*time.Second)
	assert.Len(t, bundle.Metrics, 1)
	assert.Equal(t, "Total Revenue", bundle.Metrics[0].Name)
	assert.Equal(t, []string{"region", "product_category"}, bundle.Metrics[0].GrainAllowlist)
}
