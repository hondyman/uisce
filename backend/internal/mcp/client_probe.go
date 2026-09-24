package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// SessionMode documents the HTTP transport session model.
// Stateless: WithStateLess(true) — no Mcp-Session-Id; restarts do not
// invalidate clients; no server-initiated SSE pushes on GET.
const SessionMode = "stateless"

// ProbeResult is the canonical initialize-capable live check for the
// streamable surface (same mcp-go client the stdio proxy uses).
type ProbeResult struct {
	ToolCount   int
	CallText    string
	RefuseText  string
	RefuseIsErr bool
}

// ProbeStreamable runs initialize → tools/list → tools/call → refused call
// against a streamable HTTP MCP endpoint. bearerToken may be empty for
// public tools/list-only checks (call/refuse will then fail auth).
func ProbeStreamable(ctx context.Context, endpoint, bearerToken string) (*ProbeResult, error) {
	var cancel context.CancelFunc
	if ctx == nil {
		ctx, cancel = context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
	}
	opts := []transport.StreamableHTTPCOption{}
	if bearerToken != "" {
		opts = append(opts, transport.WithHTTPHeaders(map[string]string{
			"Authorization": "Bearer " + bearerToken,
		}))
	}
	c, err := client.NewStreamableHttpClient(endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("client: %w", err)
	}
	if err := c.Start(ctx); err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}
	defer c.Close()

	initReq := mcplib.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcplib.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcplib.Implementation{Name: "uisce-mcp-live-probe", Version: "1.0.0"}
	if _, err := c.Initialize(ctx, initReq); err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
	}

	tools, err := c.ListTools(ctx, mcplib.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("tools/list: %w", err)
	}
	out := &ProbeResult{ToolCount: len(tools.Tools)}

	call := mcplib.CallToolRequest{}
	call.Params.Name = "list_pages"
	call.Params.Arguments = map[string]any{}
	res, err := c.CallTool(ctx, call)
	if err != nil {
		return out, fmt.Errorf("tools/call list_pages: %w", err)
	}
	out.CallText = toolResultText(res)

	refuse := mcplib.CallToolRequest{}
	refuse.Params.Name = "run_sql"
	refuse.Params.Arguments = map[string]any{"sql": "select 1"}
	refuseRes, refuseErr := c.CallTool(ctx, refuse)
	if refuseErr != nil {
		out.RefuseText = refuseErr.Error()
		out.RefuseIsErr = true
	} else {
		out.RefuseText = toolResultText(refuseRes)
		out.RefuseIsErr = refuseRes != nil && refuseRes.IsError
	}
	return out, nil
}

func toolResultText(res *mcplib.CallToolResult) string {
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
