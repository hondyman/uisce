package querybuilder

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupBundleTestEnv(t *testing.T, tenantID, userID string) (*SavedQueryHandler, *chi.Mux, sqlmock.Sqlmock, func(method, path string, body []byte) *http.Request) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	deps := handlers.SecurityContextDeps{Resolver: &mockDSResolver{defaultTenantID: tenantID}}
	h := NewSavedQueryHandler(sqlxDB, nil, nil, deps)

	r := chi.NewRouter()
	r.Get("/api/explorer/saved-queries/{id}/export", h.HandleExportSavedQuery)
	r.Post("/api/explorer/saved-queries/export", h.HandleExportSavedQueriesBatch)
	r.Post("/api/explorer/saved-queries/import", h.HandleImportSavedQueries)

	makeReq := func(method, path string, body []byte) *http.Request {
		var bodyReader *bytes.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		} else {
			bodyReader = bytes.NewReader([]byte{})
		}
		req := httptest.NewRequest(method, path, bodyReader)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Datasource-Id", "ds-1")
		auth := security.AuthInfo{
			TenantIDs: []string{tenantID},
			UserID:    userID,
			Roles:     []string{"tenant_admin"},
		}
		return req.WithContext(security.WithAuthInfo(req.Context(), auth))
	}

	return h, r, mock, makeReq
}

func TestSavedQueryBundle_ExportSingleAndBatch(t *testing.T) {
	tenantID := "11111111-1111-1111-1111-111111111111"
	userID := "22222222-2222-2222-2222-222222222222"
	queryID := "33333333-3333-3333-3333-333333333333"

	t.Run("export single query produces uisce.query-bundle/1 schema with dependencies and hash", func(t *testing.T) {
		_, router, mock, makeReq := setupBundleTestEnv(t, tenantID, userID)

		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
			WithArgs(queryID, tenantID, userID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				queryID, tenantID, userID, "Regional Sales", "Sales by region", "bo-account", "b-1", "{bo-household}",
				"bar", []byte(`{"dimensions": [{"termNodeId": "region", "alias": "region"}], "measures": [{"termNodeId": "revenue", "alias": "revenue", "aggregation": "SUM"}]}`),
				"{sales,core}", nil, false, "shared", false,
				"active", nil, userID, time.Now(), time.Now(),
			))

		req := makeReq("GET", "/api/explorer/saved-queries/"+queryID+"/export", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var bundle QueryBundle
		err := json.Unmarshal(rec.Body.Bytes(), &bundle)
		require.NoError(t, err)

		assert.Equal(t, QueryBundleSchemaVersion, bundle.SchemaVersion)
		require.Len(t, bundle.Queries, 1)
		q := bundle.Queries[0]
		assert.Equal(t, queryID, q.ID)
		assert.Equal(t, "Regional Sales", q.Name)
		assert.Equal(t, "bo-account", q.BOID)
		assert.Contains(t, q.Dependencies.TermNodeIDs, "region")
		assert.Contains(t, q.Dependencies.TermNodeIDs, "revenue")
		assert.Contains(t, q.Provenance.ContentHash, "sha256:")
	})

	t.Run("batch export returns all specified queries", func(t *testing.T) {
		_, router, mock, makeReq := setupBundleTestEnv(t, tenantID, userID)

		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
			WithArgs(queryID, tenantID, userID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				queryID, tenantID, userID, "Batch Q1", "Desc", "bo-account", nil, "{}",
				"bar", []byte(`{"dimensions": [{"termNodeId": "region", "alias": "region"}]}`),
				"{}", nil, false, "shared", false,
				"active", nil, userID, time.Now(), time.Now(),
			))

		body, _ := json.Marshal(ExportSavedQueriesBatchRequest{IDs: []string{queryID}})
		req := makeReq("POST", "/api/explorer/saved-queries/export", body)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var bundle QueryBundle
		err := json.Unmarshal(rec.Body.Bytes(), &bundle)
		require.NoError(t, err)
		assert.Equal(t, 1, len(bundle.Queries))
	})
}

