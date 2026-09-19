package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestCatalog_TextToAST_ThroughServer(t *testing.T) {
	s := NewServer(nil)
	tid := uuid.New()
	got, err := s.CallTool(context.Background(), tid, "text_to_semantic_ast", json.RawMessage(`{"prompt":"Show me security price and industry sector for Apple"}`))
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("nil result")
	}
}

func TestCatalog_TextToAST_NilTenant(t *testing.T) {
	s := NewServer(nil)
	_, err := s.CallTool(context.Background(), uuid.Nil, "text_to_semantic_ast", json.RawMessage(`{"prompt":"x"}`))
	if err == nil || !strings.Contains(err.Error(), "auth required") {
		t.Fatalf("expected auth required, got %v", err)
	}
}

func TestCatalog_Triage_InvalidUUID(t *testing.T) {
	s := NewServer(nil)
	_, err := s.CallTool(context.Background(), uuid.New(), "triage_mdm_exception", json.RawMessage(`{"exceptionId":"not-a-uuid"}`))
	if err == nil || !strings.Contains(err.Error(), "invalid exceptionId") {
		t.Fatalf("got %v", err)
	}
}

func TestCatalog_Triage_NilDB_Mock(t *testing.T) {
	s := NewServer(nil)
	got, err := s.CallTool(context.Background(), uuid.New(), "triage_mdm_exception", json.RawMessage(`{"exceptionId":"11111111-1111-1111-1111-111111111111"}`))
	if err != nil {
		t.Fatal(err)
	}
	m, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("%T %#v", got, got)
	}
	if m["diagnosis"] == nil {
		t.Fatalf("expected mock diagnosis: %#v", m)
	}
}

func TestCatalog_InspectDrift_NilDB(t *testing.T) {
	s := NewServer(nil)
	got, err := s.CallTool(context.Background(), uuid.New(), "inspect_schema_drift", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = got
}
