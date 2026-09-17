package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
)

func TestServer_ListTools_HasPath1AndPath2(t *testing.T) {
	s := NewServer(nil)
	tools := s.ListTools()
	if len(tools) < 10 {
		t.Fatalf("expected >=10 tools (7 Path 1 + 3 Path 2), got %d", len(tools))
	}
	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}
	for _, want := range []string{
		"get_business_object_contract",
		"resolve_relationship_path",
		"list_business_objects",
		"get_bo_terms",
		"list_pages",
		"get_page",
		"compile_semantic_query",
		"text_to_semantic_ast",
		"triage_mdm_exception",
		"inspect_schema_drift",
	} {
		if !names[want] {
			t.Errorf("missing tool %s", want)
		}
	}
	for _, refused := range []string{"save_record", "run_sql", "delete_record"} {
		if names[refused] {
			t.Errorf("refused tool %s must not appear in ListTools", refused)
		}
	}
}

func TestServer_CallTool_RequiresTenant(t *testing.T) {
	s := NewServer(nil)
	_, err := s.CallTool(context.Background(), uuid.Nil, "list_business_objects", json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "auth required") {
		t.Fatalf("expected auth required, got %v", err)
	}
}

func TestServer_CallTool_Unknown(t *testing.T) {
	s := NewServer(nil)
	_, err := s.CallTool(context.Background(), uuid.MustParse(testTenantID), "no_such_tool", json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "unknown tool") {
		t.Fatalf("expected unknown tool, got %v", err)
	}
}

func TestServer_CallTool_Refused(t *testing.T) {
	s := NewServer(nil)
	_, err := s.CallTool(context.Background(), uuid.MustParse(testTenantID), "run_sql", json.RawMessage(`{"sql":"select 1"}`))
	if err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("expected refused, got %v", err)
	}
}

func TestServer_CallTool_ListBusinessObjects(t *testing.T) {
	s := NewServer(nil)
	result, err := s.CallTool(context.Background(), uuid.MustParse(testTenantID), "list_business_objects", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("list_business_objects: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("result type %T", result)
	}
	if _, ok := m["business_objects"]; !ok {
		t.Fatalf("missing business_objects: %#v", m)
	}
}

func TestServer_CallTool_GetContract(t *testing.T) {
	s := NewServer(nil)
	result, err := s.CallTool(context.Background(), uuid.MustParse(testTenantID), "get_business_object_contract", json.RawMessage(`{"bo_key":"customer_profile"}`))
	if err != nil {
		t.Fatalf("get_business_object_contract: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("result type %T", result)
	}
	if m["tenant_id"] == nil {
		t.Fatalf("expected tenant_id in contract: %#v", m)
	}
}

func TestServer_CallTool_TextToAST(t *testing.T) {
	s := NewServer(nil)
	result, err := s.CallTool(context.Background(), uuid.MustParse(testTenantID), "text_to_semantic_ast", json.RawMessage(`{"prompt":"show accounts"}`))
	if err != nil {
		t.Fatalf("text_to_semantic_ast: %v", err)
	}
	if result == nil {
		t.Fatal("expected AST result")
	}
}

func TestTenantFromAuth(t *testing.T) {
	_, err := tenantFromAuth(context.Background())
	if err == nil {
		t.Fatal("expected error without auth")
	}
	ctx := security.WithAuthInfo(context.Background(), security.AuthInfo{
		UserID:    "u",
		TenantIDs: []string{testTenantID},
	})
	id, err := tenantFromAuth(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if id.String() != testTenantID {
		t.Fatalf("got %s", id)
	}
}

func TestServer_RegisterTool_AddsToList(t *testing.T) {
	s := NewServer(nil)
	before := len(s.ListTools())
	s.RegisterTool("test_extra", "extra", map[string]interface{}{"type": "object"}, func(context.Context, uuid.UUID, json.RawMessage) (interface{}, error) {
		return map[string]string{"ok": "yes"}, nil
	})
	if len(s.ListTools()) != before+1 {
		t.Fatalf("expected %d tools, got %d", before+1, len(s.ListTools()))
	}
	got, err := s.CallTool(context.Background(), uuid.MustParse(testTenantID), "test_extra", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.(map[string]string)["ok"] != "yes" {
		t.Fatalf("%#v", got)
	}
}
