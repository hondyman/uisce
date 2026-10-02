package querybuilder

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindSavedQueryReferencesInPageModel(t *testing.T) {
	targetQueryID := "sq-123-target"

	// 1. App model referencing query in app.queries and in widget interactionConfig
	pageJSON := []byte(`{
		"app": {
			"queries": [
				{ "id": "q1", "kind": "operation", "name": "op1" },
				{ "id": "q2", "kind": "savedQuery", "savedQueryId": "sq-123-target", "name": "My Target Query" }
			],
			"pages": [
				{
					"id": "p1",
					"name": "Executive Dashboard",
					"widgets": [
						{
							"id": "w_chart",
							"type": "chart",
							"config": {
								"savedQueryId": "sq-123-target",
								"interactionConfig": {
									"drillTargets": [
										{ "type": "query", "target": "sq-123-target", "label": "Drill into Target" },
										{ "type": "page", "target": "/pages/other", "label": "Other Page" }
									]
								}
							}
						}
					]
				}
			]
		}
	}`)

	refs := FindSavedQueryReferencesInPageModel(pageJSON, targetQueryID, "pg-001", "Executive Dashboard", false, 1)
	require.Len(t, refs, 3)

	locMap := make(map[string]SavedQueryReference)
	for _, r := range refs {
		locMap[r.Location] = r
	}

	// 1. app.queries reference
	qRef, hasQ := locMap["app.queries[1]"]
	require.True(t, hasQ)
	assert.Equal(t, "page_draft", qRef.Type)
	assert.Equal(t, "pg-001", qRef.ID)
	assert.Equal(t, "Executive Dashboard", qRef.Name)

	// 2. widget config reference
	wRef, hasW := locMap["app.pages[0].widgets[0].config"]
	require.True(t, hasW)
	assert.Equal(t, "page_draft", wRef.Type)

	// 3. drill-through target reference
	var drillRef *SavedQueryReference
	for _, r := range refs {
		if r.Type == "drill_through_target" {
			drillRef = &r
			break
		}
	}
	require.NotNil(t, drillRef)
	assert.Equal(t, "sq-123-target", drillRef.Target)
	assert.Contains(t, drillRef.Location, "drillTargets[0]")

	// 2. Published page check
	pubRefs := FindSavedQueryReferencesInPageModel(pageJSON, targetQueryID, "pg-002", "Published Page", true, 3)
	require.Len(t, pubRefs, 3)
	assert.Equal(t, "page_published", pubRefs[0].Type)
	assert.Equal(t, 3, pubRefs[0].Version)

	// 3. Deeply nested container/tab widget test (3 levels deep)
	nestedJSON := []byte(`{
		"layout": {
			"tabs": [
				{
					"key": "tab_analytics",
					"containers": [
						{
							"id": "container_summary",
							"children": [
								{
									"widgetId": "w_deep_nested",
									"type": "chart",
									"config": {
										"savedQueryId": "sq-123-target"
									}
								}
							]
						}
					]
				}
			]
		}
	}`)
	nestedRefs := FindSavedQueryReferencesInPageModel(nestedJSON, targetQueryID, "pg-nested-01", "Analytics Hub", false, 1)
	require.Len(t, nestedRefs, 1)
	assert.Equal(t, "page_draft", nestedRefs[0].Type)
	assert.Equal(t, "pg-nested-01", nestedRefs[0].ID)
	assert.Contains(t, nestedRefs[0].Location, "tabs[0].containers[0].children[0].config")

	// 4. Unrelated page model
	unrelatedJSON := []byte(`{ "app": { "queries": [{ "id": "q99", "kind": "savedQuery", "savedQueryId": "sq-other" }] } }`)
	noRefs := FindSavedQueryReferencesInPageModel(unrelatedJSON, targetQueryID, "pg-003", "Other Page", false, 1)
	assert.Empty(t, noRefs)
}

func setupUsageTestEnv(t *testing.T, tenantID, userID string) (*SavedQueryHandler, *chi.Mux, sqlmock.Sqlmock, func(method, path string, body []byte) *http.Request) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	deps := handlers.SecurityContextDeps{Resolver: &mockDSResolver{defaultTenantID: tenantID}}
	h := NewSavedQueryHandler(sqlxDB, nil, nil, deps)

	r := chi.NewRouter()
	r.Get("/api/explorer/saved-queries/{id}/usage", h.HandleGetSavedQueryUsage)
	r.Patch("/api/explorer/saved-queries/{id}", h.HandlePatchSavedQuery)
	r.Delete("/api/explorer/saved-queries/{id}", h.HandleDeleteSavedQuery)

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
		req.Header.Set("X-Region", "us-east-1") // a real client states its region; there is no default
		auth := security.AuthInfo{
			TenantIDs: []string{tenantID},
			UserID:    userID,
			Roles:     []string{"tenant_admin"},
		}
		return req.WithContext(security.WithAuthInfo(req.Context(), auth))
	}

	return h, r, mock, makeReq
}

