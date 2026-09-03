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
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/api"
	"github.com/hondyman/uisce/backend/internal/handlers"
)

// setupNodeInstanceTest wires the real chi router through
// api.RegisterNodeTypesRoutes so requests exercise routing + handler
// together, the same way they do in production.
func setupNodeInstanceTest(t *testing.T) (chi.Router, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	r := chi.NewRouter()
	api.RegisterNodeTypesRoutes(r, db, handlers.SecurityContextDeps{Resolver: &mockResolver{}})

	cleanup := func() { _ = db.Close() }
	return r, mock, cleanup
}

func TestHandleCreateNode_Success(t *testing.T) {
	r, mock, cleanup := setupNodeInstanceTest(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("INSERT INTO catalog_node").
		WithArgs(sqlmock.AnyArg(), "Acme Bond", sqlmock.AnyArg(), "type-1", "ten",
			"ds1", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), true).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(now, now))

	payload := map[string]interface{}{
		"node_name":   "Acme Bond",
		"description": "A generic asset",
		"is_active":   true,
		"properties":  map[string]interface{}{"isin": "US1234567890"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/node-types/type-1/nodes", bytes.NewReader(body))
	req = withValidHeaders(req, "ten", "ds1")
	req = withAuth(req, "ten")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var resp api.CatalogNodeInstance
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "Acme Bond", resp.NodeName)
	require.Equal(t, "type-1", resp.NodeTypeID)
	require.Equal(t, "US1234567890", resp.Properties["isin"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleCreateNode_MissingName(t *testing.T) {
	r, _, cleanup := setupNodeInstanceTest(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{"description": "no name"})
	req := httptest.NewRequest(http.MethodPost, "/node-types/type-1/nodes", bytes.NewReader(body))
	req = withValidHeaders(req, "ten", "ds1")
	req = withAuth(req, "ten")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleGetNode_Success(t *testing.T) {
	r, mock, cleanup := setupNodeInstanceTest(t)
	defer cleanup()

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "node_name", "description", "node_type_id", "tenant_id", "tenant_datasource_id",
		"parent_id", "qualified_path", "properties", "config", "is_active", "created_at", "updated_at",
	}).AddRow("node-1", "Acme Bond", "desc", "type-1", "ten", "ds1", nil, "acme-bond",
		[]byte(`{"isin":"US1234567890"}`), []byte(`{}`), true, now, now)

	mock.ExpectQuery("SELECT id, node_name, description, node_type_id, tenant_id, tenant_datasource_id").
		WithArgs("node-1", "ten").
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/nodes/node-1", nil)
	req = withValidHeaders(req, "ten", "ds1")
	req = withAuth(req, "ten")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp api.CatalogNodeInstance
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "Acme Bond", resp.NodeName)
	require.Equal(t, "US1234567890", resp.Properties["isin"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleGetNode_NotFound(t *testing.T) {
	r, mock, cleanup := setupNodeInstanceTest(t)
	defer cleanup()

	mock.ExpectQuery("SELECT id, node_name, description, node_type_id, tenant_id, tenant_datasource_id").
		WithArgs("missing", "ten").
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/nodes/missing", nil)
	req = withValidHeaders(req, "ten", "ds1")
	req = withAuth(req, "ten")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleUpdateNode_Success(t *testing.T) {
	r, mock, cleanup := setupNodeInstanceTest(t)
	defer cleanup()

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "node_name", "description", "node_type_id", "tenant_id", "tenant_datasource_id",
		"parent_id", "qualified_path", "properties", "config", "is_active", "created_at", "updated_at",
	}).AddRow("node-1", "Acme Bond Renamed", "updated desc", "type-1", "ten", "ds1", nil, nil,
		[]byte(`{"isin":"US1234567890"}`), []byte(`{}`), true, now, now)

	mock.ExpectQuery("UPDATE catalog_node").
		WithArgs("Acme Bond Renamed", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), true, "node-1", "ten").
		WillReturnRows(rows)

	payload := map[string]interface{}{
		"node_name":   "Acme Bond Renamed",
		"description": "updated desc",
		"is_active":   true,
		"properties":  map[string]interface{}{"isin": "US1234567890"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/nodes/node-1", bytes.NewReader(body))
	req = withValidHeaders(req, "ten", "ds1")
	req = withAuth(req, "ten")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp api.CatalogNodeInstance
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "Acme Bond Renamed", resp.NodeName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleDeleteNode_Success(t *testing.T) {
	r, mock, cleanup := setupNodeInstanceTest(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM catalog_node").
		WithArgs("node-1", "ten").
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodDelete, "/nodes/node-1", nil)
	req = withValidHeaders(req, "ten", "ds1")
	req = withAuth(req, "ten")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleDeleteNode_NotFound(t *testing.T) {
	r, mock, cleanup := setupNodeInstanceTest(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM catalog_node").
		WithArgs("missing", "ten").
		WillReturnResult(sqlmock.NewResult(0, 0))

	req := httptest.NewRequest(http.MethodDelete, "/nodes/missing", nil)
	req = withValidHeaders(req, "ten", "ds1")
	req = withAuth(req, "ten")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

// TestHandleGetNodesForType_DatasourceFilterParamFix is a regression test for a
// bug where the datasource filter clause referenced SQL placeholder $4 while
// only 3 args were ever bound, so any request scoped to a specific datasource
// would fail with "there is no parameter $4" instead of returning results.
func TestHandleGetNodesForType_DatasourceFilterParamFix(t *testing.T) {
	r, mock, cleanup := setupNodeInstanceTest(t)
	defer cleanup()

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "node_name", "description", "node_type_id", "tenant_id", "tenant_datasource_id",
		"properties", "config", "created_at", "updated_at",
	}).AddRow("node-1", "Acme Bond", "desc", "type-1", "ten", "ds1", []byte(`{}`), []byte(`{}`), now, now)

	mock.ExpectQuery("SELECT id, node_name, description, node_type_id, tenant_id, tenant_datasource_id").
		WithArgs("type-1", "ten", "ds1").
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/node-types/type-1/nodes", nil)
	req = withValidHeaders(req, "ten", "ds1")
	req = withAuth(req, "ten")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}
