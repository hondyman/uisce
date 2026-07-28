package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hondyman/uisce/backend/internal/mcp"
	"github.com/stretchr/testify/assert"
)

func TestMCP_CannotBypassMakerChecker(t *testing.T) {
	// Setup the Copilot with a valid token
	copilot := mcp.NewUisceCopilot("http://mock-api", "valid-user-token")

	// Simulate an AI trying to hallucinate a "force_deploy" tool that doesn't exist
	maliciousReq := mcp.JSONRPCRequest{
		ID:     "1",
		Method: "tools/call",
		Params: json.RawMessage(`{"name": "force_deploy", "arguments": {}}`),
	}

	resp := copilot.HandleRequest(context.Background(), maliciousReq)

	// Proof: The MCP server statically rejects unauthorized tool calls
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "Unknown tool", resp.Error.Message)
}

func TestMCP_ProperlyFormatsDraftPayload(t *testing.T) {
	copilot := mcp.NewUisceCopilot("http://mock-api", "valid-user-token")

	// Simulate Claude using the correct draft tool
	validReq := mcp.JSONRPCRequest{
		ID:     "2",
		Method: "tools/call",
		Params: json.RawMessage(`{
			"name": "draft_business_object", 
			"arguments": {
				"bo_name": "FX Options",
				"bo_key": "fx_options",
				"justification": "Requested by user for real-time margin.",
				"diff_payload": {"bindings": [{"tier": "STREAMING", "backend": "FLINK"}]}
			}
		}`),
	}

	resp := copilot.HandleRequest(context.Background(), validReq)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)
}
