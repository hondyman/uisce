package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hondyman/semlayer/backend/internal/identity"
	"github.com/hondyman/semlayer/backend/internal/services"
)

func TestAuthContextMiddleware_WithAPIKey(t *testing.T) {
	sm := services.NewSecurityManager(nil, nil, []byte("test-secret"))
	// generate an API key for user
	key := sm.GenerateAPIKey("user-api", "tenant-123", []string{"admin"})

	handler := AuthContextMiddleware(sm, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := identity.ActorIDFromContext(r.Context())
		if !ok || uid == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(uid))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", key)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "user-api" {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestAuthContextMiddleware_WithJWT(t *testing.T) {
	sm := services.NewSecurityManager(nil, nil, []byte("jwt-secret"))
	token, err := sm.SignToken(map[string]interface{}{"user_id": "user-jwt"})
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	handler := AuthContextMiddleware(sm, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := identity.ActorIDFromContext(r.Context())
		if !ok || uid == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(uid))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "user-jwt" {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestAuthContextMiddleware_ProfessionalServices_NotCoreAdmin(t *testing.T) {
	sm := services.NewSecurityManager(nil, nil, []byte("jwt-secret"))
	token, err := sm.SignToken(map[string]interface{}{
		"user_id":      "ali.g",
		"tenant_id":    "tenant-investco",
		"operator_role": "professional_services",
	})
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	var coreAdmin string
	var userRole string
	var tenantID string
	handler := AuthContextMiddleware(sm, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		coreAdmin = r.Header.Get("X-Is-Core-Admin")
		userRole = r.Header.Get("X-User-Role")
		tenantID = r.Header.Get("X-Tenant-ID")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/tenants/accessible", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if coreAdmin != "" {
		t.Fatalf("professional_services should not be marked as core admin, got %q", coreAdmin)
	}
	if userRole != "professional_services" {
		t.Fatalf("expected X-User-Role professional_services, got %q", userRole)
	}
	if tenantID != "tenant-investco" {
		t.Fatalf("expected X-Tenant-ID tenant-investco, got %q", tenantID)
	}
}

func TestAuthContextMiddleware_GlobalAdmin_IsCoreAdmin(t *testing.T) {
	sm := services.NewSecurityManager(nil, nil, []byte("jwt-secret"))
	token, err := sm.SignToken(map[string]interface{}{
		"user_id":       "jim.g",
		"operator_role": "global_admin",
	})
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	var coreAdmin string
	handler := AuthContextMiddleware(sm, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		coreAdmin = r.Header.Get("X-Is-Core-Admin")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/tenants/accessible", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if coreAdmin != "true" {
		t.Fatalf("global_admin should be marked as core admin, got %q", coreAdmin)
	}
}
