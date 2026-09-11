package api_test

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
	httpapi "github.com/hondyman/uisce/backend/internal/api"
	"github.com/hondyman/uisce/backend/internal/reports"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFolderTestRouter(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *chi.Mux) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	service := reports.NewReportService(db)
	handler := httpapi.NewReportHandler(service, nil, db)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	return db, mock, r
}

func authRequest(method, url string, body []byte, tenantID, userID string) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, url, nil)
	}

	auth := security.AuthInfo{
		UserID:    userID,
		TenantIDs: []string{tenantID},
		Roles:     []string{"user"},
	}
	return req.WithContext(security.WithAuthInfo(req.Context(), auth))
}

func TestReportFolderAPI_AuthAndSecurity(t *testing.T) {
	_, _, r := setupFolderTestRouter(t)

	t.Run("Security - Reject Client Headers in Production (ALLOW_CLIENT_TENANT_HEADER_FALLBACK=false)", func(t *testing.T) {
		t.Setenv("ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "false")

		endpoints := []struct {
			method string
			path   string
			body   string
		}{
			{"GET", "/api/v1/reports/folders", ""},
			{"POST", "/api/v1/reports/folders", `{"name":"Test"}`},
			{"PUT", "/api/v1/reports/folders/" + uuid.New().String(), `{"name":"Rename"}`},
			{"POST", "/api/v1/reports/folders/" + uuid.New().String() + "/move", `{"parent_id":null}`},
			{"DELETE", "/api/v1/reports/folders/" + uuid.New().String(), ""},
			{"GET", "/api/v1/reports/folders/" + uuid.New().String() + "/items", ""},
			{"POST", "/api/v1/reports/folders/" + uuid.New().String() + "/items", `{"template_id":"` + uuid.New().String() + `"}`},
			{"DELETE", "/api/v1/reports/folders/" + uuid.New().String() + "/items/" + uuid.New().String(), ""},
		}

		for _, ep := range endpoints {
			var req *http.Request
			if ep.body != "" {
				req = httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString(ep.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(ep.method, ep.path, nil)
			}
			req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
			req.Header.Set("X-User-ID", "attacker")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code, "Expected 401 on unauthenticated %s %s", ep.method, ep.path)
		}
	})

	t.Run("Security - JWT Auth Takes Absolute Precedence Over Spoofed Headers", func(t *testing.T) {
		_, mock, r2 := setupFolderTestRouter(t)
		t.Setenv("ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "true")

		realTenant := "11111111-1111-1111-1111-111111111111"
		realTenantUUID := uuid.MustParse(realTenant)
		realUser := "real-user"

		// Query must be scoped to realTenant & realUser, NOT spoofed headers
		mock.ExpectQuery(`SELECT f\.id, f\.tenant_id, f\.user_id, f\.parent_id, f\.name`).
			WithArgs(realTenantUUID, realUser, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "user_id", "parent_id", "name", "created_at", "updated_at", "report_count"}).
				AddRow(uuid.New(), realTenantUUID, realUser, nil, "My Real Folder", time.Now(), time.Now(), 0))

		req := authRequest("GET", "/api/v1/reports/folders", nil, realTenant, realUser)
		req.Header.Set("X-Tenant-ID", "99999999-9999-9999-9999-999999999999")
		req.Header.Set("X-User-ID", "spoofed-user")

		w := httptest.NewRecorder()
		r2.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReportFolderAPI_FolderCRUD(t *testing.T) {
	tenantID := "11111111-1111-1111-1111-111111111111"
	tenantUUID := uuid.MustParse(tenantID)
	userID := "user-123"

	t.Run("CreateFolder - Success (201 Created)", func(t *testing.T) {
		_, mock, r := setupFolderTestRouter(t)

		// 1. Sibling pre-check
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM public\.report_folders WHERE tenant_id = \$1 AND user_id = \$2 AND parent_id IS NULL AND LOWER\(name\) = LOWER\(\$3\)`).
			WithArgs(tenantUUID, userID, "Executive Reports", sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		// 2. Insert folder
		mock.ExpectQuery(`INSERT INTO public\.report_folders`).
			WithArgs(sqlmock.AnyArg(), tenantUUID, userID, nil, "Executive Reports").
			WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(time.Now(), time.Now()))

		body, _ := json.Marshal(map[string]interface{}{"name": "Executive Reports"})
		req := authRequest("POST", "/api/v1/reports/folders", body, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp reports.ReportFolder
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, "Executive Reports", resp.Name)
		assert.Equal(t, tenantUUID, resp.TenantID)
		assert.Equal(t, userID, resp.UserID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CreateFolder - Empty Name (400 Bad Request)", func(t *testing.T) {
		_, _, r := setupFolderTestRouter(t)

		body, _ := json.Marshal(map[string]interface{}{"name": "   "})
		req := authRequest("POST", "/api/v1/reports/folders", body, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "folder name is required")
	})

	t.Run("CreateFolder - Sibling Duplicate Conflict (409 Conflict)", func(t *testing.T) {
		_, mock, r := setupFolderTestRouter(t)

		// Sibling pre-check detects existing sibling with same name
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM public\.report_folders WHERE tenant_id = \$1 AND user_id = \$2 AND parent_id IS NULL AND LOWER\(name\) = LOWER\(\$3\)`).
			WithArgs(tenantUUID, userID, "Executive Reports", sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		body, _ := json.Marshal(map[string]interface{}{"name": "Executive Reports"})
		req := authRequest("POST", "/api/v1/reports/folders", body, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("RenameFolder - Success (200 OK)", func(t *testing.T) {
		_, mock, r := setupFolderTestRouter(t)
		folderID := uuid.New()

		// 1. Fetch current parent_id
		mock.ExpectQuery(`SELECT parent_id FROM public\.report_folders WHERE id = \$1 AND tenant_id = \$2 AND user_id = \$3`).
			WithArgs(folderID, tenantUUID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

		// 2. Sibling pre-check
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM public\.report_folders WHERE tenant_id = \$1 AND user_id = \$2 AND parent_id IS NULL AND LOWER\(name\) = LOWER\(\$3\)`).
			WithArgs(tenantUUID, userID, "Board Deck 2026", folderID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		// 3. Update name
		mock.ExpectExec(`UPDATE public\.report_folders SET name = \$1, updated_at = NOW\(\) WHERE id = \$2 AND tenant_id = \$3 AND user_id = \$4`).
			WithArgs("Board Deck 2026", folderID, tenantUUID, userID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		body, _ := json.Marshal(map[string]interface{}{"name": "Board Deck 2026"})
		req := authRequest("PUT", "/api/v1/reports/folders/"+folderID.String(), body, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("RenameFolder - Rename Payload CANNOT Move Folder (parent_id ignored)", func(t *testing.T) {
		_, mock, r := setupFolderTestRouter(t)
		folderID := uuid.New()
		foreignParentID := uuid.New()

		// Fetch current parent_id
		mock.ExpectQuery(`SELECT parent_id FROM public\.report_folders WHERE id = \$1 AND tenant_id = \$2 AND user_id = \$3`).
			WithArgs(folderID, tenantUUID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

		// Sibling pre-check
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM public\.report_folders WHERE tenant_id = \$1 AND user_id = \$2 AND parent_id IS NULL AND LOWER\(name\) = LOWER\(\$3\)`).
			WithArgs(tenantUUID, userID, "Renamed Folder", folderID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		// CRITICAL: Statement must be UPDATE ... SET name = $1, NOT touching parent_id!
		mock.ExpectExec(`UPDATE public\.report_folders SET name = \$1, updated_at = NOW\(\) WHERE id = \$2 AND tenant_id = \$3 AND user_id = \$4`).
			WithArgs("Renamed Folder", folderID, tenantUUID, userID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Payload attempts to sneak in a parent_id to move the folder
		body, _ := json.Marshal(map[string]interface{}{
			"name":      "Renamed Folder",
			"parent_id": foreignParentID.String(),
		})
		req := authRequest("PUT", "/api/v1/reports/folders/"+folderID.String(), body, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("RenameFolder - Sibling Duplicate Conflict (409 Conflict)", func(t *testing.T) {
		_, mock, r := setupFolderTestRouter(t)
		folderID := uuid.New()

		mock.ExpectQuery(`SELECT parent_id FROM public\.report_folders WHERE id = \$1 AND tenant_id = \$2 AND user_id = \$3`).
			WithArgs(folderID, tenantUUID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM public\.report_folders WHERE tenant_id = \$1 AND user_id = \$2 AND parent_id IS NULL AND LOWER\(name\) = LOWER\(\$3\)`).
			WithArgs(tenantUUID, userID, "Duplicate Name", folderID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		body, _ := json.Marshal(map[string]interface{}{"name": "Duplicate Name"})
		req := authRequest("PUT", "/api/v1/reports/folders/"+folderID.String(), body, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("RenameFolder - Cross-User / Foreign Tenant Isolation (404 Not Found)", func(t *testing.T) {
		_, mock, r := setupFolderTestRouter(t)
		foreignFolderID := uuid.New()

		// User A queries another user's folder -> 0 rows returned
		mock.ExpectQuery(`SELECT parent_id FROM public\.report_folders WHERE id = \$1 AND tenant_id = \$2 AND user_id = \$3`).
			WithArgs(foreignFolderID, tenantUUID, userID).
			WillReturnError(sql.ErrNoRows)

		body, _ := json.Marshal(map[string]interface{}{"name": "Hacked Name"})
		req := authRequest("PUT", "/api/v1/reports/folders/"+foreignFolderID.String(), body, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DeleteFolder - Success (204 No Content)", func(t *testing.T) {
		_, mock, r := setupFolderTestRouter(t)
		folderID := uuid.New()

		mock.ExpectExec(`DELETE FROM public\.report_folders WHERE id = \$1 AND tenant_id = \$2 AND user_id = \$3`).
			WithArgs(folderID, tenantUUID, userID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		req := authRequest("DELETE", "/api/v1/reports/folders/"+folderID.String(), nil, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DeleteFolder - Cross-User Isolation (404 Not Found)", func(t *testing.T) {
		_, mock, r := setupFolderTestRouter(t)
		foreignFolderID := uuid.New()

		mock.ExpectExec(`DELETE FROM public\.report_folders WHERE id = \$1 AND tenant_id = \$2 AND user_id = \$3`).
			WithArgs(foreignFolderID, tenantUUID, userID).
			WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

		req := authRequest("DELETE", "/api/v1/reports/folders/"+foreignFolderID.String(), nil, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReportFolderAPI_MoveHierarchy(t *testing.T) {
	tenantID := "11111111-1111-1111-1111-111111111111"
	tenantUUID := uuid.MustParse(tenantID)
	userID := "user-123"

	t.Run("MoveFolder - Direct Cycle Self-Parent (400 Bad Request)", func(t *testing.T) {
		_, mock, r := setupFolderTestRouter(t)
		folderID := uuid.New()

		mock.ExpectQuery(`SELECT name, parent_id FROM public\.report_folders WHERE id = \$1 AND tenant_id = \$2 AND user_id = \$3`).
			WithArgs(folderID, tenantUUID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"name", "parent_id"}).AddRow("Self Folder", nil))

		body, _ := json.Marshal(map[string]interface{}{"parent_id": folderID.String()})
		req := authRequest("POST", "/api/v1/reports/folders/"+folderID.String()+"/move", body, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "cannot move folder under itself")
	})

	t.Run("MoveFolder - Indirect Cycle Detected (400 Bad Request)", func(t *testing.T) {
		_, mock, r := setupFolderTestRouter(t)
		folderA := uuid.New()
		folderC := uuid.New()

		// Verify folder A exists
		mock.ExpectQuery(`SELECT name, parent_id FROM public\.report_folders WHERE id = \$1 AND tenant_id = \$2 AND user_id = \$3`).
			WithArgs(folderA, tenantUUID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"name", "parent_id"}).AddRow("Folder A", nil))

		// Upward CTE reveals folder A is an ancestor of folder C (is_cycle = true)
		mock.ExpectQuery(`WITH RECURSIVE parent_chain AS`).
			WithArgs(folderC, tenantUUID, userID, folderA).
			WillReturnRows(sqlmock.NewRows([]string{"count", "max_depth", "is_cycle"}).AddRow(3, 3, true))

		body, _ := json.Marshal(map[string]interface{}{"parent_id": folderC.String()})
		req := authRequest("POST", "/api/v1/reports/folders/"+folderA.String()+"/move", body, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "circular dependency")
	})

	t.Run("MoveFolder - Depth Limit Exceeded (400 Bad Request)", func(t *testing.T) {
		_, mock, r := setupFolderTestRouter(t)
		folderID := uuid.New()
		targetParentID := uuid.New()

		// Verify folder exists
		mock.ExpectQuery(`SELECT name, parent_id FROM public\.report_folders WHERE id = \$1 AND tenant_id = \$2 AND user_id = \$3`).
			WithArgs(folderID, tenantUUID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"name", "parent_id"}).AddRow("Moving Subtree", nil))

		// Target parent depth = 4, no cycle
		mock.ExpectQuery(`WITH RECURSIVE parent_chain AS`).
			WithArgs(targetParentID, tenantUUID, userID, folderID).
			WillReturnRows(sqlmock.NewRows([]string{"count", "max_depth", "is_cycle"}).AddRow(4, 4, false))

		// Downward subtree height = 2 (total depth = 4 + 2 = 6 > 5)
		mock.ExpectQuery(`WITH RECURSIVE subtree AS`).
			WithArgs(folderID, tenantUUID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"max_height"}).AddRow(2))

		body, _ := json.Marshal(map[string]interface{}{"parent_id": targetParentID.String()})
		req := authRequest("POST", "/api/v1/reports/folders/"+folderID.String()+"/move", body, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "exceeds maximum depth of 5")
	})

	t.Run("MoveFolder - Foreign Parent Folder (404 Not Found)", func(t *testing.T) {
		_, mock, r := setupFolderTestRouter(t)
		folderID := uuid.New()
		foreignParentID := uuid.New()

		// Moving folder exists
		mock.ExpectQuery(`SELECT name, parent_id FROM public\.report_folders WHERE id = \$1 AND tenant_id = \$2 AND user_id = \$3`).
			WithArgs(folderID, tenantUUID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"name", "parent_id"}).AddRow("My Folder", nil))

		// Target parent CTE returns 0 rows (foreign or non-existent)
		mock.ExpectQuery(`WITH RECURSIVE parent_chain AS`).
			WithArgs(foreignParentID, tenantUUID, userID, folderID).
			WillReturnRows(sqlmock.NewRows([]string{"count", "max_depth", "is_cycle"}).AddRow(0, 0, false))

		body, _ := json.Marshal(map[string]interface{}{"parent_id": foreignParentID.String()})
		req := authRequest("POST", "/api/v1/reports/folders/"+folderID.String()+"/move", body, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReportFolderAPI_FolderItems(t *testing.T) {
	tenantID := "11111111-1111-1111-1111-111111111111"
	tenantUUID := uuid.MustParse(tenantID)
	userID := "user-123"

	t.Run("AddFolderItem - Success (200 OK)", func(t *testing.T) {
		_, mock, r := setupFolderTestRouter(t)
		folderID := uuid.New()
		templateID := uuid.New()

		// Resolve gold copy tenant
		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("99e99e99-99e9-49e9-89e9-99e99e99e999"))

		// Step 1: Visibility predicate check
		mock.ExpectQuery(`SELECT\s+EXISTS\(SELECT 1 FROM public\.report_folders WHERE id = \$1 AND tenant_id = \$2 AND user_id = \$3\) AS folder_ok`).
			WithArgs(folderID, tenantUUID, userID, templateID, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"folder_ok", "template_ok"}).AddRow(true, true))

		// Step 2: Idempotent insert
		mock.ExpectExec(`INSERT INTO public\.report_folder_items \(folder_id, template_id, added_at\)`).
			WithArgs(folderID, templateID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		body, _ := json.Marshal(map[string]interface{}{"template_id": templateID.String()})
		req := authRequest("POST", "/api/v1/reports/folders/"+folderID.String()+"/items", body, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("AddFolderItem - Idempotent Re-Filing Returns 200 OK (Not 404)", func(t *testing.T) {
		_, mock, r := setupFolderTestRouter(t)
		folderID := uuid.New()
		templateID := uuid.New()

		// Resolve gold copy tenant
		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("99e99e99-99e9-49e9-89e9-99e99e99e999"))

		// Step 1: Visibility matches
		mock.ExpectQuery(`SELECT\s+EXISTS\(SELECT 1 FROM public\.report_folders WHERE id = \$1 AND tenant_id = \$2 AND user_id = \$3\) AS folder_ok`).
			WithArgs(folderID, tenantUUID, userID, templateID, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"folder_ok", "template_ok"}).AddRow(true, true))

		// Step 2: Conflict results in 0 rows affected (ON CONFLICT DO NOTHING)
		mock.ExpectExec(`INSERT INTO public\.report_folder_items \(folder_id, template_id, added_at\)`).
			WithArgs(folderID, templateID).
			WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows inserted

		body, _ := json.Marshal(map[string]interface{}{"template_id": templateID.String()})
		req := authRequest("POST", "/api/v1/reports/folders/"+folderID.String()+"/items", body, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Must return 200 OK on idempotent re-filing!
		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("AddFolderItem - Cross-Tenant Report Filing Forbidden (404 Not Found)", func(t *testing.T) {
		_, mock, r := setupFolderTestRouter(t)
		folderID := uuid.New()
		foreignTemplateID := uuid.New()

		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("99e99e99-99e9-49e9-89e9-99e99e99e999"))

		// Template predicate fails: folder_ok = true, but template_ok = false
		mock.ExpectQuery(`SELECT\s+EXISTS\(SELECT 1 FROM public\.report_folders WHERE id = \$1 AND tenant_id = \$2 AND user_id = \$3\) AS folder_ok`).
			WithArgs(folderID, tenantUUID, userID, foreignTemplateID, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"folder_ok", "template_ok"}).AddRow(true, false))

		body, _ := json.Marshal(map[string]interface{}{"template_id": foreignTemplateID.String()})
		req := authRequest("POST", "/api/v1/reports/folders/"+folderID.String()+"/items", body, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("RemoveFolderItem - Success (204 No Content)", func(t *testing.T) {
		_, mock, r := setupFolderTestRouter(t)
		folderID := uuid.New()
		templateID := uuid.New()

		// Verify folder ownership
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM public\.report_folders WHERE id = \$1 AND tenant_id = \$2 AND user_id = \$3\)`).
			WithArgs(folderID, tenantUUID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		mock.ExpectExec(`DELETE FROM public\.report_folder_items WHERE folder_id = \$1 AND template_id = \$2`).
			WithArgs(folderID, templateID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		req := authRequest("DELETE", "/api/v1/reports/folders/"+folderID.String()+"/items/"+templateID.String(), nil, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ListFolderItems - Success With Read Lockstep (200 OK)", func(t *testing.T) {
		_, mock, r := setupFolderTestRouter(t)
		folderID := uuid.New()
		template1 := uuid.New()
		template2 := uuid.New()

		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("99e99e99-99e9-49e9-89e9-99e99e99e999"))

		// 1. Verify folder exists and belongs to caller
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM public\.report_folders WHERE id = \$1 AND tenant_id = \$2 AND user_id = \$3\)`).
			WithArgs(folderID, tenantUUID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// 2. Select with read-path lockstep join
		mock.ExpectQuery(`SELECT i\.template_id FROM public\.report_folder_items i JOIN public\.report_templates t ON t\.id = i\.template_id WHERE i\.folder_id = \$1 AND t\.is_active = true AND \(t\.is_personal = false OR t\.created_by_id = \$2\) AND t\.tenant_id IN \(\$3, \$4\)`).
			WithArgs(folderID, userID, tenantUUID, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"template_id"}).AddRow(template1).AddRow(template2))

		req := authRequest("GET", "/api/v1/reports/folders/"+folderID.String()+"/items", nil, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var itemIDs []uuid.UUID
		err := json.NewDecoder(w.Body).Decode(&itemIDs)
		require.NoError(t, err)
		assert.Len(t, itemIDs, 2)
		assert.Equal(t, template1, itemIDs[0])
		assert.Equal(t, template2, itemIDs[1])
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
