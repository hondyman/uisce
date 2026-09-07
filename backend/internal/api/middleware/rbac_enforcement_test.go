package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

func testRBACEnforcer(t *testing.T) (*RBACEnforcer, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	return NewRBACEnforcer(sqlx.NewDb(db, "postgres")), mock
}

func TestRequirePermission_NoAuthContext_Returns401NotPanic(t *testing.T) {
	enforcer, _ := testRBACEnforcer(t)
	handler := enforcer.RequirePermission("process.read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called when auth context is missing")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RequirePermission panicked with no auth context: %v", r)
		}
	}()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

func TestRequirePermission_WithAuthInfo_ProceedsToPermissionCheck(t *testing.T) {
	enforcer, mock := testRBACEnforcer(t)

	mock.ExpectQuery("SELECT bp_user_has_permission").
		WithArgs("user-123", "tenant-abc", "ds-456", "process.read").
		WillReturnRows(sqlmock.NewRows([]string{"has_permission"}).AddRow(true))

	nextCalled := false
	handler := enforcer.RequirePermission("process.read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Tenant-Datasource-ID", "ds-456")
	ctx := context.WithValue(context.Background(), "user_id", "user-123")
	ctx = security.WithAuthInfo(ctx, security.AuthInfo{
		UserID:    "user-123",
		TenantIDs: []string{"tenant-abc"},
		Roles:     []string{"editor"},
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code == http.StatusInternalServerError {
		t.Fatalf("permission check query error: %s", rr.Body.String())
	}
	if !nextCalled {
		t.Fatalf("handler was not called; response code: %d body: %s", rr.Code, rr.Body.String())
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

// localTenantIDFromRequest mirrors the AuthInfo branch of helpers.TenantIDFromRequest.
// This is a local copy to avoid an import cycle (middleware cannot import api).
// The test validates the 401-contract behavior only; implementation parity with
// the canonical helper is maintained manually.
func localTenantIDFromRequest(r *http.Request) (string, bool) {
	if auth, ok := security.AuthInfoFromContext(r.Context()); ok && len(auth.TenantIDs) > 0 {
		return auth.TenantIDs[0], true
	}
	return "", false
}

func TestTenantIDFromRequest_AuthInfoOnly_ReturnsTenantID(t *testing.T) {
	ctx := security.WithAuthInfo(context.Background(), security.AuthInfo{
		UserID:    "user-123",
		TenantIDs: []string{"tenant-abc"},
		Roles:     []string{"editor"},
	})

	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)

	tenantID, ok := localTenantIDFromRequest(req)
	if !ok {
		t.Fatal("expected tenantID to be resolved from AuthInfo")
	}
	if tenantID != "tenant-abc" {
		t.Fatalf("expected tenant-abc, got %s", tenantID)
	}
}

func TestTenantIDFromRequest_NoAuthNoClaims_ReturnsFalse(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	tenantID, ok := localTenantIDFromRequest(req)
	if ok {
		t.Fatalf("expected false, got true with tenantID=%s", tenantID)
	}
	if tenantID != "" {
		t.Fatalf("expected empty string, got %s", tenantID)
	}
}

func TestRequirePermission_MissingDatasource_Returns400(t *testing.T) {
	enforcer, _ := testRBACEnforcer(t)
	handler := enforcer.RequirePermission("process.read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(context.Background(), "user_id", "user-123")
	ctx = security.WithAuthInfo(ctx, security.AuthInfo{
		UserID:    "user-123",
		TenantIDs: []string{"tenant-abc"},
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
