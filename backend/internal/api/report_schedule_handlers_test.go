package api_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

	t.Run("CreateSchedule - Legacy creates retired (410 Gone)", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]interface{}{
			"schedule_name":   "Daily Valuation",
			"cron_expression": "0 8 * * 1-5",
		})
		req := authRequest(http.MethodPost, "/api/v1/reports/"+tmplID+"/schedules", payload, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusGone, w.Code)
		assert.Contains(t, w.Body.String(), "legacy_report_schedules_retired")
	})

	t.Run("CreateSchedule - Empty Cron Expression also Gone (410)", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]interface{}{
			"schedule_name":   "Daily Valuation",
			"cron_expression": "",
		})
		req := authRequest(http.MethodPost, "/api/v1/reports/"+tmplID+"/schedules", payload, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusGone, w.Code)
		assert.Contains(t, w.Body.String(), "legacy_report_schedules_retired")
	})

	t.Run("CreateSchedule - Invalid UUID in path also Gone (410)", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]interface{}{
			"schedule_name":   "Daily Valuation",
			"cron_expression": "0 8 * * 1-5",
		})
		req := authRequest(http.MethodPost, "/api/v1/reports/invalid-uuid/schedules", payload, tenantID, userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusGone, w.Code)
	})
}

func TestReportScheduleAPI_SqlmockScenarios(t *testing.T) {
	db, mock, r := setupScheduleTestRouter(t)
	_ = db
	tenantID := uuid.New()
	userID := "user-alice"
	tmplID := uuid.New()
	schedID := uuid.New()

	t.Run("CreateSchedule - Legacy creates retired (410 Gone)", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]interface{}{
			"schedule_name":   "Daily Valuation",
			"cron_expression": "0 8 * * 1-5",
			"export_format":   "PDF",
			"notify_in_app":   true,
		})

		req := authRequest(http.MethodPost, "/api/v1/reports/"+tmplID.String()+"/schedules", payload, tenantID.String(), userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusGone, w.Code)
		assert.Contains(t, w.Body.String(), "legacy_report_schedules_retired")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CreateSchedule - Gone even when template would be invisible", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]interface{}{
			"schedule_name":   "Secret Sched",
			"cron_expression": "0 8 * * 1-5",
		})

		req := authRequest(http.MethodPost, "/api/v1/reports/"+tmplID.String()+"/schedules", payload, tenantID.String(), userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusGone, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DeleteSchedule - Legacy retired (410 Gone)", func(t *testing.T) {
		req := authRequest(http.MethodDelete, "/api/v1/reports/"+tmplID.String()+"/schedules/"+schedID.String(), nil, tenantID.String(), userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusGone, w.Code)
		assert.Contains(t, w.Body.String(), "legacy_report_schedules_retired")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ListSchedulesForTemplate - Legacy retired (410 Gone)", func(t *testing.T) {
		req := authRequest(http.MethodGet, "/api/v1/reports/"+tmplID.String()+"/schedules", nil, tenantID.String(), userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusGone, w.Code)
		assert.Contains(t, w.Body.String(), "legacy_report_schedules_retired")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("TriggerScheduleRun - Legacy retired (410 Gone)", func(t *testing.T) {
		req := authRequest(http.MethodPost, "/api/v1/reports/"+tmplID.String()+"/schedules/"+schedID.String()+"/run", nil, tenantID.String(), userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusGone, w.Code)
		assert.Contains(t, w.Body.String(), "legacy_report_schedules_retired")
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

		req := httptest.NewRequest(http.MethodPost, "/api/reports/schedules", bytes.NewBufferString(`{"schedule_name":"Test"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Reject Spoofed Header in Production on CreateSchedule", func(t *testing.T) {

		req := httptest.NewRequest(http.MethodPost, "/api/reports/schedules", bytes.NewBufferString(`{"schedule_name":"Test"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", uuid.New().String())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Unauthenticated ListSchedules Returns 401 Unauthorized", func(t *testing.T) {

		req := httptest.NewRequest(http.MethodGet, "/api/reports/schedules", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Reject Spoofed Header on GetBatchTelemetry When DB Has No Batch", func(t *testing.T) {
		batchID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/api/reports/batches/"+batchID.String()+"/telemetry", nil)
		req.Header.Set("X-Tenant-ID", uuid.New().String())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
