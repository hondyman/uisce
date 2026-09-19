package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestGovernance_SearchCatalog_RequiresQuery(t *testing.T) {
	s := NewServer(nil)
	_, err := s.CallTool(context.Background(), uuid.MustParse(testTenantID), "search_catalog", json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "query") {
		t.Fatalf("expected query required, got %v", err)
	}
}

func TestGovernance_SearchCatalog_Offline(t *testing.T) {
	s := NewServer(nil)
	got, err := s.CallTool(context.Background(), uuid.MustParse(testTenantID), "search_catalog", json.RawMessage(`{"query":"order"}`))
	if err != nil {
		t.Fatal(err)
	}
	m := got.(map[string]interface{})
	if m["matches"] == nil {
		t.Fatalf("%#v", m)
	}
}

func TestGovernance_DraftBO_RequiresFields(t *testing.T) {
	s := NewServer(nil)
	_, err := s.CallTool(context.Background(), uuid.MustParse(testTenantID), "draft_business_object", json.RawMessage(`{"bo_name":"X"}`))
	if err == nil || !strings.Contains(err.Error(), "justification") {
		t.Fatalf("expected justification required, got %v", err)
	}
}

func TestGovernance_DraftBO_NoAuth(t *testing.T) {
	s := NewServer(nil)
	_, err := s.CallTool(context.Background(), uuid.Nil, "draft_business_object", json.RawMessage(`{"bo_name":"X","justification":"y"}`))
	if err == nil || !strings.Contains(err.Error(), "auth required") {
		t.Fatalf("expected auth required, got %v", err)
	}
}

func TestGovernance_DraftBO_QueuesProposal(t *testing.T) {
	s := NewServer(nil)
	got, err := s.CallTool(context.Background(), uuid.MustParse(testTenantID), "draft_business_object", json.RawMessage(`{"bo_name":"FX Options","justification":"user request","diff_payload":{"x":1}}`))
	if err != nil {
		t.Fatal(err)
	}
	m := got.(map[string]interface{})
	if m["ticket_id"] == nil || m["ticket_id"] == "" {
		t.Fatalf("%#v", m)
	}
	if m["status"] != "QUEUED_FOR_MAKER_CHECKER_APPROVAL" {
		t.Fatalf("%#v", m)
	}
}

func TestGovernance_CatalogCount(t *testing.T) {
	s := NewServer(nil)
	if n := len(s.ListTools()); n < 16 {
		t.Fatalf("expected >=16 tools after governance register, got %d", n)
	}
}
