package mcp_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/mcp"
)

// Path 2 tools now live on the unified Server (PR D port from mcp_server.go).

func TestCatalogTools_RegisteredOnServer(t *testing.T) {
	s := mcp.NewServer(nil)
	names := map[string]bool{}
	for _, tool := range s.ListTools() {
		names[tool.Name] = true
	}
	for _, want := range []string{"text_to_semantic_ast", "triage_mdm_exception", "inspect_schema_drift"} {
		if !names[want] {
			t.Errorf("missing %s", want)
		}
	}
}

func TestCatalogTools_TextToAST_ThroughServer(t *testing.T) {
	s := mcp.NewServer(nil)
	tid := uuid.New()
	got, err := s.CallTool(context.Background(), tid, "text_to_semantic_ast", json.RawMessage(`{"prompt":"Show me security price and industry sector for Apple"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("nil result")
	}
	_, err = s.CallTool(context.Background(), uuid.Nil, "text_to_semantic_ast", json.RawMessage(`{"prompt":"x"}`))
	if err == nil || !strings.Contains(err.Error(), "auth required") {
		t.Fatalf("expected auth required on nil tenant, got %v", err)
	}
}
