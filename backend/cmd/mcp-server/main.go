package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/hondyman/uisce/backend/internal/mcp"
)

// MCP StdIO Protocol requires reading/writing JSON-RPC over stdin/stdout
func main() {
	// The User's API key is passed via the Cursor/Claude MCP configuration
	apiToken := os.Getenv("UISCE_API_TOKEN")
	targetURL := os.Getenv("UISCE_API_URL") // e.g., http://localhost:8080

	if apiToken == "" {
		log.Fatalf("FATAL: UISCE_API_TOKEN environment variable required for ABAC enforcement")
	}

	copilot := mcp.NewUisceCopilot(targetURL, apiToken)
	scanner := bufio.NewScanner(os.Stdin)

	// Listen for MCP Tool Execution Requests from Claude/Cursor
	for scanner.Scan() {
		rawLine := scanner.Bytes()
		var req mcp.JSONRPCRequest
		if err := json.Unmarshal(rawLine, &req); err != nil {
			mcp.SendError(req.ID, mcp.ParseError, "Invalid JSON-RPC payload")
			continue
		}

		// Route the request to the secure tool handlers
		resp := copilot.HandleRequest(context.Background(), req)

		// Send response back to the LLM
		out, _ := json.Marshal(resp)
		fmt.Println(string(out))
		os.Stdout.Sync()
	}
}
