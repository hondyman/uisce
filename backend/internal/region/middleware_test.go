package region

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
)

// asTenant gives a request the verified active tenant AuthContextMiddleware
// would have established. The region gate takes its tenant from there only.
func asTenant(req *http.Request, tenantID string) *http.Request {
	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID: "user-1", TenantIDs: []string{tenantID}, ActiveTenantID: tenantID,
	})
	return req.WithContext(ctx)
}

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
	req = asTenant(req, "tenant-123")
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
	req = asTenant(req, "tenant-123")
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

	req := asTenant(httptest.NewRequest("GET", "/test", nil), "tenant-123")
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

// The ABAC capability map is authorization metadata about the caller's own
// scope, not tenant data. Requiring a region here 400'd every non-gold-copy
// tenant before the handler ran, which made the whole menu-authorization
// feature look dead. Regression guard: the exemption must not be removed.
func TestRegionMiddleware_ExemptsCapabilities_NoRegionRequired(t *testing.T) {
	r := chi.NewRouter()
	// A provider that would reject any region, so a pass-through proves the
	// exemption short-circuited before region validation ran.
	r.Use(RegionValidationMiddleware(&mockRegionsProvider{allowed: []string{"eu-west"}}))
	r.Get("/api/capabilities", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"menu:platform":false}`))
	}))

	req := httptest.NewRequest("GET", "/api/capabilities", nil)
	req = asTenant(req, "tenant-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for /api/capabilities with no region header, got %d (body: %s)",
			resp.StatusCode, w.Body.String())
	}
}

// An even a disallowed region must not block the capability map: it is the
// caller's own scope, so there is no cross-region data to guard.
func TestRegionMiddleware_ExemptsCapabilities_DisallowedRegionIgnored(t *testing.T) {
	r := chi.NewRouter()
	r.Use(RegionValidationMiddleware(&mockRegionsProvider{allowed: []string{"us-east"}}))
	r.Get("/api/capabilities", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"menu:platform":true}`))
	}))

	req := httptest.NewRequest("GET", "/api/capabilities", nil)
	req.Header.Set(RegionHeader, "eu-west")
	req = asTenant(req, "tenant-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for /api/capabilities with a disallowed region, got %d (body: %s)",
			resp.StatusCode, w.Body.String())
	}
}

func newGateRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(RegionValidationMiddleware(&mockRegionsProvider{allowed: []string{"us-east"}}))
	r.Get("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	return r
}

// No verified tenant means the request is refused, not waved through.
func TestRegionMiddleware_RejectsRequestWithNoVerifiedTenant(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(RegionHeader, "us-east")
	w := httptest.NewRecorder()
	newGateRouter().ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a verified tenant, got %d", w.Code)
	}
}

// A client must not be able to skip the region requirement by asserting the
// gold-copy tenant in the URL, query string or a header.
func TestRegionMiddleware_ClientAssertedGoldCopyTenantDoesNotBypassRegion(t *testing.T) {
	gold := resolveGoldCopyTenantID().String()
	for name, mutate := range map[string]func(*http.Request){
		"query":  func(r *http.Request) { r.URL.RawQuery = "tenant_id=" + gold },
		"header": func(r *http.Request) { r.Header.Set("X-Tenant-ID", gold) },
	} {
		req := httptest.NewRequest("GET", "/test", nil)
		mutate(req)
		req = asTenant(req, uuid.NewString()) // verified: an ordinary tenant, no region sent
		w := httptest.NewRecorder()
		newGateRouter().ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: a client-asserted gold-copy tenant must not skip the region requirement; got %d", name, w.Code)
		}
	}
}
