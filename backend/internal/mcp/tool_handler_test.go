package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCP_ToolsList(t *testing.T) {
	handler := NewMCPToolHandler(nil)

	reqBody := []byte(`{
		"jsonrpc": "2.0",
		"id": "1",
		"method": "tools/list"
	}`)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewBuffer(reqBody))
	rec := httptest.NewRecorder()

	handler.HandleRPC(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var resp JSONRPCResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed decoding JSON-RPC response: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %v", resp.Error)
	}

	resMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", resp.Result)
	}

	tools, ok := resMap["tools"].([]interface{})
	if !ok || len(tools) < 2 {
		t.Fatalf("expected at least 2 tools, got %v", tools)
	}
}

func TestMCP_GetContractCall(t *testing.T) {
	handler := NewMCPToolHandler(nil)

	reqBody := []byte(`{
		"jsonrpc": "2.0",
		"id": "2",
		"method": "tools/call",
		"params": {
			"name": "get_business_object_contract",
			"arguments": {
				"tenant_id": "99e99e99-99e9-49e9-89e9-99e99e99e999",
				"bo_key": "customer_profile"
			}
		}
	}`)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewBuffer(reqBody))
	rec := httptest.NewRecorder()

	handler.HandleRPC(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var resp JSONRPCResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed decoding response: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %v", resp.Error)
	}
}
