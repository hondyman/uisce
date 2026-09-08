package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hondyman/uisce/backend/internal/security"
)

func gateRequest(path string, authed bool) *http.Request {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	if authed {
		r = r.WithContext(security.WithAuthInfo(r.Context(), security.AuthInfo{UserID: "u1"}))
	}
	return r
}

func runGate(cfg AuthGateConfig, req *http.Request) (calledNext bool, status int) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledNext = true
		w.WriteHeader(http.StatusOK)
	})
	rr := httptest.NewRecorder()
	AuthGateMiddleware(cfg)(next).ServeHTTP(rr, req)
	return calledNext, rr.Code
}

func TestAuthGate_Off_AlwaysProceedsRegardlessOfAuth(t *testing.T) {
	cfg := AuthGateConfig{Mode: AuthGateOff}
	if called, _ := runGate(cfg, gateRequest("/api/anything", false)); !called {
		t.Fatal("off mode must always call next")
	}
}

func TestAuthGate_Shadow_LogsButNeverBlocks(t *testing.T) {
	cfg := AuthGateConfig{Mode: AuthGateShadow}
	called, status := runGate(cfg, gateRequest("/api/anything", false))
	if !called || status != http.StatusOK {
		t.Fatalf("shadow mode must never change response behavior, got called=%v status=%d", called, status)
	}
}

func TestAuthGate_Enforce_BlocksUnauthenticatedNonAllowlisted(t *testing.T) {
	cfg := AuthGateConfig{Mode: AuthGateEnforce}
	called, status := runGate(cfg, gateRequest("/api/anything", false))
	if called || status != http.StatusUnauthorized {
		t.Fatalf("enforce mode must reject with 401, got called=%v status=%d", called, status)
	}
}

func TestAuthGate_Enforce_AllowsAuthenticated(t *testing.T) {
	cfg := AuthGateConfig{Mode: AuthGateEnforce}
	called, status := runGate(cfg, gateRequest("/api/anything", true))
	if !called || status != http.StatusOK {
		t.Fatalf("enforce mode must pass authenticated callers through, got called=%v status=%d", called, status)
	}
}

func TestAuthGate_Enforce_AllowlistedBypassesAuthRequirement(t *testing.T) {
	cfg := AuthGateConfig{Mode: AuthGateEnforce, Allowlist: []string{"/health"}}
	called, status := runGate(cfg, gateRequest("/health", false))
	if !called || status != http.StatusOK {
		t.Fatalf("allowlisted route must pass through even unauthenticated, got called=%v status=%d", called, status)
	}
}

func TestAuthGate_Enforce_PrefixAllowlist(t *testing.T) {
	cfg := AuthGateConfig{Mode: AuthGateEnforce, Allowlist: []string{"/api/public/*"}}
	called, _ := runGate(cfg, gateRequest("/api/public/dax/functions", false))
	if !called {
		t.Fatal("prefix-allowlisted route must pass through unauthenticated")
	}
	called2, status2 := runGate(cfg, gateRequest("/api/private/dax/functions", false))
	if called2 || status2 != http.StatusUnauthorized {
		t.Fatal("a route not matching the prefix must still be gated")
	}
}

func TestAuthGateModeFromEnv_DefaultsToOff(t *testing.T) {
	t.Setenv("AUTH_GATE_MODE", "")
	if got := AuthGateModeFromEnv(); got != AuthGateOff {
		t.Fatalf("got %q, want off", got)
	}
	t.Setenv("AUTH_GATE_MODE", "bogus")
	if got := AuthGateModeFromEnv(); got != AuthGateOff {
		t.Fatalf("unrecognized value must default to off, got %q", got)
	}
}

func TestAuthGateModeFromEnv_Shadow(t *testing.T) {
	t.Setenv("AUTH_GATE_MODE", "shadow")
	if got := AuthGateModeFromEnv(); got != AuthGateShadow {
		t.Fatalf("got %q, want shadow", got)
	}
}
