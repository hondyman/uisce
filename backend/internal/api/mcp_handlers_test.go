package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
)

const (
	mcpTestTenantID  = "99e99e99-99e9-49e9-89e9-99e99e99e999"
	mcpTestTenantID2 = "11111111-1111-1111-1111-111111111111"
)

func mcpContextWithAuth(tenantIDs ...string) context.Context {
	if len(tenantIDs) == 0 {
		return context.Background()
	}
	return security.WithAuthInfo(context.Background(), security.AuthInfo{
		UserID:    "test-user",
		TenantIDs: tenantIDs,
	})
}

func mcpRequestWithAuth(method, path string, body []byte, tenantIDs ...string) (*http.Request, *httptest.ResponseRecorder) {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if len(tenantIDs) > 0 {
		r = r.WithContext(mcpContextWithAuth(tenantIDs...))
	}
	return r, httptest.NewRecorder()
}

// TestMCPHandler_ListTools_RequiresAuth verifies that Path 4's
// ListTools returns 401 when no JWT-derived AuthInfo is in context.
// Closes the impersonation gap where ListTools had no auth at all.
func TestMCPHandler_ListTools_RequiresAuth(t *testing.T) {
	h := NewMCPHandler(nil)
	r, rec := mcpRequestWithAuth("GET", "/api/v1/mcp/tools", nil) // no auth
	h.ListTools(rec, r)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated ListTools, got %d", rec.Code)
	}
}

// TestMCPHandler_ListTools_WithAuth_ReturnsTools verifies the positive
// case: authenticated request returns the tool manifest.
func TestMCPHandler_ListTools_WithAuth_ReturnsTools(t *testing.T) {
	h := NewMCPHandler(nil)
	r, rec := mcpRequestWithAuth("GET", "/api/v1/mcp/tools", nil, mcpTestTenantID)
	h.ListTools(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := resp["tools"]; !ok {
		t.Fatalf("expected 'tools' key in response, got %v", resp)
	}
}

// TestMCPHandler_ExecuteTool_RequiresAuth verifies that ExecuteTool returns
// 401 when no AuthInfo is present. Previously trusted X-Tenant-ID header.
func TestMCPHandler_ExecuteTool_RequiresAuth(t *testing.T) {
	h := NewMCPHandler(nil)
	body := []byte(`{"name":"text_to_semantic_ast","parameters":{"prompt":"test"}}`)
	r, rec := mcpRequestWithAuth("POST", "/api/v1/mcp/tools", body) // no auth
	h.ExecuteTool(rec, r)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated ExecuteTool, got %d", rec.Code)
	}
}

// TestMCPHandler_ExecuteTool_XTenantIDHeaderIgnored verifies that the
// X-Tenant-ID header is no longer trusted. The dispatch tenant comes
// from the JWT only. We can't directly verify "default tenant" wasn't
// read because the handler still needs a real DB to actually query;
// the dispatch helper is unit-tested separately.
func TestMCPHandler_ExecuteTool_XTenantIDHeaderIgnored(t *testing.T) {
	h := NewMCPHandler(nil)
	body := []byte(`{"name":"text_to_semantic_ast","parameters":{"prompt":"test"}}`)
	r, rec := mcpRequestWithAuth("POST", "/api/v1/mcp/tools", body, mcpTestTenantID)
	// Set a malicious X-Tenant-ID header claiming a different tenant.
	r.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000000")
	h.ExecuteTool(rec, r)
	// With nil DB the handler will fail at the SQL layer (which is fine
	// for this test); what matters is that auth gate didn't reject (401).
	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("auth gate rejected valid JWT: %d", rec.Code)
	}
}

// TestMCPHandler_OmniboxSearch_RequiresAuth verifies OmniboxSearch returns
// 401 without auth. Previously read X-Tenant-ID header.
func TestMCPHandler_OmniboxSearch_RequiresAuth(t *testing.T) {
	h := NewMCPHandler(nil)
	body := []byte(`{"prompt":"test"}`)
	r, rec := mcpRequestWithAuth("POST", "/api/v1/mcp/omnibox", body) // no auth
	h.OmniboxSearch(rec, r)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated OmniboxSearch, got %d", rec.Code)
	}
}

// TestMCPHandler_OmniboxSearch_XTenantIDHeaderIgnored verifies the body
// tenant_id is validated against the JWT auth set, not trusted verbatim.
func TestMCPHandler_OmniboxSearch_XTenantIDHeaderIgnored(t *testing.T) {
	h := NewMCPHandler(nil)
	body := []byte(`{"prompt":"test","tenant_id":"00000000-0000-4000-8000-000000000000"}`)
	r, rec := mcpRequestWithAuth("POST", "/api/v1/mcp/omnibox", body, mcpTestTenantID)
	h.OmniboxSearch(rec, r)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for body tenant_id outside auth set, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestMCPHandler_OmniboxSearch_BodyTenantIDInAuthSet verifies the
// membership check accepts a body tenant_id that IS in the auth set.
func TestMCPHandler_OmniboxSearch_BodyTenantIDInAuthSet(t *testing.T) {
	h := NewMCPHandler(nil)
	body := []byte(`{"prompt":"test","tenant_id":"` + mcpTestTenantID2 + `"}`)
	r, rec := mcpRequestWithAuth("POST", "/api/v1/mcp/omnibox", body, mcpTestTenantID, mcpTestTenantID2)
	h.OmniboxSearch(rec, r)
	if rec.Code == http.StatusBadRequest {
		t.Fatalf("body tenant_id in auth set should pass; got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestMCPHandler_DispatchTenantFromAuth_UnitTest tests the gate helper directly.
func TestMCPHandler_DispatchTenantFromAuth_UnitTest(t *testing.T) {
	h := NewMCPHandler(nil)

	// No auth → 401
	r, rec := mcpRequestWithAuth("GET", "/api/v1/mcp/tools", nil)
	tenant, ok := h.dispatchTenantFromAuth(rec, r, "")
	if ok || tenant != uuid.Nil {
		t.Fatalf("expected (uuid.Nil, false) without auth, got (%v, %v)", tenant, ok)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	// Valid auth → returns first tenant
	r, rec = mcpRequestWithAuth("GET", "/api/v1/mcp/tools", nil, mcpTestTenantID)
	tenant, ok = h.dispatchTenantFromAuth(rec, r, "")
	if !ok {
		t.Fatalf("expected ok=true with valid auth")
	}
	if tenant.String() != mcpTestTenantID {
		t.Fatalf("expected tenant %s, got %s", mcpTestTenantID, tenant)
	}

	// Body tenant_id outside auth set → 400
	r, rec = mcpRequestWithAuth("GET", "/api/v1/mcp/tools", nil, mcpTestTenantID)
	_, ok = h.dispatchTenantFromAuth(rec, r, "00000000-0000-4000-8000-000000000000")
	if ok {
		t.Fatalf("expected ok=false for body tenant outside auth set")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	// Body tenant_id inside auth set → returns the matching tenant
	r, rec = mcpRequestWithAuth("GET", "/api/v1/mcp/tools", nil, mcpTestTenantID, mcpTestTenantID2)
	tenant, ok = h.dispatchTenantFromAuth(rec, r, mcpTestTenantID2)
	if !ok || tenant.String() != mcpTestTenantID2 {
		t.Fatalf("expected (%v, true) for body tenant in auth set, got (%v, %v)", mcpTestTenantID2, tenant, ok)
	}
}
