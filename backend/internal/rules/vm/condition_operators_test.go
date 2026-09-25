package vm

import "testing"

func cond(field, op string, value interface{}) RuleNode {
	return RuleNode{Type: NodeTypeCondition, Condition: &RuleCondition{Field: field, FieldPath: field, Operator: op, Value: value}}
}

// Semantics mirror AdvancedConditionBuilder.tsx's client-side evaluator.
func TestConditionOperators(t *testing.T) {
	data := map[string]interface{}{
		"name":    "Total Revenue",
		"empty":   "",
		"symbol":  "BAD",
		"n":       7.0,
		"i":       3,
		"zero":    0.0,
		"flag":    true,
		"sflag":   "false",
		"tags":    []interface{}{"salary", "amount"},
		"none":    []interface{}{},
		"nodes":   []interface{}{map[string]interface{}{"type": "task"}, map[string]interface{}{"type": "start"}},
		"numlist": []interface{}{1.0, 2.0},
	}
	for _, tc := range []struct {
		name  string
		node  RuleNode
		want  bool
		isErr bool
	}{
		{"contains case-insensitive", cond("name", "contains", "revenue"), true, false},
		{"not_contains", cond("name", "not_contains", "cost"), true, false},
		{"starts_with", cond("name", "starts_with", "TOTAL"), true, false},
		{"ends_with", cond("name", "ends_with", "nue"), true, false},
		{"matches_regex", cond("name", "matches_regex", `^[a-zA-Z0-9_ ]+$`), true, false},
		{"matches_regex no match", cond("symbol", "matches_regex", `^[a-z]+$`), false, false},
		{"matches_regex invalid pattern is an error", cond("name", "matches_regex", `([`), false, true},
		{"is_empty empty string", cond("empty", "is_empty", nil), true, false},
		{"is_empty empty array", cond("none", "is_empty", nil), true, false},
		{"is_empty absent field", cond("missing", "is_empty", nil), true, false},
		{"is_not_empty absent field", cond("missing", "is_not_empty", nil), false, false},
		{"is_not_empty array", cond("nodes", "is_not_empty", nil), true, false},
		{"length_less", cond("name", "length_less", 5), false, false},
		{"length_greater", cond("name", "length_greater", 4), true, false},
		{"length_equals", cond("symbol", "length_equals", 3), true, false},
		{"length non-numeric is an error", cond("symbol", "length_equals", "x"), false, true},
		{"in array", cond("symbol", "in", []interface{}{"RSTR", "BAD"}), true, false},
		{"in comma string", cond("symbol", "in", "RSTR, BAD"), true, false},
		{"not_in", cond("symbol", "not_in", []interface{}{"RSTR", "LOCKED"}), true, false},
		{"in numeric by string form", cond("n", "in", []interface{}{7.0}), true, false},
		{"in int vs float string form", cond("i", "in", "3"), true, false},
		{"contains_any", cond("tags", "contains_any", []interface{}{"ssn", "salary"}), true, false},
		{"contains_any none", cond("tags", "contains_any", []interface{}{"ssn"}), false, false},
		{"contains_any non-array", cond("symbol", "contains_any", []interface{}{"BAD"}), false, false},
		{"contains_all", cond("tags", "contains_all", "salary,amount"), true, false},
		{"contains_all missing one", cond("tags", "contains_all", []interface{}{"salary", "ssn"}), false, false},
		{"projected array field", cond("nodes.type", "contains_any", []interface{}{"start"}), true, false},
		{"is_true bool", cond("flag", "is_true", nil), true, false},
		{"is_false string", cond("sflag", "is_false", nil), true, false},
		{"is_positive", cond("n", "is_positive", nil), true, false},
		{"is_zero", cond("zero", "is_zero", nil), true, false},
		{"is_negative", cond("n", "is_negative", nil), false, false},
		{"absent field fails other operators", cond("missing", "in", []interface{}{"x"}), false, false},
		{"unknown operator still an error", cond("name", "sounds_like", "x"), false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NewAdvancedEvaluator().Evaluate(tc.node, data)
			if (err != nil) != tc.isErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.isErr)
			}
			if !tc.isErr && got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
