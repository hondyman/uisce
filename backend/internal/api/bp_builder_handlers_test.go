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

	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/security"
)

type bpMockResolver struct {
	tenantID string
}

func (m *bpMockResolver) Resolve(ctx context.Context, datasourceID string) (*security.ResolvedDatasource, error) {
	return &security.ResolvedDatasource{
		DatasourceID:   datasourceID,
		TenantID:       m.tenantID,
		InstanceID:     "inst1",
		ProductID:      "prod1",
		AllowedRegions: []string{"us-east-1"},
	}, nil
}

func setupBPBuilderTest(t *testing.T, tenantID string) (*BPBuilderHandlers, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	handler := NewBPBuilderHandlers(sqlxDB, handlers.SecurityContextDeps{Resolver: &bpMockResolver{tenantID: tenantID}})

	cleanup := func() {
		_ = sqlxDB.Close()
	}

	return handler, mock, cleanup
}

func TestCreateBusinessProcess(t *testing.T) {
	handler, mock, cleanup := setupBPBuilderTest(t, "tenant-123")
	defer cleanup()
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

	mock.ExpectExec("INSERT INTO business_processes").
		WillReturnResult(sqlmock.NewResult(1, 1))

	body, _ := json.Marshal(requestPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/business-processes?tenant_id=tenant-123&datasource_id=datasource-456", bytes.NewReader(body))
	req = withAuthContext(req, "tenant-123")
	rr := httptest.NewRecorder()

	handler.CreateBusinessProcess(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)

	var resp BPAPIResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.True(t, resp.Success)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListBusinessProcesses(t *testing.T) {
	handler, mock, cleanup := setupBPBuilderTest(t, "tenant-123")
	defer cleanup()

	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "datasource_id", "process_name", "entity", "description",
		"steps_json", "is_active", "created_by", "created_at", "updated_at", "version", "tags_json",
	}).AddRow(
		"bp-1", "tenant-123", "datasource-456", "HireEmployee", "employee", "Demo",
		`[{"id":"step-1"}]`, true, "demo-user", "2025-01-01T00:00:00Z", nil, 1, `[]`,
	)

	mock.ExpectQuery("SELECT ").
		WithArgs("tenant-123", sqlmock.AnyArg()).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/business-processes?tenant_id=tenant-123&datasource_id=datasource-456", nil)
	req = withAuthContext(req, "tenant-123")
	rr := httptest.NewRecorder()

	handler.ListBusinessProcesses(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp BPAPIResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.True(t, resp.Success)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBusinessProcess(t *testing.T) {
	handler, mock, cleanup := setupBPBuilderTest(t, "tenant-123")
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
	req = withAuthContext(req, "tenant-123")
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
