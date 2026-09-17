package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hondyman/uisce/backend/internal/security"
)

const (
	handlersTestTenantID = "99e99e99-99e9-49e9-89e9-99e99e99e999"
)

func handlersContextWithAuth(tenantIDs ...string) context.Context {
	if len(tenantIDs) == 0 {
		return context.Background()
	}
	return security.WithAuthInfo(context.Background(), security.AuthInfo{
		UserID:    "test-user",
		TenantIDs: tenantIDs,
	})
}

func handlersRequest(method, path string, body []byte, tenantIDs ...string) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if len(tenantIDs) > 0 {
		r = r.WithContext(handlersContextWithAuth(tenantIDs...))
	}
	return r
}

// TestMCPHandler_HandleMCPRequest_RequiresAuth verifies the gate fires
// when no JWT-derived AuthInfo is in context. Closes the impersonation gap
// where handleCallTool hardcoded tenantID := "default".
func TestMCPHandler_HandleMCPRequest_RequiresAuth(t *testing.T) {
	h := &MCPHandler{} // GraphService: nil — gate should fire before any service call

	tests := []struct {
		name   string
		method string
		body   string
	}{
		{"mcp.list_tools", "mcp.list_tools", `{"jsonrpc":"2.0","id":"1","method":"mcp.list_tools"}`},
		{"mcp.call_tool", "mcp.call_tool", `{"jsonrpc":"2.0","id":"1","method":"mcp.call_tool","params":{"name":"get_node_schema","arguments":{"node_name":"foo"}}}`},
		{"mcp.list_resources", "mcp.list_resources", `{"jsonrpc":"2.0","id":"1","method":"mcp.list_resources"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := handlersRequest("POST", "/mcp", []byte(tc.body)) // no auth
			rec := httptest.NewRecorder()
			h.HandleMCPRequest(rec, r)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected HTTP 200 (gate returns envelope, not raw error), got %d", rec.Code)
			}
			var resp JSONRPCResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if resp.Error == nil {
				t.Fatalf("expected error envelope, got success")
			}
			if resp.Error.Code != -32001 {
				t.Fatalf("expected code -32001 (auth required), got %d", resp.Error.Code)
			}
		})
	}
}

// TestMCPHandler_HandleMCPRequest_RejectsUnknownMethod verifies the gate
// also fires before the method switch (auth gate runs unconditionally first).
func TestMCPHandler_HandleMCPRequest_RejectsUnknownMethod(t *testing.T) {
	h := &MCPHandler{}
	r := handlersRequest("POST", "/mcp", []byte(`{"jsonrpc":"2.0","id":"1","method":"unknown.method"}`))
	rec := httptest.NewRecorder()
	h.HandleMCPRequest(rec, r)
	// Without auth, the gate fires first and returns -32001, not -32601 (method not found).
	var resp JSONRPCResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Error == nil || resp.Error.Code != -32001 {
		t.Fatalf("expected -32001 (auth required) before method dispatch, got %v", resp.Error)
	}
}

// TestMCPHandler_DispatchTenant_NoAuth verifies the dispatch helper.
func TestMCPHandler_DispatchTenant_NoAuth(t *testing.T) {
	h := &MCPHandler{}
	r := httptest.NewRequest("POST", "/mcp", nil)
	rec := httptest.NewRecorder()
	tenant, ok := h.dispatchTenant(rec, r, "test-id")
	if ok {
		t.Fatalf("expected ok=false without auth")
	}
	if tenant.String() != "00000000-0000-0000-0000-000000000000" {
		t.Fatalf("expected zero UUID, got %v", tenant)
	}
	if rec.Body.Len() == 0 {
		t.Fatalf("expected error envelope written")
	}
}

// TestMCPHandler_DispatchTenant_ValidAuth verifies the dispatch helper
// returns the JWT-derived tenant.
func TestMCPHandler_DispatchTenant_ValidAuth(t *testing.T) {
	h := &MCPHandler{}
	r := handlersRequest("POST", "/mcp", nil, handlersTestTenantID)
	rec := httptest.NewRecorder()
	tenant, ok := h.dispatchTenant(rec, r, "test-id")
	if !ok {
		t.Fatalf("expected ok=true with valid auth")
	}
	if tenant.String() != handlersTestTenantID {
		t.Fatalf("expected tenant %s, got %s", handlersTestTenantID, tenant)
	}
}

// TestMCPToolsHandler_ListTools_RequiresAuth verifies the gate fires
// when no AuthInfo is in context. Closes the impersonation gap where
// ListTools hardcoded tenantID := "core" and trusted X-Functional-Role.
func TestMCPToolsHandler_ListTools_RequiresAuth(t *testing.T) {
	h := &MCPToolsHandler{} // registry: nil — gate should fire before registry call
	r := httptest.NewRequest("GET", "/mcp/tools", nil)
	rec := httptest.NewRecorder()
	h.ListTools(rec, r)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated ListTools, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestMCPToolsHandler_ListTools_XFunctionalRoleIgnored verifies the
// X-Functional-Role header trust is removed. Without auth, the header
// alone must NOT produce a successful response.
func TestMCPToolsHandler_ListTools_XFunctionalRoleIgnored(t *testing.T) {
	h := &MCPToolsHandler{}
	r := httptest.NewRequest("GET", "/mcp/tools", nil)
	r.Header.Set("X-Functional-Role", "admin")
	rec := httptest.NewRecorder()
	h.ListTools(rec, r)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("X-Functional-Role header should NOT bypass auth; expected 401, got %d", rec.Code)
	}
}
