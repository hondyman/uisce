// Command mcp-server is the Cursor/Claude stdio MCP proxy for Uisce.
//
// It speaks MCP over stdio to the IDE and forwards tools to the live API
// via mark3labs streamable HTTP (UISCE_API_URL + UISCE_API_TOKEN).
// Credentials: only UISCE_API_TOKEN from the environment. Stdout is the
// protocol channel — never log the token (stderr only for errors, redacted).
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(0)

	token := strings.TrimSpace(os.Getenv("UISCE_API_TOKEN"))
	baseURL := strings.TrimSpace(os.Getenv("UISCE_API_URL"))
	if token == "" {
		log.Fatal("FATAL: UISCE_API_TOKEN required")
	}
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/api/mcp"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	remote, err := client.NewStreamableHttpClient(endpoint, transport.WithHTTPHeaders(map[string]string{
		"Authorization": "Bearer " + token,
	}))
	if err != nil {
		log.Fatalf("streamable client: %v", err)
	}
	if err := remote.Start(ctx); err != nil {
		log.Fatalf("start client: %v", err)
	}
	defer remote.Close()

	initReq := mcplib.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcplib.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcplib.Implementation{Name: "uisce-stdio-proxy", Version: "1.0.0"}
	if _, err := remote.Initialize(ctx, initReq); err != nil {
		log.Fatalf("initialize: %v", err)
	}

	toolsRes, err := remote.ListTools(ctx, mcplib.ListToolsRequest{})
	if err != nil {
		log.Fatalf("list tools: %v", err)
	}

	local := server.NewMCPServer("uisce-stdio-proxy", "1.0.0")
	for _, tool := range toolsRes.Tools {
		t := tool
		local.AddTool(t, func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			req.Params.Name = t.Name
			return remote.CallTool(ctx, req)
		})
	}

	if err := server.ServeStdio(local); err != nil {
		fmt.Fprintf(os.Stderr, "stdio server: %v\n", err)
		os.Exit(1)
	}
}
