// Command stdio-live-receipt exercises cmd/mcp-server against a live API:
// mint → stdio proxy → initialize/list/call/refuse, and asserts the token
// never appears on the proxy's stderr.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hondyman/uisce/backend/internal/services"
	"github.com/mark3labs/mcp-go/client"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run() error {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return fmt.Errorf("JWT_SECRET required")
	}
	proxyBin := os.Getenv("UISCE_MCP_SERVER_BIN")
	if proxyBin == "" {
		proxyBin = "/tmp/uisce-mcp-server"
	}
	apiURL := os.Getenv("UISCE_API_URL")
	if apiURL == "" {
		apiURL = "http://127.0.0.1:8080"
	}
	tenant := "99e99e99-99e9-49e9-89e9-99e99e99e999"

	sm := services.NewSecurityManager(nil, nil, []byte(secret))
	token, err := sm.MintDevToken(services.DevTokenInput{
		UserID:    "stdio-live",
		TenantIDs: []string{tenant},
		Roles:     []string{"portfolio_manager"},
	})
	if err != nil {
		return fmt.Errorf("mint: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	c, err := client.NewStdioMCPClient(proxyBin, []string{
		"UISCE_API_URL=" + apiURL,
		"UISCE_API_TOKEN=" + token,
	})
	if err != nil {
		return fmt.Errorf("stdio client: %w", err)
	}
	defer c.Close()

	if stderr, ok := client.GetStderr(c); ok && stderr != nil {
		go io.Copy(io.Discard, stderr) // drained; leak checked via side spawn below
	}

	initReq := mcplib.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcplib.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcplib.Implementation{Name: "stdio-live-receipt", Version: "1.0.0"}
	if _, err := c.Initialize(ctx, initReq); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	tools, err := c.ListTools(ctx, mcplib.ListToolsRequest{})
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}
	call := mcplib.CallToolRequest{}
	call.Params.Name = "list_pages"
	call.Params.Arguments = map[string]any{}
	res, err := c.CallTool(ctx, call)
	if err != nil {
		return fmt.Errorf("call: %w", err)
	}
	text := toolText(res)

	refuse := mcplib.CallToolRequest{}
	refuse.Params.Name = "run_sql"
	refuse.Params.Arguments = map[string]any{"sql": "select 1"}
	refuseRes, refuseErr := c.CallTool(ctx, refuse)
	refuseOK := refuseErr != nil || (refuseRes != nil && refuseRes.IsError) || strings.Contains(toolText(refuseRes), "refused") || strings.Contains(toolText(refuseRes), "unknown")

	// Client-invariant: spawn proxy alone and capture stderr during boot/init window.
	cmd := exec.Command(proxyBin)
	cmd.Env = append(os.Environ(), "UISCE_API_URL="+apiURL, "UISCE_API_TOKEN="+token)
	var errBuf strings.Builder
	cmd.Stderr = &errBuf
	cmd.Stdout = io.Discard
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("stderr spawn: %w", err)
	}
	time.Sleep(1200 * time.Millisecond)
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
	tokenInStderr := strings.Contains(errBuf.String(), token)

	out := map[string]any{
		"tools":           len(tools.Tools),
		"tenant_in_call":  strings.Contains(text, tenant),
		"refuse_ok":       refuseOK,
		"token_in_stderr": tokenInStderr,
		"stderr_bytes":    errBuf.Len(),
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(out)

	if len(tools.Tools) < 7 || !strings.Contains(text, tenant) || !refuseOK || tokenInStderr {
		return fmt.Errorf("stdio live receipt failed: %+v", out)
	}
	return nil
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
