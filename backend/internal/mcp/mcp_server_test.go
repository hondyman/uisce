package mcp_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/mcp"
)

func TestMCPServer_ListTools(t *testing.T) {
	server := mcp.NewMCPServer(nil)
	tools := server.ListTools()

	if len(tools) != 5 {
		t.Fatalf("expected 5 registered MCP tools, got: %d", len(tools))
	}
}

func TestMCPServer_TextToAST_Execution(t *testing.T) {
	server := mcp.NewMCPServer(nil)
	tenantID := uuid.New()

	req := mcp.ToolExecutionRequest{
		TenantID:   tenantID,
		ToolName:   "text_to_semantic_ast",
		Actor:      "UNIT_TEST",
		Parameters: map[string]interface{}{"prompt": "Show me security price and industry sector for Apple"},
	}

	resp, err := server.ExecuteTool(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error executing MCP tool: %v", err)
	}

	if resp.ToolName != "text_to_semantic_ast" {
		t.Errorf("tool name mismatch: %s", resp.ToolName)
	}

	// Verify Rule 7 Security Guard on nil tenant
	req.TenantID = uuid.Nil
	_, nilTenantErr := server.ExecuteTool(context.Background(), req)
	if nilTenantErr == nil {
		t.Fatalf("expected Rule 7 violation error on nil tenant_id")
	}
}