func TestSavedQueryUsage_HandlerEndpoints(t *testing.T) {
	tenantID := "11111111-1111-1111-1111-111111111111"
	userID := "22222222-2222-2222-2222-222222222222"
	queryID := "33333333-3333-3333-3333-333333333333"

	t.Run("GET /usage returns 200 with usage report", func(t *testing.T) {
		_, router, mock, makeReq := setupUsageTestEnv(t, tenantID, userID)

		// Mock loadOwned query
		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
			WithArgs(queryID, tenantID, userID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				queryID, tenantID, userID, "Active Query", "Desc", "bo-1", nil, nil,
				"bar", []byte(`{}`), nil, nil, false, "shared", false,
				"active", nil, userID, time.Now(), time.Now(),
			))

		// Mock page scan
		appModelJSON := []byte(`{"app": {"queries": [{"id": "q1", "kind": "savedQuery", "savedQueryId": "` + queryID + `"}]}}`)
		mock.ExpectQuery(`SELECT id, tenant_id, name, slug, status, version, .* FROM page_definitions WHERE tenant_id = \$1`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "name", "slug", "status", "version", "app_model", "layout", "components", "data_sources",
			}).AddRow("pg-1", tenantID, "Dashboard", "dashboard", "draft", 1, appModelJSON, []byte(`[]`), []byte(`[]`), []byte(`[]`)))

		// Mock other saved query scan
		mock.ExpectQuery(`SELECT id, name, query_state FROM data_explorer\.saved_query WHERE id != \$1 AND archived_at IS NULL AND tenant_id = \$2`).
			WithArgs(queryID, tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "query_state"}))

		// Mock schedules scan
		mock.ExpectQuery(`SELECT id::text, name, tenant_id::text FROM public\.schedules WHERE deleted_at IS NULL AND .* AND tenant_id::text = \$2`).
			WithArgs(queryID, tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "tenant_id"}))

		req := makeReq("GET", "/api/explorer/saved-queries/"+queryID+"/usage", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var report SavedQueryUsageReport
		err := json.Unmarshal(rec.Body.Bytes(), &report)
		require.NoError(t, err)
		assert.True(t, report.InUse)
		require.Len(t, report.References, 1)
		assert.Equal(t, "page_draft", report.References[0].Type)
		assert.Equal(t, "pg-1", report.References[0].ID)
	})

	t.Run("DELETE returns 409 Conflict when query is in use", func(t *testing.T) {
		_, router, mock, makeReq := setupUsageTestEnv(t, tenantID, userID)

		// Mock loadOwned
		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
			WithArgs(queryID, tenantID, userID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				queryID, tenantID, userID, "Referenced Query", "Desc", "bo-1", nil, nil,
				"bar", []byte(`{}`), nil, nil, false, "shared", false,
				"active", nil, userID, time.Now(), time.Now(),
			))

		// Mock page scan finding reference
		appModelJSON := []byte(`{"app": {"queries": [{"id": "q1", "kind": "savedQuery", "savedQueryId": "` + queryID + `"}]}}`)
		mock.ExpectQuery(`SELECT id, tenant_id, name, slug, status, version, .* FROM page_definitions WHERE tenant_id = \$1`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "name", "slug", "status", "version", "app_model", "layout", "components", "data_sources",
			}).AddRow("pg-pub-1", tenantID, "Live Exec Dashboard", "live-dashboard", "published", 2, appModelJSON, []byte(`[]`), []byte(`[]`), []byte(`[]`)))

		// Mock other saved query scan
		mock.ExpectQuery(`SELECT id, name, query_state FROM data_explorer\.saved_query WHERE id != \$1 AND archived_at IS NULL AND tenant_id = \$2`).
			WithArgs(queryID, tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "query_state"}))

		// Mock schedules scan
		mock.ExpectQuery(`SELECT id::text, name, tenant_id::text FROM public\.schedules WHERE deleted_at IS NULL AND .* AND tenant_id::text = \$2`).
			WithArgs(queryID, tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "tenant_id"}))

		req := makeReq("DELETE", "/api/explorer/saved-queries/"+queryID, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
		var conflictResp map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &conflictResp)
		require.NoError(t, err)
		assert.Equal(t, "Conflict", conflictResp["error"])
		assert.Equal(t, true, conflictResp["inUse"])
	})

	t.Run("DELETE performs soft delete (204) when zero references exist", func(t *testing.T) {
		_, router, mock, makeReq := setupUsageTestEnv(t, tenantID, userID)

		// Mock loadOwned
		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
			WithArgs(queryID, tenantID, userID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				queryID, tenantID, userID, "Unused Query", "Desc", "bo-1", nil, nil,
				"bar", []byte(`{}`), nil, nil, false, "shared", false,
				"active", nil, userID, time.Now(), time.Now(),
			))

		// Mock page scan (no matches)
		mock.ExpectQuery(`SELECT id, tenant_id, name, slug, status, version, .* FROM page_definitions WHERE tenant_id = \$1`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "name", "slug", "status", "version", "app_model", "layout", "components", "data_sources",
			}))

		// Mock other saved query scan (no matches)
		mock.ExpectQuery(`SELECT id, name, query_state FROM data_explorer\.saved_query WHERE id != \$1 AND archived_at IS NULL AND tenant_id = \$2`).
			WithArgs(queryID, tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "query_state"}))

		// Mock schedules scan (no matches)
		mock.ExpectQuery(`SELECT id::text, name, tenant_id::text FROM public\.schedules WHERE deleted_at IS NULL AND .* AND tenant_id::text = \$2`).
			WithArgs(queryID, tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "tenant_id"}))

		// Mock soft-delete UPDATE
		mock.ExpectExec(`UPDATE data_explorer\.saved_query SET archived_at = NOW\(\), status = 'archived', updated_at = NOW\(\) WHERE id = \$1 AND tenant_id = \$2`).
			WithArgs(queryID, tenantID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		req := makeReq("DELETE", "/api/explorer/saved-queries/"+queryID, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("PATCH updates lifecycle status (e.g. deprecate)", func(t *testing.T) {
		_, router, mock, makeReq := setupUsageTestEnv(t, tenantID, userID)

		// Mock loadOwned
		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
			WithArgs(queryID, tenantID, userID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				queryID, tenantID, userID, "Old Query", "Desc", "bo-1", nil, nil,
				"bar", []byte(`{}`), nil, nil, false, "shared", false,
				"active", nil, userID, time.Now(), time.Now(),
			))

		// Mock UPDATE RETURNING
		mock.ExpectQuery(`UPDATE data_explorer\.saved_query SET status = \$1, archived_at = \$2, updated_at = NOW\(\) WHERE id = \$3 AND tenant_id = \$4 RETURNING .*`).
			WithArgs("deprecated", nil, queryID, tenantID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				queryID, tenantID, userID, "Old Query", "Desc", "bo-1", nil, nil,
				"bar", []byte(`{}`), nil, nil, false, "shared", false,
				"deprecated", nil, userID, time.Now(), time.Now(),
			))

		req := makeReq("PATCH", "/api/explorer/saved-queries/"+queryID, []byte(`{"status": "deprecated"}`))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var patched SavedQuery
		err := json.Unmarshal(rec.Body.Bytes(), &patched)
		require.NoError(t, err)
		assert.Equal(t, "deprecated", patched.Status)
	})

	t.Run("PATCH allows client tenant to archive own query (setting archived_at)", func(t *testing.T) {
		_, router, mock, makeReq := setupUsageTestEnv(t, tenantID, userID)

		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
			WithArgs(queryID, tenantID, userID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				queryID, tenantID, userID, "My Query", "Desc", "bo-1", nil, nil,
				"bar", []byte(`{}`), nil, nil, false, "shared", false,
				"active", nil, userID, time.Now(), time.Now(),
			))

		mock.ExpectQuery(`UPDATE data_explorer\.saved_query SET status = \$1, archived_at = \$2, updated_at = NOW\(\) WHERE id = \$3 AND tenant_id = \$4 RETURNING .*`).
			WithArgs("archived", sqlmock.AnyArg(), queryID, tenantID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				queryID, tenantID, userID, "My Query", "Desc", "bo-1", nil, nil,
				"bar", []byte(`{}`), nil, nil, false, "shared", false,
				"archived", time.Now(), userID, time.Now(), time.Now(),
			))

		req := makeReq("PATCH", "/api/explorer/saved-queries/"+queryID, []byte(`{"status": "archived"}`))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var patched SavedQuery
		err := json.Unmarshal(rec.Body.Bytes(), &patched)
		require.NoError(t, err)
		assert.Equal(t, "archived", patched.Status)
		assert.NotNil(t, patched.ArchivedAt)
	})

	t.Run("PATCH rejects client tenant attempting to deprecate core query with 403", func(t *testing.T) {
		_, router, mock, makeReq := setupUsageTestEnv(t, tenantID, userID)
		coreID := "44444444-4444-4444-4444-444444444444"
		goldTenant := "99999999-9999-9999-9999-999999999999"

		// Mock loadOwned: not found in tenant's own queries
		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
			WithArgs(coreID, tenantID, userID).
			WillReturnError(sql.ErrNoRows)

		// Gold tenant query in loadOwned
		mock.ExpectQuery(`SELECT id FROM \(SELECT public\.uisce_gold_copy_tenant_id\(\) AS id\) g WHERE id IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.MustParse(goldTenant)))

		// Found as core query from master tenant
		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND is_core = true AND tenant_id = \$2`).
			WithArgs(coreID, goldTenant).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				coreID, goldTenant, userID, "Gold Core Query", "Desc", "bo-1", nil, nil,
				"bar", []byte(`{}`), nil, nil, false, "shared", true,
				"active", nil, userID, time.Now(), time.Now(),
			))

		// Adoption query in loadOwned
		mock.ExpectQuery(`SELECT .* FROM public\.core_object_adoption WHERE tenant_id = \$1 AND object_type = \$2 AND core_object_id = \$3`).
			WithArgs(tenantID, "saved_query", coreID).
			WillReturnError(sql.ErrNoRows)

		// canCustomize gold tenant check in loadOwned
		mock.ExpectQuery(`SELECT id FROM \(SELECT public\.uisce_gold_copy_tenant_id\(\) AS id\) g WHERE id IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.MustParse(goldTenant)))

		// isGoldCopy check in HandlePatchSavedQuery
		mock.ExpectQuery(`SELECT id FROM \(SELECT public\.uisce_gold_copy_tenant_id\(\) AS id\) g WHERE id IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.MustParse(goldTenant)))

		req := makeReq("PATCH", "/api/explorer/saved-queries/"+coreID, []byte(`{"status": "deprecated"}`))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Contains(t, rec.Body.String(), "cannot modify status of core query")
	})

	t.Run("DELETE returns 403 when client tenant attempts to delete core query", func(t *testing.T) {
		_, router, mock, makeReq := setupUsageTestEnv(t, tenantID, userID)
		coreID := "44444444-4444-4444-4444-444444444444"
		goldTenant := "99999999-9999-9999-9999-999999999999"

		// Mock loadOwned: not found in tenant's own queries
		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
			WithArgs(coreID, tenantID, userID).
			WillReturnError(sql.ErrNoRows)

		// Gold tenant query in loadOwned
		mock.ExpectQuery(`SELECT id FROM \(SELECT public\.uisce_gold_copy_tenant_id\(\) AS id\) g WHERE id IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.MustParse(goldTenant)))

		// Found as core query from master tenant
		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND is_core = true AND tenant_id = \$2`).
			WithArgs(coreID, goldTenant).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				coreID, goldTenant, userID, "Gold Copy Query", "Desc", "bo-1", nil, nil,
				"bar", []byte(`{}`), nil, nil, false, "shared", true,
				"active", nil, userID, time.Now(), time.Now(),
			))

		// Adoption query in loadOwned
		mock.ExpectQuery(`SELECT .* FROM public\.core_object_adoption WHERE tenant_id = \$1 AND object_type = \$2 AND core_object_id = \$3`).
			WithArgs(tenantID, "saved_query", coreID).
			WillReturnError(sql.ErrNoRows)

		// canCustomize gold tenant check in loadOwned
		mock.ExpectQuery(`SELECT id FROM \(SELECT public\.uisce_gold_copy_tenant_id\(\) AS id\) g WHERE id IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.MustParse(goldTenant)))

		// isGoldCopy check in HandleDeleteSavedQuery
		mock.ExpectQuery(`SELECT id FROM \(SELECT public\.uisce_gold_copy_tenant_id\(\) AS id\) g WHERE id IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.MustParse(goldTenant)))

		req := makeReq("DELETE", "/api/explorer/saved-queries/"+coreID, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Contains(t, rec.Body.String(), "cannot delete core query")
	})

	t.Run("DELETE returns 409 when gold-copy tenant deletes core query with active adoptions", func(t *testing.T) {
		_, router, mock, makeReq := setupUsageTestEnv(t, tenantID, userID)
		coreID := "55555555-5555-5555-5555-555555555555"

		// Mock loadOwned for master tenant
		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
			WithArgs(coreID, tenantID, userID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				coreID, tenantID, userID, "Master Core Query", "Desc", "bo-1", nil, nil,
				"bar", []byte(`{}`), nil, nil, false, "shared", true,
				"active", nil, userID, time.Now(), time.Now(),
			))

		// Gold tenant check returns tenantID
		mock.ExpectQuery(`SELECT id FROM \(SELECT public\.uisce_gold_copy_tenant_id\(\) AS id\) g WHERE id IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.MustParse(tenantID)))

		// Page scan: no pages
		mock.ExpectQuery(`SELECT id, tenant_id, name, slug, status, version, .* FROM page_definitions`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "name", "slug", "status", "version", "app_model", "layout", "components", "data_sources",
			}))

		// Query scan: no other queries
		mock.ExpectQuery(`SELECT id, name, query_state FROM data_explorer\.saved_query WHERE id != \$1 AND archived_at IS NULL`).
			WithArgs(coreID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "query_state"}))

		// Core adoption scan: 2 client tenants have adopted this core query!
		mock.ExpectQuery(`SELECT tenant_id, COUNT\(\*\) as count FROM public\.core_object_adoption WHERE object_type = 'saved_query' AND core_object_id = \$1 GROUP BY tenant_id`).
			WithArgs(coreID).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "count"}).
				AddRow("client-tenant-a", 3).
				AddRow("client-tenant-b", 1))

		// Schedules scan
		mock.ExpectQuery(`SELECT id::text, name, tenant_id::text FROM public\.schedules WHERE deleted_at IS NULL AND .*`).
			WithArgs(coreID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "tenant_id"}))

		req := makeReq("DELETE", "/api/explorer/saved-queries/"+coreID, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
		var conflictResp map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &conflictResp)
		require.NoError(t, err)
		assert.Equal(t, "Conflict", conflictResp["error"])
		assert.Equal(t, true, conflictResp["inUse"])
	})

	t.Run("DELETE returns 409 when query has active scheduled_job reference in schedules", func(t *testing.T) {
		_, router, mock, makeReq := setupUsageTestEnv(t, tenantID, userID)
		targetID := "66666666-6666-6666-6666-666666666666"

		// Mock loadOwned
		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2 AND \(visibility = 'shared' OR user_id = \$3\)`).
			WithArgs(targetID, tenantID, userID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				targetID, tenantID, userID, "Scheduled Metric Query", "Desc", "bo-1", nil, nil,
				"bar", []byte(`{}`), nil, nil, false, "shared", false,
				"active", nil, userID, time.Now(), time.Now(),
			))

		// Page scan: no pages
		mock.ExpectQuery(`SELECT id, tenant_id, name, slug, status, version, .* FROM page_definitions WHERE tenant_id = \$1`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "name", "slug", "status", "version", "app_model", "layout", "components", "data_sources",
			}))

		// Query scan: no other queries
		mock.ExpectQuery(`SELECT id, name, query_state FROM data_explorer\.saved_query WHERE id != \$1 AND archived_at IS NULL AND tenant_id = \$2`).
			WithArgs(targetID, tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "query_state"}))

		// Scheduled job scan: active schedule found!
		mock.ExpectQuery(`SELECT id::text, name, tenant_id::text FROM public\.schedules WHERE deleted_at IS NULL AND .* AND tenant_id::text = \$2`).
			WithArgs(targetID, tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "tenant_id"}).
				AddRow("sched-daily-report-001", "Daily Executive Email Dispatch", tenantID))

		req := makeReq("DELETE", "/api/explorer/saved-queries/"+targetID, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
		var conflictResp map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &conflictResp)
		require.NoError(t, err)
		assert.Equal(t, "Conflict", conflictResp["error"])
		assert.Equal(t, true, conflictResp["inUse"])

		refs, ok := conflictResp["references"].([]interface{})
		require.True(t, ok)
		require.Len(t, refs, 1)
		firstRef := refs[0].(map[string]interface{})
		assert.Equal(t, "scheduled_job", firstRef["type"])
		assert.Equal(t, "sched-daily-report-001", firstRef["id"])
		assert.Equal(t, "Daily Executive Email Dispatch", firstRef["name"])
	})
}
