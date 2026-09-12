package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	api "github.com/hondyman/uisce/backend/internal/api"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminGetExecution_Redaction_PersonalExecution_KeysAbsent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.MatchExpectationsInOrder(false)

	handler := api.NewAdminReportHandler(db, db)
	execID := uuid.New()
	tenantID := uuid.New()
	templateID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "template_id", "schedule_id", "report_key", "status",
		"parameters", "output_url", "output_size_bytes", "rows_processed",
		"execution_time_ms", "error_message", "workflow_id", "run_id",
		"requested_by", "triggered_by", "metadata", "created_at", "completed_at",
		"is_personal", "created_by_id",
	}).AddRow(
		execID, tenantID, templateID, nil, "test-report", "completed",
		[]byte(`{"key":"value"}`), "https://example.com/output.pdf", int64(1024), int64(100),
		int64(500), "", "", "",
		"user1", "user1", nil, now, nil,
		true, "user1",
	)

	mock.ExpectQuery("SELECT e.id, e.tenant_id, e.template_id, e.schedule_id, e.report_key, e.status,").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(rows)

	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO public.admin_audit_logs").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/v1/admin/report-executions/"+execID.String(), nil)
	auth := security.AuthInfo{
		UserID:        "admin-user",
		TenantIDs:     []string{tenantID.String()},
		Roles:         []string{"global_admin"},
		IsGlobalAdmin: true,
	}
	req = req.WithContext(security.WithAuthInfo(req.Context(), auth))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	keys := make(map[string]bool)
	for k := range resp {
		keys[k] = true
	}

	assert.NotContains(t, keys, "parameters", "personal execution must not include parameters key")
	assert.NotContains(t, keys, "output_url", "personal execution must not include output_url key")
	assert.Equal(t, true, resp["parameters_redacted"], "parameters_redacted must be true")

	assert.Contains(t, keys, "id")
	assert.Contains(t, keys, "tenant_id")
	assert.Contains(t, keys, "status")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminGetExecution_Redaction_NonPersonalExecution_KeysPresent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := api.NewAdminReportHandler(db, db)
	execID := uuid.New()
	tenantID := uuid.New()
	templateID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "template_id", "schedule_id", "report_key", "status",
		"parameters", "output_url", "output_size_bytes", "rows_processed",
		"execution_time_ms", "error_message", "workflow_id", "run_id",
		"requested_by", "triggered_by", "metadata", "created_at", "completed_at",
		"is_personal", "created_by_id",
	}).AddRow(
		execID, tenantID, templateID, nil, "test-report", "completed",
		[]byte(`{"key":"value"}`), "https://example.com/output.pdf", int64(1024), int64(100),
		int64(500), "", "", "",
		"user1", "user1", nil, now, now,
		false, "",
	)

	mock.ExpectQuery("SELECT e.id, e.tenant_id, e.template_id, e.schedule_id, e.report_key, e.status,").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(rows)

	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO public.admin_audit_logs").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/v1/admin/report-executions/"+execID.String(), nil)
	auth := security.AuthInfo{
		UserID:        "admin-user",
		TenantIDs:     []string{tenantID.String()},
		Roles:         []string{"global_admin"},
		IsGlobalAdmin: true,
	}
	req = req.WithContext(security.WithAuthInfo(req.Context(), auth))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	keys := make(map[string]bool)
	for k := range resp {
		keys[k] = true
	}

	assert.Contains(t, keys, "parameters", "non-personal execution must include parameters key")
	assert.Contains(t, keys, "output_url", "non-personal execution must include output_url key")
	assert.NotContains(t, keys, "parameters_redacted", "non-personal execution must not have parameters_redacted flag")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminGetExecution_AuditWriteFails_Returns500(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := api.NewAdminReportHandler(db, db)
	execID := uuid.New()
	tenantID := uuid.New()
	templateID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "template_id", "schedule_id", "report_key", "status",
		"parameters", "output_url", "output_size_bytes", "rows_processed",
		"execution_time_ms", "error_message", "workflow_id", "run_id",
		"requested_by", "triggered_by", "metadata", "created_at", "completed_at",
		"is_personal", "created_by_id",
	}).AddRow(
		execID, tenantID, templateID, nil, "test-report", "completed",
		nil, "", nil, nil, nil, "", "", "",
		"user1", "user1", nil, now, nil,
		false, "",
	)

	mock.ExpectQuery("SELECT e.id, e.tenant_id, e.template_id, e.schedule_id, e.report_key, e.status,").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(rows)

	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO public.admin_audit_logs").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(context.DeadlineExceeded)
	mock.ExpectRollback()

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/v1/admin/report-executions/"+execID.String(), nil)
	auth := security.AuthInfo{
		UserID:        "admin-user",
		TenantIDs:     []string{tenantID.String()},
		Roles:         []string{"global_admin"},
		IsGlobalAdmin: true,
	}
	req = req.WithContext(security.WithAuthInfo(req.Context(), auth))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "audit log write failed")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminGetExecution_NotGlobalAdmin_Returns403(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := api.NewAdminReportHandler(db, db)
	execID := uuid.New()

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/v1/admin/report-executions/"+execID.String(), nil)
	auth := security.AuthInfo{
		UserID:        "regular-user",
		TenantIDs:     []string{uuid.New().String()},
		Roles:         []string{"user"},
		IsGlobalAdmin: false,
	}
	req = req.WithContext(security.WithAuthInfo(req.Context(), auth))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAdminListExecutions_Redaction_PersonalInList(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := api.NewAdminReportHandler(db, db)
	tenantID := uuid.New()
	templateID := uuid.New()
	execID1 := uuid.New()
	execID2 := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "template_id", "schedule_id", "report_key", "status",
		"parameters", "output_url", "output_size_bytes", "rows_processed",
		"execution_time_ms", "error_message", "workflow_id", "run_id",
		"requested_by", "triggered_by", "metadata", "created_at", "completed_at",
		"is_personal", "created_by_id",
	}).AddRow(
		execID1, tenantID, templateID, nil, "personal-report", "completed",
		[]byte(`{}`), "https://example.com/personal.pdf", int64(100), int64(10),
		int64(200), "", "", "", "user1", "user1", nil, now, nil,
		true, "user1",
	).AddRow(
		execID2, tenantID, templateID, nil, "corp-report", "completed",
		[]byte(`{}`), "https://example.com/corp.pdf", int64(200), int64(20),
		int64(300), "", "", "", "user2", "user2", nil, now, now,
		false, "",
	)

	mock.ExpectQuery("SELECT e.id, e.tenant_id, e.template_id, e.schedule_id, e.report_key, e.status,").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(rows)

	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO public.admin_audit_logs").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/v1/admin/report-executions", nil)
	auth := security.AuthInfo{
		UserID:        "admin-user",
		TenantIDs:     []string{tenantID.String()},
		Roles:         []string{"global_admin"},
		IsGlobalAdmin: true,
	}
	req = req.WithContext(security.WithAuthInfo(req.Context(), auth))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	items, ok := resp["items"].([]interface{})
	require.True(t, ok)
	require.Len(t, items, 2)

	byID := make(map[string]map[string]interface{})
	for _, item := range items {
		m := item.(map[string]interface{})
		byID[m["id"].(string)] = m
	}

	personal := byID[execID1.String()]
	assert.NotContains(t, personal, "parameters", "personal execution in list must not have parameters key")
	assert.NotContains(t, personal, "output_url", "personal execution in list must not have output_url key")
	assert.Equal(t, true, personal["parameters_redacted"], "personal execution must have parameters_redacted=true")

	corp := byID[execID2.String()]
	assert.Contains(t, corp, "parameters", "non-personal execution in list must have parameters key")
	assert.Contains(t, corp, "output_url", "non-personal execution in list must have output_url key")
	assert.NotContains(t, corp, "parameters_redacted")

	assert.NoError(t, mock.ExpectationsWereMet())
}
