package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTemplateDistribution_CoreRebind tests that a template packaged as uisce.page-bundle/1
// containing references to core queries is imported into a client tenant and cleanly rebinds
// to the client's local gold-copy core queries with status "rebound_core".
func TestTemplateDistribution_CoreRebind(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	h := &PageStudioHandler{db: sqlxDB}

	clientTenant := uuid.New()
	goldTenant := uuid.New()
	goldCoreQueryID := "gold-kpi-query-99"

	// Template bundle distributed across environments
	templateBundle := PageBundle{
		SchemaVersion: PageBundleSchemaVersion,
		ExportedAt:    time.Now().UTC(),
		SourceEnvironment: PageSourceEnvironment{
			TenantID: goldTenant.String(),
			Product:  "uisce-templates",
		},
		Page: PageBundleDefinition{
			Name:        "Executive KPI Overview Template",
			Slug:        "exec-kpi-template",
			Description: "Standard grid report with executive metrics and slicers",
			Layout:      json.RawMessage(`{"type":"grid","columns":12}`),
			Tabs:        json.RawMessage(`[]`),
			Components: json.RawMessage(`[
				{"id":"param-bar","type":"report_parameter_bar","config":{"variables":["v_region","v_status"]}},
				{"id":"kpi-1","type":"kpi_tile","config":{"savedQueryId":"core-kpi-rev","title":"Total Revenue"}},
				{"id":"chart-1","type":"saved_query_widget","config":{"savedQueryId":"core-kpi-rev","chartType":"bar"}}
			]`),
			DataSources: json.RawMessage(`{}`),
		},
		Dependencies: PageBundleDependencies{
			Queries: []PageBundleQueryItem{
				{
					ID:        "core-kpi-rev",
					Name:      "Core Revenue KPI",
					BOID:      "bo-account",
					ChartType: "kpi",
					State:     json.RawMessage(`{"dimensions":[],"measures":[{"termNodeId":"meas-revenue","alias":"total_revenue"}]}`),
					Provenance: PageBundleProvenance{
						IsCore:      true,
						OriginalID:  "core-kpi-rev",
						ContentHash: "sha256:core_rev_kpi_hash",
					},
					Dependencies: PageBundleQueryDependencies{
						BOID:        "bo-account",
						TermNodeIDs: []string{"meas-revenue"},
					},
				},
			},
		},
	}

	// 1. Resolve gold copy tenant for the target client
	mock.ExpectQuery(`uisce_gold_copy_tenant_id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldTenant))

	// 2. Client tenant deduplication check (not present in client's own custom queries)
	mock.ExpectQuery(`SELECT id FROM data_explorer.saved_query`).
		WithArgs(clientTenant, "bo-account", "kpi", string(templateBundle.Dependencies.Queries[0].State)).
		WillReturnError(sql.ErrNoRows)

	// 3. Core query resolution in gold-copy master catalog (found!)
	mock.ExpectQuery(`SELECT id FROM data_explorer.saved_query WHERE tenant_id = \$1 AND is_core = true`).
		WithArgs(goldTenant, "core-kpi-rev", "Core Revenue KPI").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldCoreQueryID))

	// 4. Page slug / name collision check in client tenant
	mock.ExpectQuery(`SELECT id FROM page_definitions WHERE tenant_id = \$1 AND \(slug = \$2 OR name = \$3\)`).
		WithArgs(clientTenant, "exec-kpi-template", "Executive KPI Overview Template").
		WillReturnError(sql.ErrNoRows)

	// 5. Page definition insertion in client tenant with remapped query IDs
	mock.ExpectExec(`INSERT INTO page_definitions`).
		WithArgs(
			sqlmock.AnyArg(),
			clientTenant,
			"Executive KPI Overview Template",
			"exec-kpi-template",
			"Standard grid report with executive metrics and slicers",
			sqlmock.AnyArg(), // layout
			sqlmock.AnyArg(), // tabs
			sqlmock.AnyArg(), // components (contains remapped goldCoreQueryID!)
			sqlmock.AnyArg(), // data_sources
			sqlmock.AnyArg(), // presentation_events
			sqlmock.AnyArg(), // filter_bar
			sqlmock.AnyArg(), // app_model
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body, err := json.Marshal(ImportPageRequest{Bundle: templateBundle})
	require.NoError(t, err)

	req := tenantRequest(http.MethodPost, "/page-studio/pages/import", clientTenant.String(), security.AuthInfo{UserID: "client-admin-1"}, string(body))
	rec := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Post("/page-studio/pages/import", h.importPage)
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var report ImportPageReport
	err = json.Unmarshal(rec.Body.Bytes(), &report)
	require.NoError(t, err)

	assert.Equal(t, "created", report.Status)
	assert.Equal(t, "Executive KPI Overview Template", report.PageName)
	assert.Len(t, report.QueryResults, 1)

	// Explicit assertion on "rebound_core" status and remapped query ID
	assert.Equal(t, "rebound_core", report.QueryResults[0].Status)
	assert.Equal(t, "core-kpi-rev", report.QueryResults[0].OriginalID)
	assert.Equal(t, goldCoreQueryID, report.QueryResults[0].NewID)
	assert.Equal(t, "Core Revenue KPI", report.QueryResults[0].Name)

	assert.NoError(t, mock.ExpectationsWereMet())
}
