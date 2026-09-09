package vm

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestRuleNode_MarshalUnmarshal_RoundTrip covers every RuleNode variant
// (Group, Condition, Expression with each ExprNode kind, and nesting).
// Before RuleNode.MarshalJSON/Expression.MarshalJSON/etc existed,
// encoding/json's default struct marshaling produced capitalized,
// nested-wrapper JSON ({"Expression":{"Root":{...}}}) that
// UnmarshalJSON's flat, lowercase-keyed parsing could not read back -
// Marshal always succeeded, the round trip silently lost data. Found by
// testing the actual round trip, not by inspecting the marshal output.
func TestRuleNode_MarshalUnmarshal_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		node RuleNode
	}{
		{
			name: "condition",
			node: RuleNode{
				Type: NodeTypeCondition,
				Condition: &RuleCondition{
					ID: "c1", Field: "age", Operator: "greater_than", Value: 18.0,
				},
			},
		},
		{
			name: "group with nested conditions",
			node: RuleNode{
				Type: NodeTypeGroup,
				Group: &RuleGroup{
					ID:       "g1",
					Operator: "AND",
					Conditions: []RuleNode{
						{Type: NodeTypeCondition, Condition: &RuleCondition{Field: "age", Operator: ">", Value: 18.0}},
						{Type: NodeTypeCondition, Condition: &RuleCondition{Field: "status", Operator: "equals", Value: "active"}},
					},
				},
			},
		},
		{
			name: "expression: binary arithmetic",
			node: RuleNode{
				Type: NodeTypeExpression,
				Expression: &Expression{
					Root: &BinaryExpr{
						Op:    "/",
						Left:  &FieldRef{Path: "revenue"},
						Right: &FieldRef{Path: "cogs"},
					},
				},
			},
		},
		{
			name: "expression: FuncCall with mixed FieldRef/Literal args",
			node: RuleNode{
				Type: NodeTypeExpression,
				Expression: &Expression{
					Root: &FuncCall{
						Name: "MAX_LENGTH",
						Args: []ExprNode{&FieldRef{Path: "city"}, &Literal{Value: 15}},
					},
				},
			},
		},
		{
			name: "expression: nested FuncCall inside BinaryExpr",
			node: RuleNode{
				Type: NodeTypeExpression,
				Expression: &Expression{
					Root: &BinaryExpr{
						Op:   "==",
						Left: &FuncCall{Name: "SUM", Args: []ExprNode{&FieldRef{Path: "cash_flows"}}},
						Right: &Literal{Value: 0},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.node)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var back RuleNode
			if err := json.Unmarshal(b, &back); err != nil {
				t.Fatalf("Unmarshal failed: %v\njson: %s", err, b)
			}

			// Re-marshal and compare: cheaper and more robust than a deep
			// field-by-field comparison across the ExprNode interface tree,
			// and equally conclusive - if the second marshal matches the
			// first, the round trip preserved everything Marshal expressed.
			b2, err := json.Marshal(back)
			if err != nil {
				t.Fatalf("re-Marshal failed: %v", err)
			}

			var m1, m2 map[string]any
			_ = json.Unmarshal(b, &m1)
			_ = json.Unmarshal(b2, &m2)
			if !reflect.DeepEqual(m1, m2) {
				t.Errorf("round trip mismatch\n  first:  %s\n  second: %s", b, b2)
			}
		})
	}
}

// TestExpression_StandaloneMarshal covers the calc-term convention (see
// backend/internal/analytics/pre_aggregation_service.go), which stores and
// reads a bare Expression - {"root": {...}} - not wrapped in a RuleNode.
func TestExpression_StandaloneMarshal(t *testing.T) {
	expr := Expression{
		Root: &FuncCall{Name: "NPV", Args: []ExprNode{&Literal{Value: 0.08}, &FieldRef{Path: "cash_flow"}}},
	}
	b, err := json.Marshal(expr)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var back Expression
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("Unmarshal failed: %v\njson: %s", err, b)
	}
	if back.Root == nil {
		t.Fatalf("round trip lost Root: %s", b)
	}
	fc, ok := back.Root.(*FuncCall)
	if !ok || fc.Name != "NPV" || len(fc.Args) != 2 {
		t.Errorf("round trip produced wrong shape: %+v", back.Root)
	}
}
