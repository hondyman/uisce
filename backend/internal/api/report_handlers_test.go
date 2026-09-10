package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	httpapi "github.com/hondyman/uisce/backend/internal/api"
	"github.com/hondyman/uisce/backend/internal/reports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportAPI(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := reports.NewReportService(db)
	handler := httpapi.NewReportHandler(service)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	t.Run("Create Template", func(t *testing.T) {
		// Expect duplicate check pre-query
		dupRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM report_templates WHERE tenant_id = \$1 AND LOWER\(template_name\) = LOWER\(\$2\)`).
			WillReturnRows(dupRows)

		mock.ExpectExec(`INSERT INTO report_templates`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		payload := map[string]interface{}{
			"template_name": "Monthly Performance",
			"description":   "Monthly portfolio performance report",
			"category":      "performance",
			"is_active":     true,
			"is_public":     false,
			"layout_config": map[string]interface{}{
				"sections": []interface{}{},
			},
			"parameter_schema": map[string]interface{}{},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/reports/", bytes.NewBuffer(body))
		req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
		req.Header.Set("X-User-ID", "user-123")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("List Templates", func(t *testing.T) {
		// Mock resolve gold_copy tenant
		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("99e99e99-99e9-49e9-89e9-99e99e99e999"))

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "template_name", "description", "category",
			"layout_config", "parameter_schema", "is_active", "is_public",
			"is_personal", "created_by_id", "created_by",
			"created_at", "updated_at", "version", "is_favorite",
		}).AddRow(
			"00000000-0000-0000-0000-000000000001", "11111111-1111-1111-1111-111111111111",
			"Report 1", "Desc 1", "perf", []byte("{}"), []byte("{}"),
			true, false, false, nil, nil,
			time.Now(), time.Now(), 1, false,
		)

		mock.ExpectQuery(`SELECT t\.id, t\.tenant_id, t\.template_name`).
			WillReturnRows(rows)

		req := httptest.NewRequest("GET", "/api/v1/reports/", nil)
		req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
		req.Header.Set("X-User-ID", "user-123")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Logf("Response Body: %s", w.Body.String())
		}
		assert.Equal(t, http.StatusOK, w.Code)
		var templates []map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&templates)
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, "Report 1", templates[0]["template_name"])
	})

	t.Run("Create Template - 409 Conflict on Duplicate Name", func(t *testing.T) {
		dupRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM report_templates WHERE tenant_id = \$1 AND LOWER\(template_name\) = LOWER\(\$2\)`).
			WillReturnRows(dupRows)

		payload := map[string]interface{}{
			"template_name": "Existing Report",
			"description":   "Duplicate test",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/reports/", bytes.NewBuffer(body))
		req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
		req.Header.Set("X-User-ID", "user-123")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("Favorite Endpoint - Rejects Request Body", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/v1/reports/00000000-0000-0000-0000-000000000001/favorite", bytes.NewBufferString(`{"inject": true}`))
		req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
		req.Header.Set("X-User-ID", "user-123")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Update Template - 403 on Modifying Gold-Copy Core Report", func(t *testing.T) {
		goldCopyTenant := "99e99e99-99e9-49e9-89e9-99e99e99e999"
		clientTenant := "11111111-1111-1111-1111-111111111111"

		// GetTemplate returns goldCopy report
		mock.ExpectQuery(`SELECT id, tenant_id, template_name`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "template_name", "description", "category",
				"semantic_view_ids", "layout_config", "parameter_schema",
				"is_active", "is_public", "is_personal", "created_by_id", "created_by",
				"created_at", "updated_at", "version",
			}).AddRow(
				"00000000-0000-0000-0000-000000000001", goldCopyTenant,
				"Core Report", "Desc", "cat",
				nil, []byte("{}"), []byte("{}"),
				true, false, false, nil, "",
				time.Now(), time.Now(), 1,
			))

		// ResolveGoldCopyTenantID
		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldCopyTenant))

		payload := map[string]interface{}{
			"template_name": "Hacked Name",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("PUT", "/api/v1/reports/00000000-0000-0000-0000-000000000001", bytes.NewBuffer(body))
		req.Header.Set("X-Tenant-ID", clientTenant)
		req.Header.Set("X-User-ID", "client-user")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Update Template - 403 on Unauthorized Personalization Change", func(t *testing.T) {
		clientTenant := "11111111-1111-1111-1111-111111111111"
		authorID := "author-456"
		nonAuthorID := "other-user-789"

		// GetTemplate returns report created by authorID
		mock.ExpectQuery(`SELECT id, tenant_id, template_name`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "template_name", "description", "category",
				"semantic_view_ids", "layout_config", "parameter_schema",
				"is_active", "is_public", "is_personal", "created_by_id", "created_by",
				"created_at", "updated_at", "version",
			}).AddRow(
				"00000000-0000-0000-0000-000000000002", clientTenant,
				"Personal Report", "Desc", "cat",
				nil, []byte("{}"), []byte("{}"),
				true, false, true, authorID, "",
				time.Now(), time.Now(), 1,
			))

		// ResolveGoldCopyTenantID
		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("99e99e99-99e9-49e9-89e9-99e99e99e999"))

		// Non-author attempts to un-personalize (is_personal: false)
		payload := map[string]interface{}{
			"is_personal": false,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("PUT", "/api/v1/reports/00000000-0000-0000-0000-000000000002", bytes.NewBuffer(body))
		req.Header.Set("X-Tenant-ID", clientTenant)
		req.Header.Set("X-User-ID", nonAuthorID)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Update Template - 403 on Sharing Non-Personal Report", func(t *testing.T) {
		clientTenant := "11111111-1111-1111-1111-111111111111"
		authorID := "author-456"

		// GetTemplate returns tenant-wide report (is_personal: false)
		mock.ExpectQuery(`SELECT id, tenant_id, template_name`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "template_name", "description", "category",
				"semantic_view_ids", "layout_config", "parameter_schema",
				"is_active", "is_public", "is_personal", "created_by_id", "created_by",
				"created_at", "updated_at", "version",
			}).AddRow(
				"00000000-0000-0000-0000-000000000003", clientTenant,
				"Tenant Report", "Desc", "cat",
				nil, []byte("{}"), []byte("{}"),
				true, false, false, authorID, "",
				time.Now(), time.Now(), 1,
			))

		// ResolveGoldCopyTenantID
		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("99e99e99-99e9-49e9-89e9-99e99e99e999"))

		// Author attempts to share a non-personal report
		payload := map[string]interface{}{
			"metadata": map[string]interface{}{
				"is_shared": true,
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("PUT", "/api/v1/reports/00000000-0000-0000-0000-000000000003", bytes.NewBuffer(body))
		req.Header.Set("X-Tenant-ID", clientTenant)
		req.Header.Set("X-User-ID", authorID)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

