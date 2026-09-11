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
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	httpapi "github.com/hondyman/uisce/backend/internal/api"
	"github.com/hondyman/uisce/backend/internal/reports"
	"github.com/hondyman/uisce/backend/internal/security"
)

func setupScheduleTestRouter(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *chi.Mux) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	service := reports.NewReportService(db)
	executor := reports.NewTestInProcessExecutor(db)
	handler := httpapi.NewReportHandler(service, executor, db)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	return db, mock, r
}

func adminAuthRequest(method, url string, body []byte, tenantID, userID string) *http.Request {
	req := authRequest(method, url, body, tenantID, userID)
	auth := security.AuthInfo{
		UserID:    userID,
		TenantIDs: []string{tenantID},
		Roles:     []string{"admin"},
	}
	return req.WithContext(security.WithAuthInfo(req.Context(), auth))
}

func TestReportScheduleAPI_AuthAndSecurity(t *testing.T) {
	_, _, r := setupScheduleTestRouter(t)

	t.Run("Reject Unauthenticated Access", func(t *testing.T) {
		t.Setenv("ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "false")
		templateID := uuid.New().String()

		endpoints := []struct {
			method string
			path   string
		}{
			{http.MethodGet, "/api/v1/reports/" + templateID + "/schedules"},
			{http.MethodPost, "/api/v1/reports/" + templateID + "/schedules"},
			{http.MethodDelete, "/api/v1/reports/" + templateID + "/schedules/" + uuid.New().String()},
			{http.MethodPost, "/api/v1/reports/" + templateID + "/schedules/" + uuid.New().String() + "/run"},
		}

		for _, ep := range endpoints {
			req := httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString("{}"))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code, "endpoint %s %s should reject unauthenticated caller", ep.method, ep.path)
		}
	})
}

