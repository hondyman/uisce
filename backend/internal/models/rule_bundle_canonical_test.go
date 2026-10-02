package models

import (
	"encoding/json"
	"testing"
)

func sampleBundle() *RuleBundle {
	return &RuleBundle{
		BundleVersion: "1.0",
		TenantScope:   "core",
		Origin:        "core",
		Rules: []PortableRuleSpec{
			{
				RuleKey: "qty-limit", Name: "Single Order Qty Limit", BOName: "order",
				Domain: "compliance", Severity: "BLOCK", Timing: "pre_write",
				GovernanceStatus: "published",
				RuleAST:          json.RawMessage(`{"type":"condition","id":"c1","field":"TargetQuantity","operator":"greater_than","value":150000}`),
			},
			{
				RuleKey: "price-limit", Name: "Limit Price Sanity", BOName: "order",
				Domain: "validation", Severity: "WARN", Timing: "pre_write",
				GovernanceStatus: "published",
				RuleAST:          json.RawMessage(`{ "value" : 105.50 , "field" : "LimitPrice" , "type" : "condition" , "operator" : "greater_than" }`),
			},
		},
	}
}

// TestChecksumOrderAndWhitespaceInsensitive: same rules, different array order
// and JSON whitespace/key order must yield identical checksums.
func TestChecksumOrderAndWhitespaceInsensitive(t *testing.T) {
	a := sampleBundle()
	if err := a.ComputeChecksum(); err != nil {
		t.Fatalf("ComputeChecksum a: %v", err)
	}

	// Reverse rule order and reformat AST whitespace.
	b := sampleBundle()
	b.Rules[0], b.Rules[1] = b.Rules[1], b.Rules[0]
	if err := b.ComputeChecksum(); err != nil {
		t.Fatalf("ComputeChecksum b: %v", err)
	}
	if a.Checksum != b.Checksum {
		t.Fatalf("checksum not order/whitespace insensitive:\n %s\n %s", a.Checksum, b.Checksum)
	}
}

// TestChecksumTombstoneChanges: marking a rule Deleted must change the checksum.
func TestChecksumTombstoneChanges(t *testing.T) {
	a := sampleBundle()
	if err := a.ComputeChecksum(); err != nil {
		t.Fatal(err)
	}
	b := sampleBundle()
	b.Rules[1].Deleted = true
	b.Rules[1].RuleAST = nil
	if err := b.ComputeChecksum(); err != nil {
		t.Fatal(err)
	}
	if a.Checksum == b.Checksum {
		t.Fatal("tombstone did not change checksum")
	}
}

// TestChecksumFloatNormalization: 105.50 and 105.5 ASTs must checksum identically.
func TestChecksumFloatNormalization(t *testing.T) {
	a := sampleBundle()
	if err := a.ComputeChecksum(); err != nil {
		t.Fatal(err)
	}
	b := sampleBundle()
	b.Rules[1].RuleAST = json.RawMessage(`{"type":"condition","field":"LimitPrice","operator":"greater_than","value":105.5}`)
	if err := b.ComputeChecksum(); err != nil {
		t.Fatal(err)
	}
	if a.Checksum != b.Checksum {
		t.Fatalf("float normalization failed: %s vs %s", a.Checksum, b.Checksum)
	}
}

// TestVerifyChecksumDetectsTampering: mutating a severity after checksumming must fail verification.
func TestVerifyChecksumDetectsTampering(t *testing.T) {
	b := sampleBundle()
	if err := b.ComputeChecksum(); err != nil {
		t.Fatal(err)
	}
	b.Rules[0].Severity = "WARN" // tamper
	if err := b.VerifyChecksum(); err == nil {
		t.Fatal("tampered bundle passed checksum verification")
	}
}
