package region_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/region"
)

type mockProv struct{ allowed []string }

func (m *mockProv) GetAllowedRegions(tenantID string) ([]string, error) { return m.allowed, nil }

func TestRegionValidation_AllowsRequest_WhenAllowed(t *testing.T) {
	r := chi.NewRouter()
	prov := &mockProv{allowed: []string{"eu-west"}}
	r.Use(region.RegionValidationMiddleware(prov))
	r.Get("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reg, ok := region.GetRegionFromContext(r.Context())
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("missing region in context"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"region": reg})
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(region.RegionHeader, "eu-west")
	req.Header.Set("X-Tenant-ID", "tenant-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestRegionValidation_Blocks_WhenNotAllowed(t *testing.T) {
	r := chi.NewRouter()
	prov := &mockProv{allowed: []string{"us-east"}}
	r.Use(region.RegionValidationMiddleware(prov))
	r.Get("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(region.RegionHeader, "eu-west")
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
	if body["error"] == "" {
		t.Fatalf("expected an error message, got empty")
	}
}
