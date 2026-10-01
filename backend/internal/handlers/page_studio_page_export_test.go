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

func TestPageStudioBundle_ExtractionAndRemapping(t *testing.T) {
	page := &PageStudioPage{
		ID:          uuid.New(),
		Name:        "Executive Dashboard",
		Slug:        "exec-dash",
		Description: "Executive overview dashboard",
		Layout:      json.RawMessage(`{"type":"grid","columns":12}`),
		Tabs:        json.RawMessage(`[{"id":"tab1","savedQueryId":"q-101"}]`),
		Components: json.RawMessage(`[
			{"id":"c1","type":"chart","config":{"savedQueryId":"q-101","title":"Pipeline"}},
			{"id":"c2","type":"table","config":{"queryId":"q-102","target":"/saved-queries/q-103"}}
		]`),
		DataSources: json.RawMessage(`{"sources":[{"savedQueryId":"q-104"}]}`),
		App: func() *json.RawMessage {
			raw := json.RawMessage(`{
				"variables": [],
				"queries": [{"id":"local-q1","savedQueryId":"q-105"}],
				"drillTargets": [{"target":"query:q-106"}]
			}`)
			return &raw
		}(),
	}

	// 1. Test extraction of all referenced query IDs
	extracted := ExtractAllReferencedQueryIDs(page)
	expectedSet := map[string]bool{
		"q-101": true,
		"q-102": true,
		"q-103": true,
		"q-104": true,
		"q-105": true,
		"q-106": true,
	}

	assert.Len(t, extracted, 6)
	for _, id := range extracted {
		assert.True(t, expectedSet[id], "unexpected query ID extracted: %s", id)
	}

	// 2. Test deep remapping of all query IDs
	idMap := map[string]string{
		"q-101": "new-q-101",
		"q-102": "new-q-102",
		"q-103": "new-q-103",
		"q-104": "new-q-104",
		"q-105": "new-q-105",
		"q-106": "new-q-106",
	}

	pageDef := PageBundleDefinition{
		Name:        page.Name,
		Slug:        page.Slug,
		Description: page.Description,
		Layout:      page.Layout,
		Tabs:        page.Tabs,
		Components:  page.Components,
		DataSources: page.DataSources,
		App:         page.App,
	}

	RemapPageQueryIDs(&pageDef, idMap)

	// Verify tabs remapped
	assert.Contains(t, string(pageDef.Tabs), `"savedQueryId":"new-q-101"`)
	assert.NotContains(t, string(pageDef.Tabs), `"q-101"`)

	// Verify components remapped (including savedQueryId, queryId, and target path)
	assert.Contains(t, string(pageDef.Components), `"savedQueryId":"new-q-101"`)
	assert.Contains(t, string(pageDef.Components), `"queryId":"new-q-102"`)
	assert.Contains(t, string(pageDef.Components), `"target":"/saved-queries/new-q-103"`)
	assert.NotContains(t, string(pageDef.Components), `"q-102"`)
	assert.NotContains(t, string(pageDef.Components), `"q-103"`)

	// Verify data sources remapped
	assert.Contains(t, string(pageDef.DataSources), `"savedQueryId":"new-q-104"`)

	// Verify app_model remapped (including query: target prefix)
	require.NotNil(t, pageDef.App)
	assert.Contains(t, string(*pageDef.App), `"savedQueryId":"new-q-105"`)
	assert.Contains(t, string(*pageDef.App), `"target":"query:new-q-106"`)
}

