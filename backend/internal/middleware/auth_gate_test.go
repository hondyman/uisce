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

func runGate(cfg AuthGateConfig, req *http.Request, next http.Handler) (calledNext bool, status int, body string) {
	rr := httptest.NewRecorder()
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledNext = true
		next.ServeHTTP(w, r)
	})
	AuthGateMiddleware(cfg)(wrapped).ServeHTTP(rr, req)
	return calledNext, rr.Code, rr.Body.String()
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	})
}

func TestAuthGate_Off_AlwaysProceedsRegardlessOfAuth(t *testing.T) {
	cfg := AuthGateConfig{Mode: AuthGateOff}
	if called, _, _ := runGate(cfg, gateRequest("/api/anything", false), okHandler()); !called {
		t.Fatal("off mode must always call next")
	}
}

func TestAuthGate_Shadow_LogsButNeverBlocks(t *testing.T) {
	cfg := AuthGateConfig{Mode: AuthGateShadow}
	called, status, _ := runGate(cfg, gateRequest("/api/anything", false), okHandler())
	if !called || status != http.StatusOK {
		t.Fatalf("shadow mode must never change response behavior, got called=%v status=%d", called, status)
	}
}

// This pins the load-bearing invariant directly: shadow mode's status-
// capturing wrapper must be byte-for-byte transparent. Compare the
// response produced through the gate against the response the handler
// would produce completely unwrapped.
func TestAuthGate_Shadow_NeverAltersResponse(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "value")
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("teapot body"))
	}

	baseline := httptest.NewRecorder()
	http.HandlerFunc(handler).ServeHTTP(baseline, gateRequest("/api/anything", false))

	cfg := AuthGateConfig{Mode: AuthGateShadow}
	_, status, body := runGate(cfg, gateRequest("/api/anything", false), http.HandlerFunc(handler))

	if status != baseline.Code {
		t.Fatalf("status altered: gate=%d baseline=%d", status, baseline.Code)
	}
	if body != baseline.Body.String() {
		t.Fatalf("body altered: gate=%q baseline=%q", body, baseline.Body.String())
	}
}

func TestAuthGate_Enforce_BlocksUnauthenticatedNonAllowlisted(t *testing.T) {
	cfg := AuthGateConfig{Mode: AuthGateEnforce}
	called, status, _ := runGate(cfg, gateRequest("/api/anything", false), okHandler())
	if called || status != http.StatusUnauthorized {
		t.Fatalf("enforce mode must reject with 401, got called=%v status=%d", called, status)
	}
}

func TestAuthGate_Enforce_AllowsAuthenticated(t *testing.T) {
	cfg := AuthGateConfig{Mode: AuthGateEnforce}
	called, status, _ := runGate(cfg, gateRequest("/api/anything", true), okHandler())
	if !called || status != http.StatusOK {
		t.Fatalf("enforce mode must pass authenticated callers through, got called=%v status=%d", called, status)
	}
}

func TestAuthGate_Enforce_AllowlistedBypassesAuthRequirement(t *testing.T) {
	cfg := AuthGateConfig{Mode: AuthGateEnforce, Allowlist: []string{"/health"}}
	called, status, _ := runGate(cfg, gateRequest("/health", false), okHandler())
	if !called || status != http.StatusOK {
		t.Fatalf("allowlisted route must pass through even unauthenticated, got called=%v status=%d", called, status)
	}
}

func TestAuthGate_Enforce_PrefixAllowlist(t *testing.T) {
	cfg := AuthGateConfig{Mode: AuthGateEnforce, Allowlist: []string{"/api/public/*"}}
	called, _, _ := runGate(cfg, gateRequest("/api/public/dax/functions", false), okHandler())
	if !called {
		t.Fatal("prefix-allowlisted route must pass through unauthenticated")
	}
	called2, status2, _ := runGate(cfg, gateRequest("/api/private/dax/functions", false), okHandler())
	if called2 || status2 != http.StatusUnauthorized {
		t.Fatal("a route not matching the prefix must still be gated")
	}
}

func TestAuthGateModeFromEnv_UnsetDefaultsToOff(t *testing.T) {
	t.Setenv("AUTH_GATE_MODE", "")
	if got := AuthGateModeFromEnv(); got != AuthGateOff {
		t.Fatalf("got %q, want off", got)
	}
}

func TestAuthGateModeFromEnv_Shadow(t *testing.T) {
	t.Setenv("AUTH_GATE_MODE", "shadow")
	if got := AuthGateModeFromEnv(); got != AuthGateShadow {
		t.Fatalf("got %q, want shadow", got)
	}
}

func TestAuthGateModeFromEnv_Enforce(t *testing.T) {
	t.Setenv("AUTH_GATE_MODE", "enforce")
	if got := AuthGateModeFromEnv(); got != AuthGateEnforce {
		t.Fatalf("got %q, want enforce", got)
	}
}

// This is the direct regression test for the fail-open flag: a typo in
// AUTH_GATE_MODE must be impossible to mistake for an intentional "off" -
// see the doc comment on AuthGateModeFromEnv for why silent fallback here
// is the same "control that exists on paper" failure this sweep has
// caught repeatedly elsewhere.
func TestAuthGateModeFromEnv_UnrecognizedValuePanics(t *testing.T) {
	t.Setenv("AUTH_GATE_MODE", "enforc") // typo
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected a panic for an unrecognized AUTH_GATE_MODE value, got none")
		}
	}()
	AuthGateModeFromEnv()
}
