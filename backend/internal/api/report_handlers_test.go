package api_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestReportAPI(t *testing.T) {
	t.Setenv("ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "true")
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := reports.NewReportService(db)
	handler := httpapi.NewReportHandler(service, nil, db)
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

	t.Run("Create Template - Non-admin forced to is_personal=true", func(t *testing.T) {
		dupRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM report_templates WHERE tenant_id = \$1 AND LOWER\(template_name\) = LOWER\(\$2\)`).
			WillReturnRows(dupRows)

		mock.ExpectExec(`INSERT INTO report_templates`).
			WithArgs(
				sqlmock.AnyArg(), // 1: id
				sqlmock.AnyArg(), // 2: tenant_id
				"Non-Admin Report", // 3: template_name
				"",               // 4: description
				"",               // 5: category
				sqlmock.AnyArg(), // 6: layout_config
				sqlmock.AnyArg(), // 7: parameter_schema
				true,             // 8: is_active
				false,            // 9: is_public
				true,             // 10: is_personal (forced to true for non-admin!)
				sqlmock.AnyArg(), // 11: created_by_id
				"",               // 12: created_by
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		payload := map[string]interface{}{
			"template_name": "Non-Admin Report",
			"is_personal":   false, // Attempting to create tenant-wide report
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/reports/", bytes.NewBuffer(body))
		req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
		req.Header.Set("X-User-ID", "user-non-admin")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Logf("Create non-admin failed with code %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)
		var resp reports.ReportTemplate
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.True(t, resp.IsPersonal)
	})

	t.Run("Create Template - Rejects Pre-baked Sharing", func(t *testing.T) {
		payload := map[string]interface{}{
			"template_name": "Shared From Birth",
			"layout_config": map[string]interface{}{
				"metadata": map[string]interface{}{
					"is_shared": true,
				},
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/reports/", bytes.NewBuffer(body))
		req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
		req.Header.Set("X-User-ID", "user-123")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Delete Template - 403 on Deleting Gold-Copy Core Report", func(t *testing.T) {
		goldCopyTenant := "99e99e99-99e9-49e9-89e9-99e99e99e999"
		clientTenant := "11111111-1111-1111-1111-111111111111"

		mock.ExpectQuery(`SELECT id, tenant_id, template_name`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "template_name", "description", "category",
				"semantic_view_ids", "layout_config", "parameter_schema",
				"is_active", "is_public", "is_personal", "created_by_id", "created_by",
				"created_at", "updated_at", "version",
			}).AddRow(
				"00000000-0000-0000-0000-000000000010", goldCopyTenant,
				"Core Report", "Desc", "cat",
				nil, []byte("{}"), []byte("{}"),
				true, false, false, nil, "",
				time.Now(), time.Now(), 1,
			))

		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldCopyTenant))

		req := httptest.NewRequest("DELETE", "/api/v1/reports/00000000-0000-0000-0000-000000000010", nil)
		auth := security.AuthInfo{
			UserID:    "client-admin",
			TenantIDs: []string{clientTenant},
			Roles:     []string{"admin"},
		}
		req = req.WithContext(security.WithAuthInfo(req.Context(), auth))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Delete Template - 403 on Non-Author Deleting Personal Report", func(t *testing.T) {
		clientTenant := "11111111-1111-1111-1111-111111111111"
		authorID := "author-456"
		nonAuthorID := "other-user-789"

		mock.ExpectQuery(`SELECT id, tenant_id, template_name`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "template_name", "description", "category",
				"semantic_view_ids", "layout_config", "parameter_schema",
				"is_active", "is_public", "is_personal", "created_by_id", "created_by",
				"created_at", "updated_at", "version",
			}).AddRow(
				"00000000-0000-0000-0000-000000000020", clientTenant,
				"Personal Report", "Desc", "cat",
				nil, []byte("{}"), []byte("{}"),
				true, false, true, authorID, "",
				time.Now(), time.Now(), 1,
			))

		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("99e99e99-99e9-49e9-89e9-99e99e99e999"))

		req := httptest.NewRequest("DELETE", "/api/v1/reports/00000000-0000-0000-0000-000000000020", nil)
		req.Header.Set("X-Tenant-ID", clientTenant)
		req.Header.Set("X-User-ID", nonAuthorID)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Delete Template - 404 on Cross-Tenant Delete even for Admin", func(t *testing.T) {
		tenantA := "11111111-1111-1111-1111-111111111111"
		tenantB := "22222222-2222-2222-2222-222222222222"

		mock.ExpectQuery(`SELECT id, tenant_id, template_name`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "template_name", "description", "category",
				"semantic_view_ids", "layout_config", "parameter_schema",
				"is_active", "is_public", "is_personal", "created_by_id", "created_by",
				"created_at", "updated_at", "version",
			}).AddRow(
				"00000000-0000-0000-0000-000000000030", tenantB,
				"Tenant B Report", "Desc", "cat",
				nil, []byte("{}"), []byte("{}"),
				true, false, false, nil, "",
				time.Now(), time.Now(), 1,
			))

		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("99e99e99-99e9-49e9-89e9-99e99e99e999"))

		// Admin of tenant A trying to delete tenant B's report
		req := httptest.NewRequest("DELETE", "/api/v1/reports/00000000-0000-0000-0000-000000000030", nil)
		auth := security.AuthInfo{
			UserID:    "tenant-a-admin",
			TenantIDs: []string{tenantA},
			Roles:     []string{"admin"},
		}
		req = req.WithContext(security.WithAuthInfo(req.Context(), auth))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Create Template - Rejects Top-Level is_public", func(t *testing.T) {
		payload := map[string]interface{}{
			"template_name": "Public From Birth",
			"is_public":     true,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/reports/", bytes.NewBuffer(body))
		req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
		req.Header.Set("X-User-ID", "user-123")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Security - Reject Client Headers in Production (ALLOW_CLIENT_TENANT_HEADER_FALLBACK=false)", func(t *testing.T) {
		t.Setenv("ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "false")

		req := httptest.NewRequest("GET", "/api/v1/reports/", nil)
		req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
		req.Header.Set("X-User-ID", "unauthenticated-user")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Security - Spoofed X-Admin Header Is Ignored", func(t *testing.T) {
		t.Setenv("ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "true")

		dupRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM report_templates WHERE tenant_id = \$1 AND LOWER\(template_name\) = LOWER\(\$2\)`).
			WillReturnRows(dupRows)

		// Expect insert where is_personal is STILL FORCED to true because X-Admin header is ignored
		mock.ExpectExec(`INSERT INTO report_templates`).
			WithArgs(
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				"Spoof Attempt Report",
				"",
				"",
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				true,
				false,
				true, // Forced to personal!
				sqlmock.AnyArg(),
				"",
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		payload := map[string]interface{}{
			"template_name": "Spoof Attempt Report",
			"is_personal":   false,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/reports/", bytes.NewBuffer(body))
		req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
		req.Header.Set("X-User-ID", "attacker")
		req.Header.Set("X-Admin", "true") // Malicious admin header
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp reports.ReportTemplate
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.True(t, resp.IsPersonal, "Spoofed X-Admin must not allow creating tenant-wide report")
	})

	t.Run("Security - JWT Auth Takes Absolute Precedence Over Headers", func(t *testing.T) {
		t.Setenv("ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "true")

		realTenant := "11111111-1111-1111-1111-111111111111"
		realTenantUUID := uuid.MustParse(realTenant)
		spoofedTenant := "22222222-2222-2222-2222-222222222222"

		// Query must be scoped to realTenant, NOT spoofedTenant
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM report_templates WHERE tenant_id = \$1 AND LOWER\(template_name\) = LOWER\(\$2\)`).
			WithArgs(realTenantUUID, "JWT Precedence Report", sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		mock.ExpectExec(`INSERT INTO report_templates`).
			WithArgs(
				sqlmock.AnyArg(),
				realTenantUUID, // Real tenant inserted!
				"JWT Precedence Report",
				"",
				"",
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				true,
				false,
				true,
				"real-user-id", // Real user inserted!
				"",
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		payload := map[string]interface{}{
			"template_name": "JWT Precedence Report",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/reports/", bytes.NewBuffer(body))
		// Spoofed headers in request:
		req.Header.Set("X-Tenant-ID", spoofedTenant)
		req.Header.Set("X-User-ID", "spoofed-user")

		// Valid authenticated AuthInfo in context:
		auth := security.AuthInfo{
			UserID:    "real-user-id",
			TenantIDs: []string{realTenant},
			Roles:     []string{"user"},
		}
		req = req.WithContext(security.WithAuthInfo(req.Context(), auth))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Logf("JWT Precedence test failed with code %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("List Templates - Unauthenticated 401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/reports/", nil)
		// No auth context or headers
		w := httptest.NewRecorder()

		// Temporarily disable fallback for strict test
		t.Setenv("ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "false")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("List Templates - Search with ?q= binds parameter", func(t *testing.T) {
		t.Setenv("ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "true")
		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("99e99e99-99e9-49e9-89e9-99e99e99e999"))

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "template_name", "description", "category",
			"layout_config", "parameter_schema", "is_active", "is_public",
			"is_personal", "created_by_id", "created_by",
			"created_at", "updated_at", "version", "is_favorite",
		}).AddRow(
			"00000000-0000-0000-0000-000000000001", "11111111-1111-1111-1111-111111111111",
			"Portfolio Summary", "Desc 1", "perf", []byte("{}"), []byte("{}"),
			true, false, false, nil, nil,
			time.Now(), time.Now(), 1, false,
		)

		// Must execute search query with $4 bound to query string
		mock.ExpectQuery(`websearch_to_tsquery`).
			WithArgs("user-123", sqlmock.AnyArg(), sqlmock.AnyArg(), "Portfolo").
			WillReturnRows(rows)

		req := httptest.NewRequest("GET", "/api/v1/reports/?q=Portfolo", nil)
		req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
		req.Header.Set("X-User-ID", "user-123")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var templates []map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&templates)
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, "Portfolio Summary", templates[0]["template_name"])
	})

	t.Run("List Templates - Query exceeding 256 chars returns 400 Bad Request", func(t *testing.T) {
		t.Setenv("ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "true")
		tooLongQuery := strings.Repeat("a", 257)
		req := httptest.NewRequest("GET", "/api/v1/reports/?q="+tooLongQuery, nil)
		req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
		req.Header.Set("X-User-ID", "user-123")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("List Templates - Empty or whitespace ?q= delegates to standard listing", func(t *testing.T) {
		t.Setenv("ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "true")
		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("99e99e99-99e9-49e9-89e9-99e99e99e999"))

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "template_name", "description", "category",
			"layout_config", "parameter_schema", "is_active", "is_public",
			"is_personal", "created_by_id", "created_by",
			"created_at", "updated_at", "version", "is_favorite",
		}).AddRow(
			"00000000-0000-0000-0000-000000000001", "11111111-1111-1111-1111-111111111111",
			"Standard Listing Report", "Desc", "perf", []byte("{}"), []byte("{}"),
			true, false, false, nil, nil,
			time.Now(), time.Now(), 1, false,
		)

		// Must execute standard listing query (NOT websearch_to_tsquery)
		mock.ExpectQuery(`SELECT t\.id, t\.tenant_id, t\.template_name`).
			WillReturnRows(rows)

		req := httptest.NewRequest("GET", "/api/v1/reports/?q=%20%20%20", nil)
		req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
		req.Header.Set("X-User-ID", "user-123")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var templates []map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&templates)
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, "Standard Listing Report", templates[0]["template_name"])
	})
}

// MockReportExecutor provides configurable responses for TriggerScheduleRun endpoint testing.
type mockPhase3Executor struct {
	execFunc func(ctx context.Context, template *reports.ReportTemplate, params map[string]interface{}) (*reports.ScheduleExecutionResult, error)
}

func (m *mockPhase3Executor) ExecuteReport(ctx context.Context, template *reports.ReportTemplate, params map[string]interface{}) (*reports.ScheduleExecutionResult, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, template, params)
	}
	return &reports.ScheduleExecutionResult{
		ExecutionID: uuid.New(),
		Status:      "pending",
		OutputURL:   "s3://reports/async-test.pdf",
		RequestedBy: template.CreatedBy,
	}, nil
}

func TestReportAPI_Phase3Executions(t *testing.T) {
	tenantID := uuid.New()
	tmplID := uuid.New()
	schedID := uuid.New()
	execID := uuid.New()
	goldCopyID := uuid.New()
	ownerID := "template-owner"

	t.Run("TriggerScheduleRun - Returns 202 Accepted with truthful pending status and workflow_id", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mockExec := &mockPhase3Executor{
			execFunc: func(ctx context.Context, template *reports.ReportTemplate, params map[string]interface{}) (*reports.ScheduleExecutionResult, error) {
				return &reports.ScheduleExecutionResult{
					ExecutionID: execID,
					Status:      "pending",
					OutputURL:   "s3://reports/async-test.pdf",
					RequestedBy: ownerID,
				}, nil
			},
		}

		service := reports.NewReportService(db)
		handler := httpapi.NewReportHandler(service, mockExec, db)
		r := chi.NewRouter()
		handler.RegisterRoutes(r)

		// 1. Fetch schedule
		mock.ExpectQuery(`SELECT id, tenant_id, report_definition_id, owner_id FROM public\.report_schedules WHERE id = \$1 AND tenant_id = \$2 AND is_active = true AND deleted_at IS NULL`).
			WithArgs(schedID, tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "report_definition_id", "owner_id"}).
				AddRow(schedID, tenantID, tmplID, ownerID))

		// 2. Gold copy resolution
		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE.*gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldCopyID))

		// 3. Fetch linked template
		mock.ExpectQuery(`SELECT id, tenant_id, template_name, description, category, semantic_view_ids, layout_config, parameter_schema, is_active, is_public, is_personal, created_by_id, created_by, created_at, updated_at, version FROM public\.report_templates WHERE id = \$1 AND is_active = true AND tenant_id IN \(\$2, \$3\)`).
			WithArgs(tmplID, tenantID, goldCopyID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "template_name", "description", "category",
				"semantic_view_ids", "layout_config", "parameter_schema",
				"is_active", "is_public", "is_personal", "created_by_id", "created_by",
				"created_at", "updated_at", "version",
			}).AddRow(
				tmplID, tenantID, "Monthly PnL", "desc", "cat",
				[]byte(`[]`), []byte(`{}`), []byte(`{}`),
				true, false, false, ownerID, ownerID,
				time.Now(), time.Now(), 1,
			))

		// 4. Update last_run_at
		mock.ExpectExec(`UPDATE public\.report_schedules SET last_run_at = NOW\(\) WHERE id = \$1`).
			WithArgs(schedID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// 5. Update cache metadata
		mock.ExpectExec(`INSERT INTO public\.report_cache_metadata`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		req := httptest.NewRequest("POST", "/api/v1/reports/"+tmplID.String()+"/schedules/"+schedID.String()+"/run", nil)
		auth := security.AuthInfo{
			UserID:    ownerID,
			TenantIDs: []string{tenantID.String()},
			Roles:     []string{"user"},
		}
		req = req.WithContext(security.WithAuthInfo(req.Context(), auth))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusAccepted, w.Code)
		var res map[string]interface{}
		err = json.NewDecoder(w.Body).Decode(&res)
		require.NoError(t, err)
		assert.Equal(t, "pending", res["status"])
		assert.Equal(t, execID.String(), res["execution_id"])
		assert.Equal(t, "report-exec-"+execID.String(), res["workflow_id"])
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("TriggerScheduleRun - Returns 503 Service Unavailable with DispatchError body", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mockExec := &mockPhase3Executor{
			execFunc: func(ctx context.Context, template *reports.ReportTemplate, params map[string]interface{}) (*reports.ScheduleExecutionResult, error) {
				return nil, &reports.DispatchError{
					ExecutionID: execID,
					Err:         errors.New("temporal cluster unreachable"),
				}
			},
		}

		service := reports.NewReportService(db)
		handler := httpapi.NewReportHandler(service, mockExec, db)
		r := chi.NewRouter()
		handler.RegisterRoutes(r)

		mock.ExpectQuery(`SELECT id, tenant_id, report_definition_id, owner_id FROM public\.report_schedules WHERE id = \$1 AND tenant_id = \$2 AND is_active = true AND deleted_at IS NULL`).
			WithArgs(schedID, tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "report_definition_id", "owner_id"}).
				AddRow(schedID, tenantID, tmplID, ownerID))

		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE.*gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldCopyID))

		mock.ExpectQuery(`SELECT id, tenant_id, template_name, description, category, semantic_view_ids, layout_config, parameter_schema, is_active, is_public, is_personal, created_by_id, created_by, created_at, updated_at, version FROM public\.report_templates WHERE id = \$1 AND is_active = true AND tenant_id IN \(\$2, \$3\)`).
			WithArgs(tmplID, tenantID, goldCopyID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "template_name", "description", "category",
				"semantic_view_ids", "layout_config", "parameter_schema",
				"is_active", "is_public", "is_personal", "created_by_id", "created_by",
				"created_at", "updated_at", "version",
			}).AddRow(
				tmplID, tenantID, "Monthly PnL", "desc", "cat",
				[]byte(`[]`), []byte(`{}`), []byte(`{}`),
				true, false, false, ownerID, ownerID,
				time.Now(), time.Now(), 1,
			))

		req := httptest.NewRequest("POST", "/api/v1/reports/"+tmplID.String()+"/schedules/"+schedID.String()+"/run", nil)
		auth := security.AuthInfo{
			UserID:    ownerID,
			TenantIDs: []string{tenantID.String()},
			Roles:     []string{"user"},
		}
		req = req.WithContext(security.WithAuthInfo(req.Context(), auth))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		var res map[string]interface{}
		err = json.NewDecoder(w.Body).Decode(&res)
		require.NoError(t, err)
		assert.Equal(t, execID.String(), res["execution_id"])
		assert.Contains(t, res["error"], "temporal cluster unreachable")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetExecution - Caller is template owner (200 OK)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		service := reports.NewReportService(db)
		handler := httpapi.NewReportHandler(service, nil, db)
		r := chi.NewRouter()
		handler.RegisterRoutes(r)

		execRows := sqlmock.NewRows([]string{
			"id", "tenant_id", "template_id", "report_key", "status", "parameters",
			"output_url", "output_size_bytes", "rows_processed", "execution_time_ms",
			"error_message", "workflow_id", "run_id", "requested_by", "triggered_by",
			"metadata", "created_at", "completed_at",
			"is_personal", "created_by_id",
		}).AddRow(
			execID, tenantID, tmplID, "Monthly PnL", "completed", []byte(`{}`),
			"s3://reports/out.pdf", int64(1024), int64(50), int64(300),
			nil, "wf-1", "run-1", ownerID, "admin-trigger",
			[]byte(`{}`), time.Now(), time.Now(),
			false, ownerID,
		)

		mock.ExpectQuery(`SELECT e\.id, e\.tenant_id, e\.template_id, e\.report_key, e\.status, e\.parameters.*FROM public\.report_executions e JOIN public\.report_templates t ON t\.id = e\.template_id WHERE e\.id = \$1 AND \(e\.tenant_id = \$2 OR e\.triggered_by = \$3\)`).
			WithArgs(execID, tenantID, ownerID).
			WillReturnRows(execRows)

		req := httptest.NewRequest("GET", "/api/v1/reports/executions/"+execID.String(), nil)
		auth := security.AuthInfo{
			UserID:    ownerID,
			TenantIDs: []string{tenantID.String()},
			Roles:     []string{"user"},
		}
		req = req.WithContext(security.WithAuthInfo(req.Context(), auth))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var res map[string]interface{}
		err = json.NewDecoder(w.Body).Decode(&res)
		require.NoError(t, err)
		assert.Equal(t, execID.String(), res["id"])
		assert.Equal(t, "completed", res["status"])
		assert.Equal(t, "s3://reports/out.pdf", res["output_url"])
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetExecution - Caller is cross-tenant trigger user (200 OK)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		service := reports.NewReportService(db)
		handler := httpapi.NewReportHandler(service, nil, db)
		r := chi.NewRouter()
		handler.RegisterRoutes(r)

		callerTenant := uuid.New() // foreign tenant
		triggerUser := "cross-tenant-caller"

		execRows := sqlmock.NewRows([]string{
			"id", "tenant_id", "template_id", "report_key", "status", "parameters",
			"output_url", "output_size_bytes", "rows_processed", "execution_time_ms",
			"error_message", "workflow_id", "run_id", "requested_by", "triggered_by",
			"metadata", "created_at", "completed_at",
			"is_personal", "created_by_id",
		}).AddRow(
			execID, tenantID, tmplID, "Core Valuation", "pending", []byte(`{}`),
			nil, nil, nil, nil,
			nil, "wf-2", "run-2", ownerID, triggerUser,
			[]byte(`{}`), time.Now(), nil,
			false, ownerID,
		)

		// Matched on e.triggered_by = $3
		mock.ExpectQuery(`SELECT e\.id, e\.tenant_id, e\.template_id, e\.report_key, e\.status, e\.parameters.*FROM public\.report_executions e JOIN public\.report_templates t ON t\.id = e\.template_id WHERE e\.id = \$1 AND \(e\.tenant_id = \$2 OR e\.triggered_by = \$3\)`).
			WithArgs(execID, callerTenant, triggerUser).
			WillReturnRows(execRows)

		req := httptest.NewRequest("GET", "/api/v1/reports/executions/"+execID.String(), nil)
		auth := security.AuthInfo{
			UserID:    triggerUser,
			TenantIDs: []string{callerTenant.String()},
			Roles:     []string{"user"},
		}
		req = req.WithContext(security.WithAuthInfo(req.Context(), auth))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var res map[string]interface{}
		err = json.NewDecoder(w.Body).Decode(&res)
		require.NoError(t, err)
		assert.Equal(t, execID.String(), res["id"])
		assert.Equal(t, "pending", res["status"])
		assert.Equal(t, triggerUser, res["triggered_by"])
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetExecution - Unrelated user returns 404 (zero existence leak)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		service := reports.NewReportService(db)
		handler := httpapi.NewReportHandler(service, nil, db)
		r := chi.NewRouter()
		handler.RegisterRoutes(r)

		intruderTenant := uuid.New()
		intruderUser := "intruder-user"

		// Query returns sql.ErrNoRows due to predicate mismatch
		mock.ExpectQuery(`SELECT e\.id, e\.tenant_id, e\.template_id, e\.report_key, e\.status, e\.parameters.*FROM public\.report_executions e JOIN public\.report_templates t ON t\.id = e\.template_id WHERE e\.id = \$1 AND \(e\.tenant_id = \$2 OR e\.triggered_by = \$3\)`).
			WithArgs(execID, intruderTenant, intruderUser).
			WillReturnError(sql.ErrNoRows)

		req := httptest.NewRequest("GET", "/api/v1/reports/executions/"+execID.String(), nil)
		auth := security.AuthInfo{
			UserID:    intruderUser,
			TenantIDs: []string{intruderTenant.String()},
			Roles:     []string{"user"},
		}
		req = req.WithContext(security.WithAuthInfo(req.Context(), auth))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Execution not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Route Specificity - GET /api/v1/reports/executions does not 400 Invalid UUID", func(t *testing.T) {
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		service := reports.NewReportService(db)
		handler := httpapi.NewReportHandler(service, nil, db)
		r := chi.NewRouter()
		handler.RegisterRoutes(r)

		req := httptest.NewRequest("GET", "/api/v1/reports/executions", nil)
		auth := security.AuthInfo{
			UserID:    ownerID,
			TenantIDs: []string{tenantID.String()},
			Roles:     []string{"user"},
		}
		req = req.WithContext(security.WithAuthInfo(req.Context(), auth))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		// Assert route is not shadowed by /{id} which returns 400 "Invalid template ID"
		assert.NotEqual(t, http.StatusBadRequest, w.Code, "GET /executions was routed to /{id} and returned 400!")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}



