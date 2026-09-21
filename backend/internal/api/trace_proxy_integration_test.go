package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestProxyTempoTracesWithValidAPIKey(t *testing.T) {
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer mockBackend.Close()

	t.Setenv("TRACE_QUERY_URL", mockBackend.URL)

	// Ensure traceAuthConfig has a test key
	traceAuthConfig = DefaultTraceAuthConfig()
	traceAuthConfig.APIKeys["test-key"] = []string{"admin"}

	server := &Server{}
	router := chi.NewRouter()
	router.Get("/traces", server.proxyTempoTraces)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/traces?plan_id=plan-123", nil)
	req.Header.Set("X-API-Key", "test-key")
	// With an API key the tenant is taken from a VALIDATED token, never from a header.
	req.Header.Set("Authorization", "Bearer "+createTestToken("tenant-123", "user-1"))

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body: %s", w.Code, w.Body.String())
	}
}

// A valid API key plus an X-Tenant-ID header is NOT enough: the header is client-controlled, so
// honouring it would let any key holder read another tenant's traces.
func TestProxyTempoTraces_TenantHeaderAloneIsRejected(t *testing.T) {
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("the upstream trace backend must not be reached without a validated tenant")
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()
	t.Setenv("TRACE_QUERY_URL", mockBackend.URL)

	traceAuthConfig = DefaultTraceAuthConfig()
	traceAuthConfig.APIKeys["test-key"] = []string{"admin"}

	server := &Server{}
	router := chi.NewRouter()
	router.Get("/traces", server.proxyTempoTraces)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/traces?plan_id=plan-123", nil)
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("X-Tenant-ID", "tenant-123") // header only, no validated token

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (no validated tenant), got %d body: %s", w.Code, w.Body.String())
	}
}

func TestProxyTempoTracesWithoutAuth(t *testing.T) {
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer mockBackend.Close()

	t.Setenv("TRACE_QUERY_URL", mockBackend.URL)

	traceAuthConfig = DefaultTraceAuthConfig()

	server := &Server{}
	router := chi.NewRouter()
	router.Get("/traces", server.proxyTempoTraces)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/traces?plan_id=plan-123", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body: %s", w.Code, w.Body.String())
	}
	var errResp TraceAuthErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("invalid error response: %v", err)
	}
	if errResp.Error != "Authorization or validation error" {
		t.Fatalf("expected unauthorized error got %s", errResp.Error)
	}
}
