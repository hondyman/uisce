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
