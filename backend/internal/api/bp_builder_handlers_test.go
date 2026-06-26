package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func setupBPBuilderTest(t *testing.T) (*BPBuilderHandlers, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	handler := NewBPBuilderHandlers(sqlxDB)

	cleanup := func() {
		_ = sqlxDB.Close()
	}

	return handler, mock, cleanup
}

func TestCreateBusinessProcess(t *testing.T) {
	handler, mock, cleanup := setupBPBuilderTest(t)
	defer cleanup()

	requestPayload := BusinessProcess{
		ProcessName: "HireEmployee",
		Entity:      "employee",
		Description: "Demo workflow",
		Steps: []BPStep{
			{ID: "step-1", StepOrder: 1, StepType: "data_entry", StepName: "Collect Info", DurationHours: 4},
		},
		IsActive:  true,
		CreatedBy: "demo-user",
		Tags:      []string{"demo", "hire"},
	}

	stepsJSON, _ := json.Marshal(requestPayload.Steps)
	tagsJSON, _ := json.Marshal(requestPayload.Tags)

	mock.ExpectExec("INSERT INTO business_processes").
		WithArgs(
			sqlmock.AnyArg(), // id
			"tenant-123",     // tenant_id
			"datasource-456", // datasource_id
			requestPayload.ProcessName,
			requestPayload.Entity,
			requestPayload.Description,
			stepsJSON,
			requestPayload.IsActive,
			requestPayload.CreatedBy,
			sqlmock.AnyArg(), // created_at
			1,                // version
			tagsJSON,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body, _ := json.Marshal(requestPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/business-processes?tenant_id=tenant-123&datasource_id=datasource-456", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler.CreateBusinessProcess(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)

	var resp BPAPIResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.True(t, resp.Success)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListBusinessProcesses(t *testing.T) {
	handler, mock, cleanup := setupBPBuilderTest(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "datasource_id", "process_name", "entity", "description",
		"steps_json", "is_active", "created_by", "created_at", "updated_at", "version", "tags_json",
	}).AddRow(
		"bp-1", "tenant-123", "datasource-456", "HireEmployee", "employee", "Demo",
		`[{"id":"step-1"}]`, true, "demo-user", "2025-01-01T00:00:00Z", nil, 1, `[]`,
	)

	mock.ExpectQuery("SELECT ").
		WithArgs("tenant-123", "datasource-456").
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/business-processes?tenant_id=tenant-123&datasource_id=datasource-456", nil)
	rr := httptest.NewRecorder()

	handler.ListBusinessProcesses(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp BPAPIResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.True(t, resp.Success)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBusinessProcess(t *testing.T) {
	handler, mock, cleanup := setupBPBuilderTest(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "datasource_id", "process_name", "entity", "description",
		"steps_json", "is_active", "created_by", "created_at", "updated_at", "version", "tags_json",
	}).AddRow(
		"bp-1", "tenant-123", "datasource-456", "HireEmployee", "employee", "Demo",
		`[{"id":"step-1"}]`, true, "demo-user", "2025-01-01T00:00:00Z", nil, 1, `[]`,
	)

	mock.ExpectQuery("SELECT ").
		WithArgs("bp-1", "tenant-123").
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/business-processes/bp-1?tenant_id=tenant-123", nil)

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", "bp-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rr := httptest.NewRecorder()

	handler.GetBusinessProcess(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp BPAPIResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.True(t, resp.Success)

	require.NoError(t, mock.ExpectationsWereMet())
}
