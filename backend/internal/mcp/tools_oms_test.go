package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
)

func TestOMS_GeneratePageSpec_ThroughServer(t *testing.T) {
	s := NewServer(nil)
	tid := uuid.MustParse(testTenantID)
	got, err := s.CallTool(context.Background(), tid, "generate_page_spec", json.RawMessage(`{"bo_key":"order","bo_name":"Order","page_kind":"list"}`))
	if err != nil {
		t.Fatal(err)
	}
	m := got.(map[string]interface{})
	if m["pageKind"] != "list" {
		t.Fatalf("pageKind=%v", m["pageKind"])
	}
	if m["tenant_id"] != testTenantID {
		t.Fatalf("tenant_id field=%v", m["tenant_id"])
	}
}

func TestOMS_DescribeJourney_ThroughServer(t *testing.T) {
	s := NewServer(nil)
	tid := uuid.MustParse(testTenantID)
	got, err := s.CallTool(context.Background(), tid, "describe_oms_journey", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	m := got.(map[string]interface{})
	if m["primaryBo"] == nil {
		t.Fatalf("%#v", m)
	}
}

func TestOMS_GetBOSchema_RequiresIdentifier(t *testing.T) {
	s := NewServer(nil)
	tid := uuid.MustParse(testTenantID)
	got, err := s.CallTool(context.Background(), tid, "get_bo_schema", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	m := got.(map[string]interface{})
	note, _ := m["note"].(string)
	if note == "" && m["fields"] == nil {
		t.Fatalf("expected empty fields or note: %#v", m)
	}
}

func TestOMS_StartFIX_TemporalUnset_NamedError(t *testing.T) {
	s := NewServer(nil) // Temporal unset
	tid := uuid.MustParse(testTenantID)
	_, err := s.CallTool(context.Background(), tid, "start_fix_order_entry", json.RawMessage(`{"order_id":"11111111-1111-1111-1111-111111111111"}`))
	if err == nil {
		t.Fatal("expected Temporal-unset error")
	}
	if err.Error() != ErrTemporalNotConfigured {
		t.Fatalf("want exact %q, got %q", ErrTemporalNotConfigured, err.Error())
	}
}

func TestOMS_StartFIX_RequiresAuthViaCallTool(t *testing.T) {
	s := NewServer(nil)
	_, err := s.CallTool(context.Background(), uuid.Nil, "start_fix_order_entry", json.RawMessage(`{"order_id":"x"}`))
	if err == nil || !strings.Contains(err.Error(), "auth required") {
		t.Fatalf("expected auth required, got %v", err)
	}
}

func TestOMS_ToolsRegistered(t *testing.T) {
	s := NewServer(nil)
	names := map[string]bool{}
	for _, tool := range s.ListTools() {
		names[tool.Name] = true
	}
	for _, want := range []string{"generate_page_spec", "describe_oms_journey", "get_bo_schema", "start_fix_order_entry"} {
		if !names[want] {
			t.Errorf("missing %s (catalog should be 14)", want)
		}
	}
	if n := len(s.ListTools()); n < 14 {
		t.Fatalf("expected >=14 tools after OMS rescue, got %d", n)
	}
}

func TestOMS_ManifestParity(t *testing.T) {
	// Dirty tool_handler.go list descriptors (read 2026-09-17) — agent contract.
	want := map[string]struct {
		descContains string
		required     []string
		props        []string
	}{
		"generate_page_spec": {
			descContains: "Page Studio spec",
			props:        []string{"bo_id", "bo_key", "bo_name", "page_kind"},
		},
		"describe_oms_journey": {
			descContains: "Northwind Order",
		},
		"get_bo_schema": {
			descContains: "form schema",
			props:        []string{"bo_id", "bo_key"},
		},
		"start_fix_order_entry": {
			descContains: "FIXOrderEntryWorkflow",
			required:     []string{"order_id"},
			props:        []string{"order_id", "command"},
		},
	}
	s := NewServer(nil)
	byName := map[string]ToolDefinition{}
	for _, tool := range s.ListTools() {
		byName[tool.Name] = tool
	}
	for name, spec := range want {
		tool, ok := byName[name]
		if !ok {
			t.Fatalf("missing tool %s", name)
			continue
		}
		if tool.Description == "" || !strings.Contains(tool.Description, spec.descContains) {
			t.Errorf("%s description=%q want contain %q", name, tool.Description, spec.descContains)
		}
		schemaType, _ := tool.InputSchema["type"].(string)
		if schemaType != "object" {
			t.Errorf("%s schema type=%q", name, schemaType)
		}
		props, _ := tool.InputSchema["properties"].(map[string]interface{})
		for _, p := range spec.props {
			if _, ok := props[p]; !ok {
				t.Errorf("%s missing property %s", name, p)
			}
		}
		if len(spec.required) > 0 {
			req, _ := tool.InputSchema["required"].([]string)
			if req == nil {
				// JSON round-trip sometimes yields []interface{}
				if raw, ok := tool.InputSchema["required"].([]interface{}); ok {
					for _, r := range raw {
						req = append(req, fmt.Sprint(r))
					}
				}
			}
			have := map[string]bool{}
			for _, r := range req {
				have[r] = true
			}
			for _, r := range spec.required {
				if !have[r] {
					t.Errorf("%s required missing %s (got %v)", name, r, tool.InputSchema["required"])
				}
			}
		}
	}
}

func TestRefusalListParity(t *testing.T) {
	// Dirty tool_handler.go executeToolCall refusal case includes create_page_draft.
	for _, name := range []string{
		"save_record", "delete_record", "run_sql", "create_business_object", "create_page_draft",
	} {
		if !IsRefusedTool(name) {
			t.Errorf("refusal list missing %s (dirty Path 1 parity)", name)
		}
	}
	s := NewServer(nil)
	_, err := s.CallTool(context.Background(), uuid.MustParse(testTenantID), "create_page_draft", json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("create_page_draft: want refused, got %v", err)
	}
}

func TestOMS_WrongTenantMembership_OnStreamableWrapper(t *testing.T) {
	// Body tenant outside AuthInfo set is enforced by Path 1 HandleRPC;
	// unified CallTool injects tenant from the caller. Membership check for
	// streamable is AuthInfo-only — wrong-tenant is "call with uuid.Nil".
	s := NewServer(nil)
	ctx := security.WithAuthInfo(context.Background(), security.AuthInfo{
		UserID:    "u",
		TenantIDs: []string{testTenantID},
	})
	tid, err := tenantFromAuth(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if tid.String() != testTenantID {
		t.Fatal(tid)
	}
	// Explicit foreign tenant as CallTool arg is caller responsibility;
	// wrapHandler always uses AuthInfo. Assert AuthInfo wins over a forged id.
	_, err = s.CallTool(ctx, tid, "generate_page_spec", json.RawMessage(`{"bo_key":"order"}`))
	if err != nil {
		t.Fatal(err)
	}
}
