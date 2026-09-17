package agentic

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hondyman/uisce/backend/internal/middleware"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/services"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

const testTenantID = "99e99e99-99e9-49e9-89e9-99e99e99e999"

func toolCallRequest(t *testing.T, tenantIDs ...string) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	body := []byte(`{
		"jsonrpc": "2.0",
		"id": "1",
		"method": "tools/call",
		"params": {"name": "draft_business_object", "arguments": {"bo_name": "x"}}
	}`)
	r := httptest.NewRequest("POST", "/api/mcp/tools/call", bytes.NewReader(body))
	if len(tenantIDs) > 0 {
		r = r.WithContext(security.WithAuthInfo(context.Background(), security.AuthInfo{
			UserID:    "test-user",
			TenantIDs: tenantIDs,
		}))
	}
	return r, httptest.NewRecorder()
}

func decodeToolCall(t *testing.T, rec *httptest.ResponseRecorder) MCPToolCallResponse {
	t.Helper()
	var resp MCPToolCallResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rec.Body.String())
	}
	return resp
}

func TestHandleToolCall_NoAuth_DoesNotFallBackToCore(t *testing.T) {
	router := NewMCPToolRouter(nil)
	r, rec := toolCallRequest(t)
	router.HandleToolCall(rec, r)

	resp := decodeToolCall(t, rec)
	if resp.Error == nil {
		t.Fatalf("expected auth error, got result %#v", resp.Result)
	}
	if resp.Error.Code != -32001 {
		t.Fatalf("expected -32001, got %d (%s)", resp.Error.Code, resp.Error.Message)
	}
	if resp.Result != nil {
		if tid, _ := resp.Result["ticket_id"].(string); tid != "" {
			t.Fatalf("must not queue a ticket without auth, got ticket_id=%s", tid)
		}
	}
}

func TestHandleToolCall_EmptyTenantIDs_DoesNotFallBackToCore(t *testing.T) {
	router := NewMCPToolRouter(nil)
	r, rec := toolCallRequest(t, "")
	router.HandleToolCall(rec, r)

	resp := decodeToolCall(t, rec)
	if resp.Error == nil {
		t.Fatal("expected auth error for empty tenant_id")
	}
	if resp.Error.Code != -32001 {
		t.Fatalf("expected -32001, got %d (%s)", resp.Error.Code, resp.Error.Message)
	}
}

func TestHandleToolCall_WithAuth_UsesAuthTenant(t *testing.T) {
	router := NewMCPToolRouter(nil)
	r, rec := toolCallRequest(t, testTenantID)
	router.HandleToolCall(rec, r)

	resp := decodeToolCall(t, rec)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if resp.Result["ticket_id"] == nil || resp.Result["ticket_id"] == "" {
		t.Fatalf("expected ticket_id, got %#v", resp.Result)
	}
	if got := resp.Result["status"]; got != "QUEUED_FOR_MAKER_CHECKER_APPROVAL" {
		t.Fatalf("status=%v", got)
	}
	gotTenant, _ := resp.Result["tenant_id"].(string)
	if gotTenant != testTenantID {
		t.Fatalf("proposal tenant_id=%q, want JWT/AuthInfo tenant %q (must never be core)", gotTenant, testTenantID)
	}
	if gotTenant == "core" {
		t.Fatal("proposal landed as tenant core")
	}
}

func TestHandleToolCall_JwtMiddlewareClaimsAlone_Rejected(t *testing.T) {
	// AuthContextMiddleware populates security.AuthInfo, not jwtmiddleware
	// ClaimsContextKey. The pre-fix Path 6 read GetClaimsFromContext, which
	// is always nil on this mux, so every live call used tenant "core".
	router := NewMCPToolRouter(nil)
	r, rec := toolCallRequest(t)
	claims := &jwtmiddleware.JWTClaims{UserID: "u", TenantID: testTenantID}
	r = r.WithContext(context.WithValue(r.Context(), jwtmiddleware.ClaimsContextKey, claims))
	router.HandleToolCall(rec, r)

	resp := decodeToolCall(t, rec)
	if resp.Error == nil {
		t.Fatalf("jwtmiddleware claims alone must not authorize; got result %#v", resp.Result)
	}
	if resp.Error.Code != -32001 {
		t.Fatalf("expected -32001, got %d (%s)", resp.Error.Code, resp.Error.Message)
	}
	if resp.Result != nil {
		if tid, _ := resp.Result["tenant_id"].(string); tid == "core" || tid == testTenantID {
			t.Fatalf("must not create a proposal from jwtmiddleware claims alone, tenant=%q", tid)
		}
	}
}

