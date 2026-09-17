package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hondyman/uisce/backend/internal/middleware"
	"github.com/hondyman/uisce/backend/internal/services"
)

const integrationJWTSecret = "test-jwt-secret-for-integration-tests-only"

func newTestSecurityManager() *services.SecurityManager {
	return services.NewSecurityManager(nil, nil, []byte(integrationJWTSecret))
}

func mintIntegrationToken(t *testing.T, sm *services.SecurityManager, claims jwt.MapClaims) string {
	t.Helper()
	token, err := sm.SignToken(claims)
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	return token
}

func newIntegrationServer(t *testing.T) (*httptest.Server, *services.SecurityManager) {
	t.Helper()
	sm := newTestSecurityManager()
	handler := NewMCPToolHandler(nil)
	mux := http.NewServeMux()
	mux.Handle("/mcp", middleware.AuthContextMiddleware(sm)(http.HandlerFunc(handler.HandleRPC)))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, sm
}

func postMCP(url, token, body string) (int, []byte) {
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(body))
	if err != nil {
		return 0, nil
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	return resp.StatusCode, buf.Bytes()
}

func decodeMCP(t *testing.T, raw []byte) JSONRPCResponse {
	t.Helper()
	var rpc JSONRPCResponse
	if err := json.Unmarshal(raw, &rpc); err != nil {
		t.Fatalf("decode JSON-RPC: %v, body: %s", err, raw)
	}
	return rpc
}

func TestMCP_Integration_RealSecurityManager_AuthRoundtrip(t *testing.T) {
	srv, sm := newIntegrationServer(t)
	token := mintIntegrationToken(t, sm, jwt.MapClaims{
		"user_id":    "test-user",
		"email":      "test@example.com",
		"tenant_id":  testTenantID,
		"tenant_ids": []string{testTenantID},
		"roles":      []string{"portfolio_manager"},
		"iat":        time.Now().Unix(),
		"exp":        time.Now().Add(time.Hour).Unix(),
	})
	body := `{"jsonrpc":"2.0","id":"it-1","method":"tools/call","params":{"name":"list_pages","arguments":{}}}`
	status, raw := postMCP(srv.URL+"/mcp", token, body)
	if status != http.StatusOK {
		t.Fatalf("status: %d, body: %s", status, raw)
	}
	rpc := decodeMCP(t, raw)
	if rpc.Error != nil {
		t.Fatalf("unexpected RPC error: %v", rpc.Error)
	}
}

func TestMCP_Integration_TenantMismatch_RealToken(t *testing.T) {
	srv, sm := newIntegrationServer(t)
	token := mintIntegrationToken(t, sm, jwt.MapClaims{
		"user_id":    "test-user",
		"tenant_id":  testTenantID,
		"tenant_ids": []string{testTenantID},
		"roles":      []string{"portfolio_manager"},
		"iat":        time.Now().Unix(),
		"exp":        time.Now().Add(time.Hour).Unix(),
	})
	body := `{"jsonrpc":"2.0","id":"it-2","method":"tools/call","params":{"name":"list_pages","arguments":{"tenant_id":"` + nonMemberTenantID + `"}}}`
	status, raw := postMCP(srv.URL+"/mcp", token, body)
	if status != http.StatusOK {
		t.Fatalf("status: %d, body: %s", status, raw)
	}
	rpc := decodeMCP(t, raw)
	if rpc.Error == nil || rpc.Error.Code != InvalidParams {
		t.Fatalf("expected -32602, got %v", rpc.Error)
	}
}

func TestMCP_Integration_NoTenantClaim(t *testing.T) {
	srv, sm := newIntegrationServer(t)
	token := mintIntegrationToken(t, sm, jwt.MapClaims{
		"user_id": "test-user",
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	body := `{"jsonrpc":"2.0","id":"it-3","method":"tools/call","params":{"name":"list_pages","arguments":{}}}`
	status, raw := postMCP(srv.URL+"/mcp", token, body)
	if status != http.StatusOK {
		t.Fatalf("status: %d, body: %s", status, raw)
	}
	rpc := decodeMCP(t, raw)
	if rpc.Error == nil || rpc.Error.Code != Unauthorized {
		t.Fatalf("expected -32001, got %v", rpc.Error)
	}
}

func TestMCP_Integration_GarbageToken(t *testing.T) {
	srv, _ := newIntegrationServer(t)
	body := `{"jsonrpc":"2.0","id":"it-4","method":"tools/call","params":{"name":"list_pages","arguments":{}}}`
	status, raw := postMCP(srv.URL+"/mcp", "not-a-real-jwt", body)
	if status != http.StatusOK {
		t.Fatalf("status: %d, body: %s", status, raw)
	}
	rpc := decodeMCP(t, raw)
	if rpc.Error == nil || rpc.Error.Code != Unauthorized {
		t.Fatalf("expected -32001, got %v", rpc.Error)
	}
	expected := "auth required: JWT missing tenant_id claim (account may lack a tenant assignment)"
	if rpc.Error.Message != expected {
		t.Fatalf("expected message %q, got %q", expected, rpc.Error.Message)
	}
}
