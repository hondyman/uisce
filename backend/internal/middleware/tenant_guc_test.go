package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/db"
)

// TestTenantGUCSetsAppCurrentTenant verifies that the middleware pins a
// connection, sets app.current_tenant + uisce.current_tenant on it, and
// resets before returning to the pool.
func TestTenantGUCSetsAppCurrentTenant(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer mockDB.Close()

	tenantID := uuid.New().String()

	mock.ExpectExec("SELECT set_config('app.current_tenant', $1, false)").
		WithArgs(tenantID).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("SELECT set_config('uisce.current_tenant', $1, false)").
		WithArgs(tenantID).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("SELECT set_config('app.current_tenant', '', false)").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("SELECT set_config('uisce.current_tenant', '', false)").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("SELECT set_config('app.shared_reference_tenant', '', false)").
		WillReturnResult(sqlmock.NewResult(0, 0))

	var sawConn atomic.Bool
	h := TenantGUC(TenantGUCConfig{DB: mockDB})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if PinnedConn(r.Context()) == nil {
			t.Error("handler: PinnedConn returned nil")
		} else {
			sawConn.Store(true)
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Seed the request context with a tenant ID using the same key the
	// real WithTenantContext middleware uses (db.TenantContextKey).
	ctx := context.WithValue(context.Background(), db.TenantContextKey, &db.TenantCtx{TenantID: tenantID})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}
	if !sawConn.Load() {
		t.Error("handler did not see pinned connection")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestTenantGUCSkipsHealthPath verifies the SkipPaths bypass.
func TestTenantGUCSkipsHealthPath(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer mockDB.Close()
	// No expectations: skipped path should not touch the DB.

	called := false
	h := TenantGUC(TenantGUCConfig{DB: mockDB})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if PinnedConn(r.Context()) != nil {
			t.Error("skipped path should not pin connection")
		}
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !called {
		t.Fatal("downstream not called")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("skipped path emitted SQL: %v", err)
	}
}

// TestTenantGUCFallsThroughWithoutTenant verifies that requests without a
// tenant in context bypass GUC injection (rather than 500ing).
func TestTenantGUCFallsThroughWithoutTenant(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer mockDB.Close()

	called := false
	h := TenantGUC(TenantGUCConfig{DB: mockDB})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !called {
		t.Fatal("downstream not called")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("no-tenant path emitted SQL: %v", err)
	}
}

// TestTenantGUCRejectsNonUUIDTenant verifies that a malformed tenant ID
// triggers OnError (or 500 default) rather than silently corrupting state.
func TestTenantGUCRejectsNonUUIDTenant(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer mockDB.Close()

	called := false
	h := TenantGUC(TenantGUCConfig{DB: mockDB, OnError: func(w http.ResponseWriter, r *http.Request, err error) {
		if !strings.Contains(err.Error(), "non-UUID") {
			t.Errorf("OnError got %v, want non-UUID mention", err)
		}
		w.WriteHeader(http.StatusBadRequest)
	}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	ctx := context.WithValue(context.Background(), db.TenantContextKey, &db.TenantCtx{TenantID: "not-a-uuid"})
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if called {
		t.Error("downstream called despite malformed tenant")
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d want 400", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Test helper: pull db.TenantContextKey into a place this test can use it.
// In production code, the actual key is in internal/db/tenant_tx.go.
// ---------------------------------------------------------------------------

// silence unused-import warnings for tooling-only declarations
