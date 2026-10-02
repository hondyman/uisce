package analytics

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hondyman/uisce/backend/internal/models"
)

// uuidPattern matches any UUID v1-v8 shape, upper or lower case.
var uuidPattern = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

func sampleProperties() json.RawMessage {
	return json.RawMessage(`{
		"bo_name": "order",
		"tenant_id": "99e99e99-99e9-49e9-89e9-99e99e99e999",
		"severity": "BLOCK",
		"timing": "pre_write",
		"domain": "compliance",
		"governance_status": "published",
		"rule_key": "single-order-qty-limit",
		"binding_ids": ["a2154183-1234-4abc-9def-111111111111"]
	}`)
}

// TestPortableSpecNoUUIDLeak is the core portability guarantee: a spec derived
// from a realistic node (UUID id, tenant UUID in properties, binding UUIDs)
// must contain zero UUID-shaped strings anywhere in its serialized form.
func TestPortableSpecNoUUIDLeak(t *testing.T) {
	nodeID := uuid.MustParse("a2154183-beef-4abc-9def-abcdefabcdef")
	spec, ruleKey, err := portableSpecFromNode(
		nodeID, "Single Order Qty Limit", "Blocks oversized orders",
		sampleProperties(),
		json.RawMessage(`{"rule_ast":{"type":"condition","id":"c1","field":"TargetQuantity","operator":"greater_than","value":100000}}`),
	)
	if err != nil {
		t.Fatalf("portableSpecFromNode: %v", err)
	}
	if ruleKey != "single-order-qty-limit" {
		t.Fatalf("rule_key = %q, want properties rule_key", ruleKey)
	}

	serialized, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if matches := uuidPattern.FindAllString(string(serialized), -1); len(matches) > 0 {
		t.Fatalf("portable spec leaks UUIDs: %v\nserialized: %s", matches, serialized)
	}
}

// TestPortableSpecRuleKeyFallback: legacy rows without properties.rule_key
// must fall back to node_name.
func TestPortableSpecRuleKeyFallback(t *testing.T) {
	props := json.RawMessage(`{"bo_name":"order","severity":"WARN","timing":"pre_write"}`)
	spec, ruleKey, err := portableSpecFromNode(
		uuid.New(), "Legacy Rule Name", "", props,
		json.RawMessage(`{"rule_ast":{"type":"condition","id":"c1","field":"Side","operator":"equals","value":"BUY"}}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if ruleKey != "Legacy Rule Name" || spec.RuleKey != "Legacy Rule Name" {
		t.Fatalf("expected node_name fallback, got %q", ruleKey)
	}
}

// TestPortableSpecASTCanonicalized: exported AST bytes must be canonical
// (whitespace/key-order normalized), regardless of how the rule was stored.
func TestPortableSpecASTCanonicalized(t *testing.T) {
	messy := json.RawMessage(`{"rule_ast":{ "operator" : "greater_than" , "value" : 105.50 , "field" : "LimitPrice" , "type" : "condition" , "id" : "c1" }}`)
	clean := json.RawMessage(`{"rule_ast":{"type":"condition","id":"c1","field":"LimitPrice","operator":"greater_than","value":105.5}}`)

	specA, _, err := portableSpecFromNode(uuid.New(), "r", "", sampleProperties(), messy)
	if err != nil {
		t.Fatal(err)
	}
	specB, _, err := portableSpecFromNode(uuid.New(), "r", "", sampleProperties(), clean)
	if err != nil {
		t.Fatal(err)
	}
	if string(specA.RuleAST) != string(specB.RuleAST) {
		t.Fatalf("AST not canonicalized on export:\n messy=%s\n clean=%s", specA.RuleAST, specB.RuleAST)
	}
}

// TestPortableSpecMissingASTFailsLoud: a rule without rule_ast must fail
// export, not silently produce an empty spec.
func TestPortableSpecMissingASTFailsLoud(t *testing.T) {
	if _, _, err := portableSpecFromNode(uuid.New(), "r", "", sampleProperties(), json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected error for rule with no rule_ast, got nil")
	}
	if _, _, err := portableSpecFromNode(uuid.New(), "r", "", sampleProperties(), nil); err == nil {
		t.Fatal("expected error for rule with nil config, got nil")
	}
}

// TestExportBundleNoUUIDLeak: full bundle serialization (the actual wire
// format, including the computed checksum) must contain no UUID-shaped strings.
func TestExportBundleNoUUIDLeak(t *testing.T) {
	spec, _, err := portableSpecFromNode(
		uuid.New(), "Single Order Qty Limit", "",
		sampleProperties(),
		json.RawMessage(`{"rule_ast":{"type":"condition","id":"c1","field":"TargetQuantity","operator":"greater_than","value":100000}}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	bundle := &models.RuleBundle{
		BundleVersion: models.RuleBundleVersion,
		TenantScope:   models.ValidationRuleOriginCore,
		Origin:        models.ValidationRuleOriginCore,
		Rules:         []models.PortableRuleSpec{*spec},
	}
	if err := bundle.ComputeChecksum(); err != nil {
		t.Fatal(err)
	}
	serialized, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	s := string(serialized)
	if matches := uuidPattern.FindAllString(s, -1); len(matches) > 0 {
		t.Fatalf("bundle leaks UUIDs: %v\n%s", matches, s)
	}
	if strings.Contains(s, "binding_ids") {
		t.Fatal("bundle must not export physical binding_ids")
	}
}