func TestReportScheduleAPI_CreateValidation(t *testing.T) {
	_, _, r := setupScheduleTestRouter(t)
	tenantID := uuid.New().String()
	userID := "user-123"
	tmplID := uuid.New().String()

	t.Run("CreateSchedule - Empty Name (400 Bad Request)", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]interface{}{
			"schedule_name":   "",
			"cron_expression": "0 8 * * 1-5",
		})
		req := authRequest(http.MethodPost, "/api/v1/reports/"+tmplID+"/schedules", payload, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "schedule_name is required")
	})

	t.Run("CreateSchedule - Empty Cron Expression (400 Bad Request)", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]interface{}{
			"schedule_name":   "Daily Valuation",
			"cron_expression": "",
		})
		req := authRequest(http.MethodPost, "/api/v1/reports/"+tmplID+"/schedules", payload, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "cron_expression is required")
	})

	t.Run("CreateSchedule - Invalid UUID in path (400 Bad Request)", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]interface{}{
			"schedule_name":   "Daily Valuation",
			"cron_expression": "0 8 * * 1-5",
		})
		req := authRequest(http.MethodPost, "/api/v1/reports/invalid-uuid/schedules", payload, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestReportScheduleAPI_SqlmockScenarios(t *testing.T) {
	db, mock, r := setupScheduleTestRouter(t)
	_ = db
	tenantID := uuid.New()
	goldCopyID := uuid.New()
	userID := "user-alice"
	tmplID := uuid.New()
	schedID := uuid.New()

	t.Run("CreateSchedule - Success (201 Created)", func(t *testing.T) {
		// Mock gold copy resolution
		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE.*gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldCopyID))

		// Mock template visibility check
		mock.ExpectQuery(`SELECT id, tenant_id, template_name, is_active, is_personal, created_by_id FROM public\.report_templates WHERE id = \$1 AND is_active = true AND tenant_id IN \(\$2, \$3\)`).
			WithArgs(tmplID, tenantID, goldCopyID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "template_name", "is_active", "is_personal", "created_by_id"}).
				AddRow(tmplID, tenantID, "Daily Report", true, false, nil))

		// Mock insert schedule
		now := time.Now()
		mock.ExpectQuery(`INSERT INTO public\.report_schedules`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "report_definition_id", "owner_id", "schedule_name",
				"cron_expression", "region", "calendar_id", "start_of_day_time",
				"unscheduled_behavior", "business_day_offset", "burst_dimension",
				"export_format", "notification_channels", "is_active", "deleted_at",
				"last_run_at", "next_run_at", "created_at",
			}).AddRow(
				schedID, tenantID, tmplID, userID, "Daily Valuation",
				"0 8 * * 1-5", "us-west", nil, "08:00:00",
				"SKIP", 0, "client_id",
				"PDF", []byte(`{"in_app":true,"email":false}`), true, nil,
				nil, nil, now,
			))

		payload, _ := json.Marshal(map[string]interface{}{
			"schedule_name":   "Daily Valuation",
			"cron_expression": "0 8 * * 1-5",
			"export_format":   "PDF",
			"notify_in_app":   true,
		})

		req := authRequest(http.MethodPost, "/api/v1/reports/"+tmplID.String()+"/schedules", payload, tenantID.String(), userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, schedID.String(), resp["id"])
		assert.Equal(t, tmplID.String(), resp["report_definition_id"])
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CreateSchedule - Template Invisible / Non-Owner Personal (404 Not Found)", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE.*gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldCopyID))

		// Template query returns sql.ErrNoRows
		mock.ExpectQuery(`SELECT id, tenant_id, template_name, is_active, is_personal, created_by_id FROM public\.report_templates`).
			WillReturnError(sql.ErrNoRows)

		payload, _ := json.Marshal(map[string]interface{}{
			"schedule_name":   "Secret Sched",
			"cron_expression": "0 8 * * 1-5",
		})

		req := authRequest(http.MethodPost, "/api/v1/reports/"+tmplID.String()+"/schedules", payload, tenantID.String(), userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DeleteSchedule - Non-Owner Non-Admin (403 Forbidden)", func(t *testing.T) {
		// Mock query checking schedule owner returns different owner
		mock.ExpectQuery(`SELECT owner_id FROM public\.report_schedules WHERE id = \$1 AND tenant_id = \$2 AND deleted_at IS NULL`).
			WithArgs(schedID, tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow("other-user"))

		req := authRequest(http.MethodDelete, "/api/v1/reports/"+tmplID.String()+"/schedules/"+schedID.String(), nil, tenantID.String(), userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "Forbidden: only schedule owner or tenant admin can delete schedule")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DeleteSchedule - Owner (204 No Content)", func(t *testing.T) {
		mock.ExpectQuery(`SELECT owner_id FROM public\.report_schedules WHERE id = \$1 AND tenant_id = \$2 AND deleted_at IS NULL`).
			WithArgs(schedID, tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(userID))

		mock.ExpectExec(`UPDATE public\.report_schedules SET is_active = false, deleted_at = NOW\(\) WHERE id = \$1 AND tenant_id = \$2`).
			WithArgs(schedID, tenantID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		req := authRequest(http.MethodDelete, "/api/v1/reports/"+tmplID.String()+"/schedules/"+schedID.String(), nil, tenantID.String(), userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ListSchedulesForTemplate - Success (200 OK)", func(t *testing.T) {
		// Mock gold copy query
		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE.*gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldCopyID))

		// Mock template visibility query
		mock.ExpectQuery(`SELECT 1 FROM public\.report_templates WHERE id = \$1 AND is_active = true AND tenant_id IN \(\$2, \$3\) AND \(is_personal = false OR \(created_by_id IS NOT NULL AND created_by_id = \$4\)\)`).
			WithArgs(tmplID, tenantID, goldCopyID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"?column?"}).AddRow(1))

		// Mock select schedules
		now := time.Now()
		mock.ExpectQuery(`SELECT id, tenant_id, report_definition_id, owner_id, schedule_name, cron_expression, region, calendar_id, start_of_day_time::text, unscheduled_behavior, business_day_offset, burst_dimension, export_format, notification_channels, is_active, deleted_at, last_run_at, next_run_at, created_at FROM public\.report_schedules WHERE tenant_id = \$1 AND report_definition_id = \$2 AND is_active = true AND deleted_at IS NULL ORDER BY created_at DESC`).
			WithArgs(tenantID, tmplID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "report_definition_id", "owner_id", "schedule_name",
				"cron_expression", "region", "calendar_id", "start_of_day_time",
				"unscheduled_behavior", "business_day_offset", "burst_dimension",
				"export_format", "notification_channels", "is_active", "deleted_at",
				"last_run_at", "next_run_at", "created_at",
			}).AddRow(
				schedID, tenantID, tmplID, userID, "Weekly Valuation",
				"0 8 * * 1", "us-west", nil, "08:00:00",
				"SKIP", 0, "client_id",
				"PDF", []byte(`{"in_app":true,"email":false}`), true, nil,
				nil, nil, now,
			))

		req := authRequest(http.MethodGet, "/api/v1/reports/"+tmplID.String()+"/schedules", nil, tenantID.String(), userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var list []map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &list)
		require.Len(t, list, 1)
		assert.Equal(t, schedID.String(), list[0]["id"])
		assert.Equal(t, "Weekly Valuation", list[0]["schedule_name"])
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ListSchedulesForTemplate - Template Invisible / 404", func(t *testing.T) {
		// Mock gold copy query
		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE.*gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldCopyID))

		// Mock template query returns sql.ErrNoRows
		mock.ExpectQuery(`SELECT 1 FROM public\.report_templates WHERE id = \$1 AND is_active = true AND tenant_id IN \(\$2, \$3\) AND \(is_personal = false OR \(created_by_id IS NOT NULL AND created_by_id = \$4\)\)`).
			WithArgs(tmplID, tenantID, goldCopyID, userID).
			WillReturnError(sql.ErrNoRows)

		req := authRequest(http.MethodGet, "/api/v1/reports/"+tmplID.String()+"/schedules", nil, tenantID.String(), userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("TriggerScheduleRun - Non-Owner Non-Admin (403 Forbidden)", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, tenant_id, report_definition_id, owner_id FROM public\.report_schedules WHERE id = \$1 AND tenant_id = \$2 AND is_active = true AND deleted_at IS NULL`).
			WithArgs(schedID, tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "report_definition_id", "owner_id"}).
				AddRow(schedID, tenantID, tmplID, "real-owner"))

		req := authRequest(http.MethodPost, "/api/v1/reports/"+tmplID.String()+"/schedules/"+schedID.String()+"/run", nil, tenantID.String(), "intruder-user")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("TriggerScheduleRun - Admin (200 OK)", func(t *testing.T) {
		adminUserID := "admin-user"

		// Query schedule
		mock.ExpectQuery(`SELECT id, tenant_id, report_definition_id, owner_id FROM public\.report_schedules WHERE id = \$1 AND tenant_id = \$2 AND is_active = true AND deleted_at IS NULL`).
			WithArgs(schedID, tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "report_definition_id", "owner_id"}).
				AddRow(schedID, tenantID, tmplID, "template-creator"))

		// Gold copy query
		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE.*gold_copy = true`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(goldCopyID))

		// Query linked template
		templateOwner := "template-creator"
		mock.ExpectQuery(`SELECT id, tenant_id, template_name, description, category, semantic_view_ids, layout_config, parameter_schema, is_active, is_public, is_personal, created_by_id, created_by, created_at, updated_at, version FROM public\.report_templates WHERE id = \$1 AND is_active = true AND tenant_id IN \(\$2, \$3\)`).
			WithArgs(tmplID, tenantID, goldCopyID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "template_name", "description", "category",
				"semantic_view_ids", "layout_config", "parameter_schema",
				"is_active", "is_public", "is_personal", "created_by_id", "created_by",
				"created_at", "updated_at", "version",
			}).AddRow(
				tmplID, tenantID, "Daily Valuation", "desc", "cat",
				[]byte(`[]`), []byte(`{}`), []byte(`{}`),
				true, false, false, templateOwner, templateOwner,
				time.Now(), time.Now(), 1,
			))

		// Mock execution insert via InProcessSyncExecutor
		mock.ExpectQuery(`INSERT INTO public\.report_executions`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "status", "output_url", "coalesce"}).
				AddRow(uuid.New(), "synthetic", "s3://reports/snapshot.pdf", templateOwner))

		// Update schedule last_run_at
		mock.ExpectExec(`UPDATE public\.report_schedules SET last_run_at = NOW\(\) WHERE id = \$1`).
			WithArgs(schedID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Insert report_cache_metadata
		mock.ExpectExec(`INSERT INTO public\.report_cache_metadata`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		req := adminAuthRequest(http.MethodPost, "/api/v1/reports/"+tmplID.String()+"/schedules/"+schedID.String()+"/run", nil, tenantID.String(), adminUserID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusAccepted, w.Code)
		var res map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		assert.Equal(t, "pending", res["status"])
		assert.NotEmpty(t, res["execution_id"])
		assert.NotEmpty(t, res["workflow_id"])
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func setupBurstScheduleTestRouter(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock, *chi.Mux) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	handler := httpapi.NewReportScheduleHandler(sqlxDB)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	return sqlxDB, mock, r
}

func TestHardenedBurstEndpoints_AuthSecurity(t *testing.T) {
	_, _, r := setupBurstScheduleTestRouter(t)

	t.Run("Reject Unauthenticated Access on CreateSchedule", func(t *testing.T) {
		t.Setenv("ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "false")

		req := httptest.NewRequest(http.MethodPost, "/api/reports/schedules", bytes.NewBufferString(`{"schedule_name":"Test"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Reject Spoofed Header in Production on CreateSchedule", func(t *testing.T) {
		t.Setenv("ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "false")

		req := httptest.NewRequest(http.MethodPost, "/api/reports/schedules", bytes.NewBufferString(`{"schedule_name":"Test"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", uuid.New().String())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Unauthenticated ListSchedules Returns 401 Unauthorized", func(t *testing.T) {
		t.Setenv("ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "false")

		req := httptest.NewRequest(http.MethodGet, "/api/reports/schedules", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Reject Spoofed Header on GetBatchTelemetry When DB Has No Batch", func(t *testing.T) {
		t.Setenv("ALLOW_CLIENT_TENANT_HEADER_FALLBACK", "false")
		batchID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/api/reports/batches/"+batchID.String()+"/telemetry", nil)
		req.Header.Set("X-Tenant-ID", uuid.New().String())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
