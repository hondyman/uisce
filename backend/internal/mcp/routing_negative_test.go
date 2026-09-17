package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/internal/agentic"
	"github.com/hondyman/uisce/backend/internal/middleware"
	"github.com/hondyman/uisce/backend/internal/services"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// TestStreamableCall_DoesNotCreatePath6Ticket asserts a tools/call to the
// streamable /api/mcp endpoint is not misrouted into Path 6 maker-checker.
func TestStreamableCall_DoesNotCreatePath6Ticket(t *testing.T) {
	const secret = "routing-negative-secret"
	sm := services.NewSecurityManager(nil, nil, []byte(secret))
	token, err := sm.MintDevToken(services.DevTokenInput{
		UserID:    "routing-neg",
		TenantIDs: []string{testTenantID},
		Roles:     []string{"portfolio_manager"},
	})
	if err != nil {
		t.Fatal(err)
	}

	path6 := agentic.NewMCPToolRouter(nil)
	mux := http.NewServeMux()
	mux.Handle("/api/mcp", middleware.AuthContextMiddleware(sm)(NewServer(nil).HTTPHandler()))
	mux.Handle("/api/agentic/proposals", middleware.AuthContextMiddleware(sm)(http.HandlerFunc(path6.HandleToolCall)))
	mux.Handle("/api/mcp/tools/call", middleware.AuthContextMiddleware(sm)(http.HandlerFunc(path6.HandleToolCall)))
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	// Snapshot Path 6 tickets via authenticated list... Path 6 list needs AuthInfo;
	// instead call Path 6 once later only if streamable incorrectly queued.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	c, err := client.NewStreamableHttpClient(ts.URL+"/api/mcp", transport.WithHTTPHeaders(map[string]string{
		"Authorization": "Bearer " + token,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	initReq := mcplib.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcplib.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcplib.Implementation{Name: "routing-neg", Version: "1.0.0"}
	if _, err := c.Initialize(ctx, initReq); err != nil {
		t.Fatal(err)
	}
	refuse := mcplib.CallToolRequest{}
	refuse.Params.Name = "run_sql"
	refuse.Params.Arguments = map[string]any{"sql": "select 1"}
	res, callErr := c.CallTool(ctx, refuse)
	text := ""
	if res != nil {
		for _, content := range res.Content {
			if tc, ok := content.(mcplib.TextContent); ok {
				text += tc.Text
			}
		}
	}
	combined := text
	if callErr != nil {
		combined += callErr.Error()
	}
	if strings.Contains(combined, "QUEUED_FOR_MAKER_CHECKER") || strings.Contains(combined, "ticket_id") {
		t.Fatalf("streamable tools/call looks like Path 6 proposal: %s", combined)
	}

	// Direct POST to Path 6 still works (control).
	body := []byte(`{"jsonrpc":"2.0","id":"p6","method":"tools/call","params":{"name":"draft_business_object","arguments":{"bo_name":"x"}}}`)
	req, _ := http.NewRequest("POST", ts.URL+"/api/agentic/proposals", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	ts.Config.Handler.ServeHTTP(rec, req)
	var rpc map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &rpc)
	result, _ := rpc["result"].(map[string]interface{})
	if result == nil || result["ticket_id"] == nil {
		t.Fatalf("control Path 6 call failed: %s", rec.Body.String())
	}
}