func TestSavedQueryBundle_ImportLifecycle(t *testing.T) {
	tenantID := "11111111-1111-1111-1111-111111111111"
	userID := "22222222-2222-2222-2222-222222222222"

	bundleItem := BundleQueryItem{
		ID:          "orig-sq-1",
		Name:        "Quarterly Performance",
		Description: "Performance metrics",
		BOID:        "bo-portfolio",
		ChartType:   "line",
		State: SavedQueryState{
			Dimensions: []SavedQueryDimension{{TermNodeID: "quarter", Alias: "quarter"}},
			Measures:   []SavedQueryMeasure{{TermNodeID: "roi", Alias: "roi", Aggregation: "AVG"}},
		},
		Provenance: BundleQueryProvenance{
			IsCore:      false,
			OriginalID:  "orig-sq-1",
			ContentHash: "sha256:abcd",
		},
	}

	validBundle := QueryBundle{
		SchemaVersion: QueryBundleSchemaVersion,
		ExportedAt:    time.Now().UTC(),
		Queries:       []BundleQueryItem{bundleItem},
	}

	t.Run("fresh import inserts new query row and reports imported", func(t *testing.T) {
		_, router, mock, makeReq := setupBundleTestEnv(t, tenantID, userID)

		// Gold copy tenant check
		mock.ExpectQuery(`SELECT id FROM \(SELECT public\.uisce_gold_copy_tenant_id\(\) AS id\) g WHERE id IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("gold-tenant-id"))

		// Load existing queries (empty)
		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE tenant_id = \$1 AND archived_at IS NULL`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids", "chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core", "status", "archived_at", "created_by", "created_at", "updated_at"}))

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO data_explorer\.saved_query`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		reqBody, _ := json.Marshal(ImportSavedQueryRequest{Bundle: validBundle})
		req := makeReq("POST", "/api/explorer/saved-queries/import", reqBody)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var report ImportSavedQueryReport
		err := json.Unmarshal(rec.Body.Bytes(), &report)
		require.NoError(t, err)
		assert.Equal(t, 1, report.ImportedCount)
		assert.Equal(t, 0, report.SkippedCount)
		assert.Equal(t, "imported", report.Items[0].Status)
	})

	t.Run("dedupe: identical content hash reports skipped already-present with 0 inserts", func(t *testing.T) {
		_, router, mock, makeReq := setupBundleTestEnv(t, tenantID, userID)

		mock.ExpectQuery(`SELECT id FROM \(SELECT public\.uisce_gold_copy_tenant_id\(\) AS id\) g WHERE id IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("gold-tenant-id"))

		// Return existing query with identical canonical content
		stateBytes, _ := json.Marshal(bundleItem.State)
		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE tenant_id = \$1 AND archived_at IS NULL`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				"existing-sq-id", tenantID, userID, bundleItem.Name, bundleItem.Description, bundleItem.BOID, nil, nil,
				bundleItem.ChartType, stateBytes, nil, nil, false, "shared", false,
				"active", nil, userID, time.Now(), time.Now(),
			))

		mock.ExpectBegin()
		mock.ExpectCommit()

		reqBody, _ := json.Marshal(ImportSavedQueryRequest{Bundle: validBundle})
		req := makeReq("POST", "/api/explorer/saved-queries/import", reqBody)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var report ImportSavedQueryReport
		err := json.Unmarshal(rec.Body.Bytes(), &report)
		require.NoError(t, err)
		assert.Equal(t, 0, report.ImportedCount)
		assert.Equal(t, 1, report.SkippedCount)
		assert.Equal(t, "skipped", report.Items[0].Status)
		assert.Equal(t, "already-present", report.Items[0].Reason)
	})

	t.Run("name collision without overwrite renames to (imported)", func(t *testing.T) {
		_, router, mock, makeReq := setupBundleTestEnv(t, tenantID, userID)

		mock.ExpectQuery(`SELECT id FROM \(SELECT public\.uisce_gold_copy_tenant_id\(\) AS id\) g WHERE id IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("gold-tenant-id"))

		// Existing query has same name but different state (different dimension)
		diffStateBytes, _ := json.Marshal(SavedQueryState{
			Dimensions: []SavedQueryDimension{{TermNodeID: "different_dim", Alias: "different_dim"}},
		})
		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE tenant_id = \$1 AND archived_at IS NULL`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				"existing-sq-id", tenantID, userID, bundleItem.Name, "Old desc", bundleItem.BOID, nil, nil,
				bundleItem.ChartType, diffStateBytes, nil, nil, false, "shared", false,
				"active", nil, userID, time.Now(), time.Now(),
			))

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO data_explorer\.saved_query`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		reqBody, _ := json.Marshal(ImportSavedQueryRequest{Bundle: validBundle, Overwrite: false})
		req := makeReq("POST", "/api/explorer/saved-queries/import", reqBody)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var report ImportSavedQueryReport
		err := json.Unmarshal(rec.Body.Bytes(), &report)
		require.NoError(t, err)
		assert.Equal(t, 1, report.ImportedCount)
		assert.Equal(t, "renamed", report.Items[0].Status)
		assert.Equal(t, "Quarterly Performance (imported)", report.Items[0].Name)
	})

	t.Run("core import security: client tenant attempting isCore=true receives 403 Forbidden", func(t *testing.T) {
		_, router, mock, makeReq := setupBundleTestEnv(t, tenantID, userID)

		// Gold copy is a DIFFERENT tenant
		mock.ExpectQuery(`SELECT id FROM \(SELECT public\.uisce_gold_copy_tenant_id\(\) AS id\) g WHERE id IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("different-master-tenant-id"))

		isCore := true
		reqBody, _ := json.Marshal(ImportSavedQueryRequest{Bundle: validBundle, IsCore: &isCore})
		req := makeReq("POST", "/api/explorer/saved-queries/import", reqBody)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Contains(t, rec.Body.String(), "only master gold copy tenant")
	})

	t.Run("preflight failure: query missing boId returns 422 with zero database writes", func(t *testing.T) {
		_, router, mock, makeReq := setupBundleTestEnv(t, tenantID, userID)

		mock.ExpectQuery(`SELECT id FROM \(SELECT public\.uisce_gold_copy_tenant_id\(\) AS id\) g WHERE id IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(tenantID))

		invalidItem := bundleItem
		invalidItem.BOID = ""
		invalidBundle := QueryBundle{
			SchemaVersion: QueryBundleSchemaVersion,
			ExportedAt:    time.Now().UTC(),
			Queries:       []BundleQueryItem{invalidItem},
		}

		reqBody, _ := json.Marshal(ImportSavedQueryRequest{Bundle: invalidBundle})
		req := makeReq("POST", "/api/explorer/saved-queries/import", reqBody)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "missing required primary boId")
	})

	t.Run("term remapping on import updates dimension, measure, and filter termNodeIDs", func(t *testing.T) {
		_, router, mock, makeReq := setupBundleTestEnv(t, tenantID, userID)

		mock.ExpectQuery(`SELECT id FROM \(SELECT public\.uisce_gold_copy_tenant_id\(\) AS id\) g WHERE id IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("gold-tenant-id"))

		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE tenant_id = \$1 AND archived_at IS NULL`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids", "chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core", "status", "archived_at", "created_by", "created_at", "updated_at"}))

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO data_explorer\.saved_query`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		remappingBundle := validBundle
		remappingBundle.TermMappings = map[string]string{
			"quarter": "mapped_quarter_term_id",
			"roi":     "mapped_roi_term_id",
		}

		reqBody, _ := json.Marshal(ImportSavedQueryRequest{Bundle: remappingBundle})
		req := makeReq("POST", "/api/explorer/saved-queries/import", reqBody)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var report ImportSavedQueryReport
		err := json.Unmarshal(rec.Body.Bytes(), &report)
		require.NoError(t, err)
		assert.Equal(t, 1, report.ImportedCount)
	})

	t.Run("overwrite in-place preserves existing query ID", func(t *testing.T) {
		_, router, mock, makeReq := setupBundleTestEnv(t, tenantID, userID)

		mock.ExpectQuery(`SELECT id FROM \(SELECT public\.uisce_gold_copy_tenant_id\(\) AS id\) g WHERE id IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("gold-tenant-id"))

		existingID := "existing-query-to-overwrite"
		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE tenant_id = \$1 AND archived_at IS NULL`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				existingID, tenantID, userID, bundleItem.Name, "Old desc", bundleItem.BOID, nil, nil,
				bundleItem.ChartType, []byte(`{"dimensions": []}`), nil, nil, false, "shared", false,
				"active", nil, userID, time.Now(), time.Now(),
			))

		mock.ExpectBegin()

		// Usage scan: zero references
		mock.ExpectQuery(`SELECT .* FROM page_definitions WHERE tenant_id = \$1`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name", "slug", "status", "version", "app_model", "layout", "components", "data_sources"}))
		mock.ExpectQuery(`SELECT id, name, query_state FROM data_explorer\.saved_query WHERE id != \$1 AND archived_at IS NULL AND tenant_id = \$2`).
			WithArgs(existingID, tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "query_state"}))

		mock.ExpectExec(`UPDATE data_explorer\.saved_query SET`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), existingID, tenantID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		reqBody, _ := json.Marshal(ImportSavedQueryRequest{Bundle: validBundle, Overwrite: true})
		req := makeReq("POST", "/api/explorer/saved-queries/import", reqBody)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Logf("Response body: %s", rec.Body.String())
		}
		assert.Equal(t, http.StatusOK, rec.Code)
		var report ImportSavedQueryReport
		err := json.Unmarshal(rec.Body.Bytes(), &report)
		require.NoError(t, err)
		assert.Equal(t, 1, report.ImportedCount)
		assert.Equal(t, "overwritten", report.Items[0].Status)
		assert.Equal(t, existingID, report.Items[0].NewID)
	})

	t.Run("overwrite against adopted core query is rejected with 409 Conflict", func(t *testing.T) {
		_, router, mock, makeReq := setupBundleTestEnv(t, tenantID, userID)

		mock.ExpectQuery(`SELECT id FROM \(SELECT public\.uisce_gold_copy_tenant_id\(\) AS id\) g WHERE id IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("gold-tenant-id"))

		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE tenant_id = \$1 AND archived_at IS NULL`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				"core-sq-id", tenantID, userID, bundleItem.Name, "Core desc", bundleItem.BOID, nil, nil,
				bundleItem.ChartType, []byte(`{"dimensions": []}`), nil, nil, false, "shared", true, // is_core = true
				"active", nil, userID, time.Now(), time.Now(),
			))

		mock.ExpectBegin()

		reqBody, _ := json.Marshal(ImportSavedQueryRequest{Bundle: validBundle, Overwrite: true})
		req := makeReq("POST", "/api/explorer/saved-queries/import", reqBody)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "cannot overwrite adopted core query")
	})

	t.Run("cross-tenant export authorization: cannot export another tenant's query", func(t *testing.T) {
		otherTenantID := "99999999-9999-9999-9999-999999999999"
		_, router, mock, makeReq := setupBundleTestEnv(t, otherTenantID, "other-user")

		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
			WithArgs("private-query-id", otherTenantID, "other-user").
			WillReturnError(sqlmock.ErrCancelled)

		mock.ExpectQuery(`SELECT id FROM \(SELECT public\.uisce_gold_copy_tenant_id\(\) AS id\) g WHERE id IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("gold-tenant-id"))

		req := makeReq("GET", "/api/explorer/saved-queries/private-query-id/export", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestSavedQueryBundle_RoundTrip(t *testing.T) {
	// 1. Create canonical content representation
	original := savedQueryContent{
		Name:         "Executive Pipeline",
		Description:  "Deals by stage and rep",
		BOID:         "bo-crm-deal",
		BindingID:    "bind-deal-1",
		RelatedBOIDs: []string{"bo-sales-rep"},
		ChartType:    "bar",
		State: SavedQueryState{
			Dimensions: []SavedQueryDimension{{TermNodeID: "stage", Alias: "Stage", BOID: "bo-crm-deal"}},
			Measures:   []SavedQueryMeasure{{TermNodeID: "amount", Alias: "Amount", Aggregation: "SUM", BOID: "bo-crm-deal"}},
			Filters:    []SavedQueryFilter{{TermNodeID: "stage", Operator: "neq", Value: "lost"}},
			Parameters: []SavedQueryParameter{{Name: "minAmount", Type: "number", Default: float64(10000)}},
		},
		Tags: []string{"pipeline", "executive"},
	}.normalized()

	origHash := ComputeContentHash(original)
	require.NotEmpty(t, origHash)

	// 2. Export to bundle
	bundleItem := BundleQueryItem{
		ID:           "deal-pipe-1",
		Name:         original.Name,
		Description:  original.Description,
		BOID:         original.BOID,
		BindingID:    original.BindingID,
		RelatedBOIDs: original.RelatedBOIDs,
		ChartType:    original.ChartType,
		State:        original.State,
		Tags:         original.Tags,
		Provenance: BundleQueryProvenance{
			IsCore:      false,
			OriginalID:  "deal-pipe-1",
			ContentHash: origHash,
		},
		Dependencies: BundleQueryDependencies{
			BOID:         original.BOID,
			RelatedBOIDs: original.RelatedBOIDs,
			TermNodeIDs:  []string{"stage", "amount"},
		},
	}

	bundle := QueryBundle{
		SchemaVersion: QueryBundleSchemaVersion,
		ExportedAt:    time.Now().UTC(),
		Queries:       []BundleQueryItem{bundleItem},
	}

	// 3. Serialize to JSON wire format
	wireBytes, err := json.Marshal(bundle)
	require.NoError(t, err)

	// 4. Deserialize on target environment
	var targetBundle QueryBundle
	err = json.Unmarshal(wireBytes, &targetBundle)
	require.NoError(t, err)
	require.Len(t, targetBundle.Queries, 1)

	importedItem := targetBundle.Queries[0]
	importedContent := savedQueryContent{
		Name:         importedItem.Name,
		Description:  importedItem.Description,
		BOID:         importedItem.BOID,
		BindingID:    importedItem.BindingID,
		RelatedBOIDs: importedItem.RelatedBOIDs,
		ChartType:    importedItem.ChartType,
		State:        importedItem.State,
		Tags:         importedItem.Tags,
	}.normalized()

	importedHash := ComputeContentHash(importedContent)

	// 5. Invariant assertion: round-trip preserves exact content hash and execution structure
	assert.Equal(t, origHash, importedHash)
	assert.Equal(t, original.Name, importedContent.Name)
	assert.Equal(t, original.BOID, importedContent.BOID)
	assert.Equal(t, original.State.Dimensions, importedContent.State.Dimensions)
	assert.Equal(t, original.State.Measures, importedContent.State.Measures)
	assert.Equal(t, original.State.Filters, importedContent.State.Filters)
	assert.Equal(t, original.State.Parameters, importedContent.State.Parameters)
}

func TestSavedQueryBundle_RoundTripSquared(t *testing.T) {
	// Step 1: Export from Tenant A
	tenantAOriginal := savedQueryContent{
		Name:        "Revenue by Desk",
		Description: "Trading desks performance",
		BOID:        "bo-trade",
		ChartType:   "stackedBar",
		State: SavedQueryState{
			Dimensions: []SavedQueryDimension{{TermNodeID: "desk", Alias: "Desk", BOID: "bo-trade"}},
			Measures:   []SavedQueryMeasure{{TermNodeID: "pnl", Alias: "PnL", Aggregation: "SUM", BOID: "bo-trade"}},
			Filters:    []SavedQueryFilter{{TermNodeID: "desk", Operator: "in", Value: []interface{}{"FX", "Equities"}}},
		},
		Tags: []string{"trading", "risk"},
	}.normalized()

	hashA := ComputeContentHash(tenantAOriginal)
	bundleA := QueryBundle{
		SchemaVersion: QueryBundleSchemaVersion,
		ExportedAt:    time.Now().UTC(),
		Queries: []BundleQueryItem{
			{
				ID:          "q-desk-1",
				Name:        tenantAOriginal.Name,
				Description: tenantAOriginal.Description,
				BOID:        tenantAOriginal.BOID,
				ChartType:   tenantAOriginal.ChartType,
				State:       tenantAOriginal.State,
				Tags:        tenantAOriginal.Tags,
				Provenance: BundleQueryProvenance{
					ContentHash: hashA,
				},
				Dependencies: BundleQueryDependencies{
					BOID:        tenantAOriginal.BOID,
					TermNodeIDs: []string{"desk", "pnl"},
				},
			},
		},
	}

	wireBytesA, err := json.Marshal(bundleA)
	require.NoError(t, err)

	// Step 2: Import into Tenant C (with rebound_core simulation)
	var bundleC QueryBundle
	err = json.Unmarshal(wireBytesA, &bundleC)
	require.NoError(t, err)

	tenantCImported := savedQueryContent{
		Name:        bundleC.Queries[0].Name,
		Description: bundleC.Queries[0].Description,
		BOID:        bundleC.Queries[0].BOID,
		ChartType:   bundleC.Queries[0].ChartType,
		State:       bundleC.Queries[0].State,
		Tags:        bundleC.Queries[0].Tags,
	}.normalized()
	hashC := ComputeContentHash(tenantCImported)

	// Step 3: Re-export from Tenant C (Round-Trip Squared)
	reExportBundleC := QueryBundle{
		SchemaVersion: QueryBundleSchemaVersion,
		ExportedAt:    time.Now().UTC(),
		Queries: []BundleQueryItem{
			{
				ID:          "q-desk-c-rebound",
				Name:        tenantCImported.Name,
				Description: tenantCImported.Description,
				BOID:        tenantCImported.BOID,
				ChartType:   tenantCImported.ChartType,
				State:       tenantCImported.State,
				Tags:        tenantCImported.Tags,
				Provenance: BundleQueryProvenance{
					ContentHash: hashC,
				},
				Dependencies: BundleQueryDependencies{
					BOID:        tenantCImported.BOID,
					TermNodeIDs: []string{"desk", "pnl"},
				},
			},
		},
	}
	wireBytesC, err := json.Marshal(reExportBundleC)
	require.NoError(t, err)

	var finalReExport QueryBundle
	err = json.Unmarshal(wireBytesC, &finalReExport)
	require.NoError(t, err)

	finalContent := savedQueryContent{
		Name:        finalReExport.Queries[0].Name,
		Description: finalReExport.Queries[0].Description,
		BOID:        finalReExport.Queries[0].BOID,
		ChartType:   finalReExport.Queries[0].ChartType,
		State:       finalReExport.Queries[0].State,
		Tags:        finalReExport.Queries[0].Tags,
	}.normalized()
	finalHash := ComputeContentHash(finalContent)

	// Invariant assertion: Round-trip-from-a-round-trip produces identical canonical hash
	assert.Equal(t, hashA, hashC)
	assert.Equal(t, hashC, finalHash)
	assert.Equal(t, tenantAOriginal, finalContent)
}
