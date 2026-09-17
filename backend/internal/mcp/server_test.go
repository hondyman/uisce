package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// TestServer_Skeleton_BuildsAndStubsRespond verifies the Step 1 skeleton
// is wired, not just compiled. Constructs Server, calls ListTools, calls
// CallTool with a sample invocation, asserts no panic and the stub
// returns sensible zero values.
func TestServer_Skeleton_BuildsAndStubsRespond(t *testing.T) {
	// nil DB is fine for Step 1 — the stubs don't touch it.
	s := NewServer(nil)
	if s == nil {
		t.Fatal("NewServer(nil) returned nil")
	}

	// Stub ListTools returns empty list.
	tools := s.ListTools()
	if tools == nil {
		t.Fatal("ListTools returned nil, expected empty slice")
	}
	if len(tools) != 0 {
		t.Fatalf("Step 1 stub should return empty tool list, got %d", len(tools))
	}

	// Stub CallTool returns nil, nil (no-op for Step 1).
	result, err := s.CallTool(context.Background(), uuid.New(), "unknown_tool", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Step 1 stub CallTool should not error, got: %v", err)
	}
	if result != nil {
		t.Fatalf("Step 1 stub CallTool should return nil result, got: %v", result)
	}

	// RegisterTool is a no-op in Step 1; calling it must not panic.
	s.RegisterTool("test", "test desc", map[string]string{"type": "object"}, nil)
}
