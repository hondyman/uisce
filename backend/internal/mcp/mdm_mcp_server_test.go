package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestMDMMCPServer(t *testing.T) {
	server := NewMDMMCPServer(nil)
	ctx := context.Background()
	tenantID := uuid.New()
	exceptionID := uuid.New()
	goldenID := uuid.New()

	// 1. Test triage_mdm_exception
	triageParams, _ := json.Marshal(map[string]interface{}{
		"exception_id": exceptionID,
	})

	resp, err := server.ExecuteTool(ctx, MCPToolRequest{
		ToolName:   "triage_mdm_exception",
		TenantID:   tenantID,
		Parameters: triageParams,
	})
	if err != nil {
		t.Fatalf("unexpected error executing triage tool: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success=true, got error: %s", resp.ErrorMsg)
	}

	if resp.Data["winning_vendor"] != "BLOOMBERG" {
		t.Errorf("expected BLOOMBERG winner, got %v", resp.Data["winning_vendor"])
	}

	if len(resp.MerkleReceipt) != 64 {
		t.Errorf("expected 64-char SHA256 merkle receipt, got %d", len(resp.MerkleReceipt))
	}

	// 2. Test explain_survivorship_winner
	explainParams, _ := json.Marshal(map[string]interface{}{
		"golden_id": goldenID,
		"attribute": "market_price",
	})

	resp, err = server.ExecuteTool(ctx, MCPToolRequest{
		ToolName:   "explain_survivorship_winner",
		TenantID:   tenantID,
		Parameters: explainParams,
	})
	if err != nil {
		t.Fatalf("unexpected error executing explain tool: %v", err)
	}

	if resp.Data["winning_source"] != "BLOOMBERG" {
		t.Errorf("expected BLOOMBERG winning source, got %v", resp.Data["winning_source"])
	}

	// 3. Test resolve_cross_reference_break
	resolveParams, _ := json.Marshal(map[string]interface{}{
		"orphan_identifier_type":  "SEDOL",
		"orphan_identifier_value": "2046251",
		"target_golden_id":        goldenID,
	})

	resp, err = server.ExecuteTool(ctx, MCPToolRequest{
		ToolName:   "resolve_cross_reference_break",
		TenantID:   tenantID,
		Parameters: resolveParams,
	})
	if err != nil {
		t.Fatalf("unexpected error executing resolve tool: %v", err)
	}

	if resp.Data["edge_type"] != "IS_PEER_IDENTIFIER_OF" {
		t.Errorf("expected IS_PEER_IDENTIFIER_OF edge type, got %v", resp.Data["edge_type"])
	}
}
