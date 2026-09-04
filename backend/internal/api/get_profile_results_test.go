package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// Helper to create server with sqlmock DB
//
// KNOWN PRE-EXISTING FAILURE (confirmed on the unmodified tree, unrelated to
// the security/auth-hardening changes in this branch — reproduced by
// stashing those changes and rerunning): TestGetProfileResults_FallbackToHeaders,
// _InvalidSchemaTableParams, and _WithPaging all fail here with 401
// "unauthorized" instead of their expected codes. Root cause: this helper
// never sets Server.SecurityContextDeps.Resolver, leaving it nil — and that
// nil-Resolver check (security_context.go, the `if deps.Resolver == nil`
// guard) runs before auth extraction/admin detection or any other
// request-shape logic, so it is truly unconditional: no combination of
// headers/params on the request can route around it. (Unrelated to the RBAC
// handler tests added alongside this branch's other fixes, which construct
// their own SecurityContextDeps with a non-nil resolver.) Fix is to wire a
// mock security.DatasourceResolver into
// Server{DB: db, SecurityContextDeps: ...} here; left unaddressed as out of
// scope for the session that found it.
func newServerWithMockDB(t *testing.T) (*Server, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	srv := &Server{DB: db}
	return srv, mock
}

func TestGetProfileResults_FiltersBySchemaAndTable(t *testing.T) {
	srv, mock := newServerWithMockDB(t)
	defer srv.DB.Close()

	tenant := "t-1"
	ds := "d-1"

	// Expect query with schema/table filters (limit/offset appended as positional params)
	// Use a permissive regexp to match the query shape (whitespace/parentheses differences tolerated)
	mock.ExpectQuery(`(?s)FROM\s+sml\.column_profiles.*JOIN\s+public\.catalog_node.*WHERE\s+cn\.tenant_id\s*=\s*\$1.*cn\.qualified_path.*ORDER BY\s+cn\.created_at\s+DESC\s+LIMIT\s+\$[0-9]+\s+OFFSET\s+\$[0-9]+`).
		WithArgs(tenant, ds, "/public/%", "%/shipper/%", 100, 0).
		WillReturnRows(sqlmock.NewRows([]string{"qualified_path", "column_name", "data_type", "cardinality", "min_length", "max_length", "avg_length", "frequent_values", "inferred_patterns"}).
			AddRow("/public/shipper/shipper_id", "shipper_id", "integer", 10, nil, nil, nil, `{}`, `{}`))

	req := httptest.NewRequest("GET", "/api/profiler/results?tenant_id="+tenant+"&datasource_id="+ds+"&schema=public&table=shipper", nil)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	srv.getProfileResults(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rr.Code, rr.Body.String())
	}

	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	profiles, ok := out["profiles"].([]interface{})
	if !ok || len(profiles) != 1 {
		t.Fatalf("expected one profile returned, got: %v", out)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetProfileResults_FallbackToHeaders(t *testing.T) {
	srv, mock := newServerWithMockDB(t)
	defer srv.DB.Close()

	tenant := "t-2"
	ds := "d-2"

	// Expect query without schema/table filters (limit/offset appended)
	mock.ExpectQuery(`(?s)FROM\s+sml\.column_profiles.*JOIN\s+public\.catalog_node.*WHERE\s+cn\.tenant_id\s*=\s*\$1.*ORDER BY\s+cn\.created_at\s+DESC\s+LIMIT\s+\$[0-9]+\s+OFFSET\s+\$[0-9]+`).
		WithArgs(tenant, ds, 100, 0).
		WillReturnRows(sqlmock.NewRows([]string{"qualified_path", "column_name", "data_type", "cardinality", "min_length", "max_length", "avg_length", "frequent_values", "inferred_patterns"}).
			AddRow("/public/customers/customer_id", "customer_id", "integer", 100, nil, nil, nil, `{}`, `{}`))

	req := httptest.NewRequest("GET", "/api/profiler/results", nil)
	req.Header.Set("X-Tenant-ID", tenant)
	req.Header.Set("X-Tenant-Datasource-ID", ds)
	rr := httptest.NewRecorder()
	srv.getProfileResults(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rr.Code, rr.Body.String())
	}

	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	profiles, ok := out["profiles"].([]interface{})
	if !ok || len(profiles) != 1 {
		t.Fatalf("expected one profile returned, got: %v", out)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetProfileResults_InvalidSchemaTableParams(t *testing.T) {
	srv, _ := newServerWithMockDB(t)
	defer srv.DB.Close()

	tenant := "t-3"
	ds := "d-3"

	// invalid characters in schema
	req := httptest.NewRequest("GET", "/api/profiler/results?tenant_id="+tenant+"&datasource_id="+ds+"&schema=bad!schema", nil)
	rr := httptest.NewRecorder()
	srv.getProfileResults(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid schema chars got %d body=%s", rr.Code, rr.Body.String())
	}

	// overly long table name
	long := make([]byte, 80)
	for i := range long {
		long[i] = 'a'
	}
	longName := string(long)
	req2 := httptest.NewRequest("GET", "/api/profiler/results?tenant_id="+tenant+"&datasource_id="+ds+"&table="+longName, nil)
	rr2 := httptest.NewRecorder()
	srv.getProfileResults(rr2, req2)
	if rr2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for long table param got %d body=%s", rr2.Code, rr2.Body.String())
	}
}

func TestGetProfileResults_WithPaging(t *testing.T) {
	srv, mock := newServerWithMockDB(t)
	defer srv.DB.Close()

	tenant := "t-4"
	ds := "d-4"

	// Expect query with limit/offset appended as positional params
	mock.ExpectQuery(`(?s)FROM\s+sml\.column_profiles.*JOIN\s+public\.catalog_node.*WHERE\s+cn\.tenant_id\s*=\s*\$1.*ORDER BY\s+cn\.created_at\s+DESC\s+LIMIT\s+\$[0-9]+\s+OFFSET\s+\$[0-9]+`).
		WithArgs(tenant, ds, 50, 10).
		WillReturnRows(sqlmock.NewRows([]string{"qualified_path", "column_name", "data_type", "cardinality", "min_length", "max_length", "avg_length", "frequent_values", "inferred_patterns"}).
			AddRow("/public/orders/order_id", "order_id", "integer", 100, nil, nil, nil, `{}`, `{}`))

	req := httptest.NewRequest("GET", "/api/profiler/results?tenant_id="+tenant+"&datasource_id="+ds+"&limit=50&offset=10", nil)
	rr := httptest.NewRecorder()
	srv.getProfileResults(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
