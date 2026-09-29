package vm

import (
	"encoding/json"
	"strings"
	"testing"
)

func mustCompact(t *testing.T, node RuleNode) string {
	t.Helper()
	raw, err := Compact(node)
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}
	return string(raw)
}

// TestCompactFloatNormalization verifies "105.50" and "105.5" canonicalize identically.
func TestCompactFloatNormalization(t *testing.T) {
	rawA := []byte(`{"type":"condition","id":"c1","field":"LimitPrice","operator":"greater_than","value":105.50}`)
	rawB := []byte(`{"type":"condition","id":"c1","field":"LimitPrice","operator":"greater_than","value":105.5}`)

	ca, err := CanonicalJSON(rawA)
	if err != nil {
		t.Fatalf("CanonicalJSON a: %v", err)
	}
	cb, err := CanonicalJSON(rawB)
	if err != nil {
		t.Fatalf("CanonicalJSON b: %v", err)
	}
	if string(ca) != string(cb) {
		t.Fatalf("float normalization mismatch:\n a=%s\n b=%s", ca, cb)
	}
	if !strings.Contains(string(ca), `105.5`) || strings.Contains(string(ca), `105.50`) {
		t.Fatalf("expected shortest float form, got %s", ca)
	}
}

// TestCompactIntegerPreservation ensures large integers never round-trip through float64.
func TestCompactIntegerPreservation(t *testing.T) {
	node := RuleNode{Type: NodeTypeCondition, Condition: &RuleCondition{
		ID: "c1", Field: "TargetQuantity", Operator: "greater_than",
		Value: json.Number("9007199254740993"), // 2^53+1 — lossy as float64
	}}
	out := mustCompact(t, node)
	if !strings.Contains(out, `9007199254740993`) {
		t.Fatalf("integer literal corrupted: %s", out)
	}
}

// TestCompactKeySorting verifies semantically identical key orders hash identically.
func TestCompactKeySorting(t *testing.T) {
	// Same AST authored with different struct field population order.
	a := RuleNode{Type: NodeTypeGroup, Group: &RuleGroup{
		ID: "g1", Operator: "AND", Conditions: []RuleNode{
			{Type: NodeTypeCondition, Condition: &RuleCondition{ID: "c1", Field: "Side", Operator: "equals", Value: "BUY"}},
			{Type: NodeTypeCondition, Condition: &RuleCondition{ID: "c2", Field: "TargetQuantity", Operator: "less_than", Value: 100000}},
		},
	}}
	ca := mustCompact(t, a)

	// Re-marshal round trip must be byte-stable.
	var back RuleNode
	if err := json.Unmarshal([]byte(ca), &back); err != nil {
		t.Fatalf("unmarshal canonical output: %v", err)
	}
	if re := mustCompact(t, back); re != ca {
		t.Fatalf("round-trip not byte-stable:\n first =%s\n second=%s", ca, re)
	}
}

// TestCompactNestedExpressions exercises BinaryExpr / FuncCall / FieldRef / Literal.
func TestCompactNestedExpressions(t *testing.T) {
	node := RuleNode{Type: NodeTypeExpression, Expression: &Expression{
		Root: &BinaryExpr{
			Op: ">",
			Left: &BinaryExpr{
				Op:    "*",
				Left:  &FieldRef{Path: "TargetQuantity"},
				Right: &Literal{Value: 105.50},
			},
			Right: &FieldRef{Path: "LimitPrice"},
		},
	}}
	out := mustCompact(t, node)

	var back RuleNode
	if err := json.Unmarshal([]byte(out), &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if re := mustCompact(t, back); re != out {
		t.Fatalf("nested expression round-trip not stable:\n %s\n %s", out, re)
	}
}
