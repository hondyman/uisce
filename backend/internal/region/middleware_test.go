package region

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// mock provider for tests
type mockRegionsProvider struct {
	allowed []string
}

func (m *mockRegionsProvider) GetAllowedRegions(tenantID string) ([]string, error) {
	return m.allowed, nil
}

func TestRegionValidation_AllowsRequest_WhenHeaderPresent_AndAllowed(t *testing.T) {
	r := chi.NewRouter()
	prov := &mockRegionsProvider{allowed: []string{"eu-west"}}
	r.Use(RegionValidationMiddleware(prov))
	r.Get("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		region, ok := GetRegionFromContext(r.Context())
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("missing region in context"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"region": region})
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(RegionHeader, "eu-west")
	req.Header.Set("X-Tenant-ID", "tenant-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["region"] != "eu-west" {
		t.Fatalf("expected region 'eu-west', got '%s'", body["region"])
	}
}

func TestRegionValidation_BlocksRequest_WhenRegionNotAllowed(t *testing.T) {
	r := chi.NewRouter()
	prov := &mockRegionsProvider{allowed: []string{"us-east"}}
	r.Use(RegionValidationMiddleware(prov))
	r.Get("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(RegionHeader, "eu-west")
	req.Header.Set("X-Tenant-ID", "tenant-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["error"] != "region 'eu-west' is not configured for tenant 'tenant-123'" {
		t.Fatalf("unexpected error message: %v", body["error"])
	}
}

func TestRegionMiddleware_BlocksRequest_WhenHeaderMissing(t *testing.T) {
	r := chi.NewRouter()
	r.Use(RegionValidationMiddleware(nil))
	r.Get("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["error"] != "region is required for all semantic operations." {
		t.Fatalf("unexpected error message: %v", body["error"])
	}
}
