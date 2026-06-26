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
	req.Header.Set("X-Tenant-ID", "tenant-123")

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body: %s", w.Code, w.Body.String())
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
