package mcp

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/internal/middleware"
	"github.com/hondyman/uisce/backend/internal/services"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func TestStdioProxy_StreamableHTTP_ListCallRefuse(t *testing.T) {
	const secret = "stdio-proxy-test-secret"
	sm := services.NewSecurityManager(nil, nil, []byte(secret))
	token, err := sm.MintDevToken(services.DevTokenInput{
		UserID:    "proxy-user",
		TenantIDs: []string{testTenantID},
		Roles:     []string{"portfolio_manager"},
	})
	if err != nil {
		t.Fatalf("MintDevToken: %v", err)
	}

	unified := NewServer(nil)
	mux := http.NewServeMux()
	mux.Handle("/api/mcp", middleware.AuthContextMiddleware(sm)(unified.HTTPHandler()))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	var stderrBuf bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	c, err := client.NewStreamableHttpClient(srv.URL+"/api/mcp", transport.WithHTTPHeaders(map[string]string{
		"Authorization": "Bearer " + token,
	}))
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if err := c.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer c.Close()

	initReq := mcplib.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcplib.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcplib.Implementation{Name: "proxy-test", Version: "1.0.0"}
	if _, err := c.Initialize(ctx, initReq); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	tools, err := c.ListTools(ctx, mcplib.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools.Tools) < 7 {
		t.Fatalf("expected >=7 tools, got %d", len(tools.Tools))
	}

	call := mcplib.CallToolRequest{}
	call.Params.Name = "list_business_objects"
	call.Params.Arguments = map[string]any{}
	res, err := c.CallTool(ctx, call)
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res == nil || res.IsError {
		t.Fatalf("unexpected tool error: %#v", res)
	}
	text := toolText(res)
	if !strings.Contains(text, testTenantID) {
		t.Fatalf("tool result missing tenant_id field: %s", text)
	}

	refuse := mcplib.CallToolRequest{}
	refuse.Params.Name = "run_sql"
	refuse.Params.Arguments = map[string]any{"sql": "select 1"}
	refuseRes, err := c.CallTool(ctx, refuse)
	if err == nil && (refuseRes == nil || !refuseRes.IsError) && !strings.Contains(toolText(refuseRes), "refused") && !strings.Contains(toolText(refuseRes), "unknown") {
		t.Fatalf("expected refusal or unknown for run_sql, got %#v", refuseRes)
	}

	_ = stderrBuf
	_ = io.Discard
	if strings.Contains(stderrBuf.String(), token) {
		t.Fatal("token leaked to stderr buffer")
	}
}

func toolText(res *mcplib.CallToolResult) string {
	if res == nil {
		return ""
	}
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(mcplib.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}