func TestListTicketsHandler_NoAuth_DoesNotFallBackToCore(t *testing.T) {
	svc := NewMakerCheckerService(nil)
	r := httptest.NewRequest("GET", "/api/agentic/tickets", nil)
	rec := httptest.NewRecorder()
	svc.ListTicketsHandler(rec, r)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestListTicketsHandler_WithAuth_UsesAuthTenant(t *testing.T) {
	svc := NewMakerCheckerService(nil)
	r := httptest.NewRequest("GET", "/api/agentic/tickets", nil)
	r = r.WithContext(security.WithAuthInfo(context.Background(), security.AuthInfo{
		UserID:    "test-user",
		TenantIDs: []string{testTenantID},
	}))
	rec := httptest.NewRecorder()
	svc.ListTicketsHandler(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Tickets []ApprovalTicket `json:"tickets"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Tickets) == 0 {
		t.Fatal("expected mock tickets")
	}
	if payload.Tickets[0].TenantID != testTenantID {
		t.Fatalf("ticket tenant=%q, want auth tenant %q (must not be core)", payload.Tickets[0].TenantID, testTenantID)
	}
}

func TestHandleToolCall_AuthContextMiddleware_ProposalCarriesJWTTenant(t *testing.T) {
	// Same middleware the API process mounts (api.go:862). It injects
	// security.AuthInfo and does NOT set jwtmiddleware.ClaimsContextKey.
	sm := services.NewSecurityManager(nil, nil, []byte("test-jwt-secret-for-path6-discriminator"))
	token, err := sm.MintDevToken(services.DevTokenInput{
		UserID:    "test-user",
		Email:     "test@example.com",
		TenantIDs: []string{testTenantID},
		Roles:     []string{"portfolio_manager"},
	})
	if err != nil {
		t.Fatalf("MintDevToken: %v", err)
	}

	router := NewMCPToolRouter(nil)
	mux := http.NewServeMux()
	mux.Handle("/mcp/tools/call", middleware.AuthContextMiddleware(sm)(http.HandlerFunc(router.HandleToolCall)))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	body := `{"jsonrpc":"2.0","id":"mw-1","method":"tools/call","params":{"name":"draft_business_object","arguments":{"bo_name":"x"}}}`
	req, err := http.NewRequest("POST", srv.URL+"/mcp/tools/call", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var rpc MCPToolCallResponse
	if err := json.Unmarshal(raw, &rpc); err != nil {
		t.Fatalf("decode: %v body=%s", err, raw)
	}
	if rpc.Error != nil {
		t.Fatalf("unexpected RPC error: %+v", rpc.Error)
	}
	got, _ := rpc.Result["tenant_id"].(string)
	if got != testTenantID {
		t.Fatalf("proposal tenant_id=%q want %q (AuthInfo path). If this is %q the jwtmiddleware claims key is still in play or the core fallback survived", got, testTenantID, "core")
	}
}

func TestSubmitAgentProposal_EmptyTenant_Rejected(t *testing.T) {
	svc := NewMakerCheckerService(nil)
	_, err := svc.SubmitAgentProposal(context.Background(), ProposalRequest{
		AgentID:    "test",
		TargetBOID: "customers",
		ActionType: "x",
		Payload:    json.RawMessage(`{"ok":true}`),
	})
	if err == nil {
		t.Fatal("expected error for empty tenant_id")
	}
}