func TestPageStudioBundle_ExportRoundTrip(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	h := &PageStudioHandler{db: sqlxDB}

	tenantID := uuid.New()
	pageID := uuid.New()

	// 1. Mock page select
	mock.ExpectQuery(`FROM page_definitions WHERE id = \$1 AND tenant_id = \$2`).
		WithArgs(pageID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "name", "slug", "description",
			"layout", "tabs", "components", "data_sources",
			"presentation_events", "filter_bar", "app_model",
			"version", "is_core", "status", "created_at", "updated_at",
		}).AddRow(
			pageID, tenantID, "Sales Analytics", "sales-analytics", "Monthly sales metrics",
			[]byte(`{"type":"grid"}`), []byte(`[]`),
			[]byte(`[{"id":"c1","config":{"savedQueryId":"q-sales-1"}}]`),
			[]byte(`{}`), []byte(`[]`), []byte(`{}`),
			[]byte(`{"queries":[{"savedQueryId":"q-sales-1"}]}`),
			1, false, "published", time.Now(), time.Now(),
		))

	// 2. Mock query select for dependency bundling
	mock.ExpectQuery(`FROM data_explorer.saved_query WHERE id = \$1 AND \(tenant_id = \$2 OR is_core = true\)`).
		WithArgs("q-sales-1", tenantID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
			"chart_type", "query_state", "tags", "is_core",
		}).AddRow(
			"q-sales-1", tenantID.String(), "Monthly Revenue", "Revenue aggregate", "bo-crm-deal",
			nil, "{crm_account}", "bar",
			[]byte(`{"dimensions":[{"termNodeId":"deal_stage","alias":"Stage"}],"measures":[{"termNodeId":"deal_amount","alias":"Revenue","agg":"SUM"}]}`),
			"{sales,revenue}", false,
		))

	req := tenantRequest(http.MethodGet, "/page-studio/pages/"+pageID.String()+"/export", tenantID.String(), security.AuthInfo{UserID: "u1"}, "")
	rec := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Get("/page-studio/pages/{id}/export", h.exportPage)
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var bundle PageExportBundle
	err = json.Unmarshal(rec.Body.Bytes(), &bundle)
	require.NoError(t, err)

	assert.Equal(t, PageBundleSchemaVersion, bundle.SchemaVersion)
	assert.Equal(t, "Sales Analytics", bundle.Page.Name)
	assert.Equal(t, "sales-analytics", bundle.Page.Slug)
	assert.Len(t, bundle.Dependencies.Queries, 1)

	qItem := bundle.Dependencies.Queries[0]
	assert.Equal(t, "q-sales-1", qItem.ID)
	assert.Equal(t, "Monthly Revenue", qItem.Name)
	assert.Equal(t, "bo-crm-deal", qItem.BOID)
	assert.Contains(t, qItem.Dependencies.TermNodeIDs, "deal_stage")
	assert.Contains(t, qItem.Dependencies.TermNodeIDs, "deal_amount")
	assert.Equal(t, "bo-crm-deal.deal_stage", bundle.Dependencies.TermMappings["deal_stage"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPageStudioBundle_ImportLifecycle(t *testing.T) {
	t.Run("fresh import with new queries and deep remapping", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		h := &PageStudioHandler{db: sqlxDB}

		targetTenant := uuid.New()

		bundle := PageExportBundle{
			SchemaVersion: PageBundleSchemaVersion,
			ExportedAt:    time.Now().UTC(),
			SourceEnvironment: PageSourceEnvironment{
				TenantID: "source-tenant-1",
			},
			Page: PageBundleDefinition{
				Name:        "Executive Overview",
				Slug:        "exec-overview",
				Description: "Imported overview",
				Layout:      json.RawMessage(`{"type":"grid"}`),
				Tabs:        json.RawMessage(`[]`),
				Components:  json.RawMessage(`[{"id":"c1","type":"chart","config":{"savedQueryId":"src-q-1"}}]`),
				DataSources: json.RawMessage(`{}`),
			},
			Dependencies: PageBundleDependencies{
				Queries: []PageBundleQueryItem{
					{
						ID:           "src-q-1",
						Name:         "Pipeline Revenue",
						BOID:         "bo-deal",
						ChartType:    "bar",
						State:        json.RawMessage(`{"dimensions":[{"termNodeId":"stage"}],"measures":[{"termNodeId":"amount","agg":"SUM"}]}`),
						Tags:         []string{"crm"},
						Provenance:   PageBundleProvenance{IsCore: false, OriginalID: "src-q-1", ContentHash: "sha256:abc"},
						Dependencies: PageBundleQueryDependencies{BOID: "bo-deal", TermNodeIDs: []string{"stage", "amount"}},
					},
				},
			},
		}

		// 1. Check gold copy tenant
		mock.ExpectQuery(`uisce_gold_copy_tenant_id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldTenant))

		// 2. Query dedupe check (not found)
		mock.ExpectQuery(`SELECT id FROM data_explorer.saved_query`).
			WithArgs(targetTenant, "bo-deal", "bar", string(bundle.Dependencies.Queries[0].State)).
			WillReturnError(sql.ErrNoRows)

		// 3. Query name collision check (not found)
		mock.ExpectQuery(`SELECT id, is_core FROM data_explorer.saved_query WHERE tenant_id = \$1 AND name = \$2`).
			WithArgs(targetTenant, "Pipeline Revenue").
			WillReturnError(sql.ErrNoRows)

		// 4. Query insertion
		mock.ExpectExec(`INSERT INTO data_explorer.saved_query`).
			WithArgs(sqlmock.AnyArg(), targetTenant, targetTenant.String(), "Pipeline Revenue", "", "bo-deal",
				sqlmock.AnyArg(), sqlmock.AnyArg(), "bar", bundle.Dependencies.Queries[0].State, sqlmock.AnyArg(), false).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// 5. Page collision check (not found)
		mock.ExpectQuery(`SELECT id FROM page_definitions WHERE tenant_id = \$1 AND \(slug = \$2 OR name = \$3\)`).
			WithArgs(targetTenant, "exec-overview", "Executive Overview").
			WillReturnError(sql.ErrNoRows)

		// 6. Page insertion
		mock.ExpectExec(`INSERT INTO page_definitions`).
			WithArgs(sqlmock.AnyArg(), targetTenant, "Executive Overview", "exec-overview", "Imported overview",
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		body, _ := json.Marshal(ImportPageRequest{Bundle: bundle})
		req := tenantRequest(http.MethodPost, "/page-studio/pages/import", targetTenant.String(), security.AuthInfo{UserID: "u1"}, string(body))
		rec := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/page-studio/pages/import", h.importPage)
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var report ImportPageReport
		err = json.Unmarshal(rec.Body.Bytes(), &report)
		require.NoError(t, err)

		assert.Equal(t, "Executive Overview", report.PageName)
		assert.Equal(t, "exec-overview", report.PageSlug)
		assert.Equal(t, "created", report.Status)
		assert.Len(t, report.QueryResults, 1)
		assert.Equal(t, "imported", report.QueryResults[0].Status)
		assert.NotEmpty(t, report.QueryResults[0].NewID)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query dedupe skips insert and preserves existing ID in remapped page", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		h := &PageStudioHandler{db: sqlxDB}

		targetTenant := uuid.New()
		existingQueryID := "existing-q-99"

		bundle := PageExportBundle{
			SchemaVersion: PageBundleSchemaVersion,
			ExportedAt:    time.Now().UTC(),
			Page: PageBundleDefinition{
				Name:       "Overview",
				Slug:       "overview",
				Layout:     json.RawMessage(`{}`),
				Tabs:       json.RawMessage(`[]`),
				Components: json.RawMessage(`[{"config":{"savedQueryId":"src-q-1"}}]`),
			},
			Dependencies: PageBundleDependencies{
				Queries: []PageBundleQueryItem{
					{
						ID:        "src-q-1",
						Name:      "Deals",
						BOID:      "bo-deal",
						ChartType: "table",
						State:     json.RawMessage(`{"dimensions":[{"termNodeId":"stage"}]}`),
					},
				},
			},
		}

		mock.ExpectQuery(`uisce_gold_copy_tenant_id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldTenant))

		// Dedupe finds matching existing query!
		mock.ExpectQuery(`SELECT id FROM data_explorer.saved_query`).
			WithArgs(targetTenant, "bo-deal", "table", string(bundle.Dependencies.Queries[0].State)).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(existingQueryID))

		// Page collision check
		mock.ExpectQuery(`SELECT id FROM page_definitions WHERE tenant_id = \$1 AND \(slug = \$2 OR name = \$3\)`).
			WithArgs(targetTenant, "overview", "Overview").
			WillReturnError(sql.ErrNoRows)

		// Page insertion (assert components will have remapped existingQueryID!)
		mock.ExpectExec(`INSERT INTO page_definitions`).
			WithArgs(sqlmock.AnyArg(), targetTenant, "Overview", "overview", "",
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		body, _ := json.Marshal(ImportPageRequest{Bundle: bundle})
		req := tenantRequest(http.MethodPost, "/page-studio/pages/import", targetTenant.String(), security.AuthInfo{UserID: "u1"}, string(body))
		rec := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/page-studio/pages/import", h.importPage)
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var report ImportPageReport
		err = json.Unmarshal(rec.Body.Bytes(), &report)
		require.NoError(t, err)

		assert.Len(t, report.QueryResults, 1)
		assert.Equal(t, "skipped", report.QueryResults[0].Status)
		assert.Equal(t, existingQueryID, report.QueryResults[0].NewID)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("core query rebinds to gold copy core query with status rebound_core", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		h := &PageStudioHandler{db: sqlxDB}

		targetTenant := uuid.New()
		goldCoreQueryID := "gold-core-q-88"

		bundle := PageExportBundle{
			SchemaVersion: PageBundleSchemaVersion,
			ExportedAt:    time.Now().UTC(),
			Page: PageBundleDefinition{
				Name:       "Core Overview",
				Slug:       "core-overview",
				Layout:     json.RawMessage(`{}`),
				Tabs:       json.RawMessage(`[]`),
				Components: json.RawMessage(`[{"config":{"savedQueryId":"ext-core-q"}}]`),
			},
			Dependencies: PageBundleDependencies{
				Queries: []PageBundleQueryItem{
					{
						ID:        "ext-core-q",
						Name:      "Core Revenue Metric",
						BOID:      "bo-account",
						ChartType: "bar",
						State:     json.RawMessage(`{"dimensions":[{"termNodeId":"d1"}]}`),
						Provenance: PageBundleProvenance{
							IsCore:      true,
							OriginalID:  "ext-core-q",
							ContentHash: "sha256:core123",
						},
					},
				},
			},
		}

		mock.ExpectQuery(`uisce_gold_copy_tenant_id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldTenant))

		// Dedupe check in target tenant (not found locally)
		mock.ExpectQuery(`SELECT id FROM data_explorer.saved_query`).
			WithArgs(targetTenant, "bo-account", "bar", string(bundle.Dependencies.Queries[0].State)).
			WillReturnError(sql.ErrNoRows)

		// Core check in goldTenant (found!)
		mock.ExpectQuery(`SELECT id FROM data_explorer.saved_query WHERE tenant_id = \$1 AND is_core = true`).
			WithArgs(goldTenant, "ext-core-q", "Core Revenue Metric").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldCoreQueryID))

		// Page collision check
		mock.ExpectQuery(`SELECT id FROM page_definitions WHERE tenant_id = \$1 AND \(slug = \$2 OR name = \$3\)`).
			WithArgs(targetTenant, "core-overview", "Core Overview").
			WillReturnError(sql.ErrNoRows)

		// Page insertion
		mock.ExpectExec(`INSERT INTO page_definitions`).
			WithArgs(sqlmock.AnyArg(), targetTenant, "Core Overview", "core-overview", "",
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		body, _ := json.Marshal(ImportPageRequest{Bundle: bundle})
		req := tenantRequest(http.MethodPost, "/page-studio/pages/import", targetTenant.String(), security.AuthInfo{UserID: "u1"}, string(body))
		rec := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/page-studio/pages/import", h.importPage)
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var report ImportPageReport
		err = json.Unmarshal(rec.Body.Bytes(), &report)
		require.NoError(t, err)

		assert.Len(t, report.QueryResults, 1)
		assert.Equal(t, "rebound_core", report.QueryResults[0].Status)
		assert.Equal(t, goldCoreQueryID, report.QueryResults[0].NewID)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
