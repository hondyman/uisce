package vm

import "testing"

// TestFieldFormatPredicates covers the 9 predicates added for the
// catalog_validation_rules -> rule_ast migration, against the same
// operator vocabulary found in that table (not_empty, is_integer,
// is_number, is_boolean, is_uuid, is_date, is_datetime, is_json,
// max_length).
func TestFieldFormatPredicates(t *testing.T) {
	tests := []struct {
		fn   string
		args []any
		want bool
	}{
		{"NOT_EMPTY", []any{"hello"}, true},
		{"NOT_EMPTY", []any{""}, false},
		{"NOT_EMPTY", []any{nil}, false},

		{"IS_INTEGER", []any{float64(42)}, true},
		{"IS_INTEGER", []any{float64(4.2)}, false},
		{"IS_INTEGER", []any{"42"}, true},
		{"IS_INTEGER", []any{"abc"}, false},

		{"IS_NUMBER", []any{float64(4.2)}, true},
		{"IS_NUMBER", []any{"4.2"}, true},
		{"IS_NUMBER", []any{"abc"}, false},

		{"IS_BOOLEAN", []any{true}, true},
		{"IS_BOOLEAN", []any{"true"}, true},
		{"IS_BOOLEAN", []any{"yes"}, false},

		{"IS_UUID", []any{"99e99e99-99e9-49e9-89e9-99e99e99e999"}, true},
		{"IS_UUID", []any{"not-a-uuid"}, false},

		{"IS_DATE", []any{"2026-09-09"}, true},
		{"IS_DATE", []any{"2026-09-09T00:00:00Z"}, false}, // datetime, not date
		{"IS_DATE", []any{"not a date"}, false},

		{"IS_DATETIME", []any{"2026-09-09T13:00:00Z"}, true},
		{"IS_DATETIME", []any{"2026-09-09"}, false},

		{"IS_JSON", []any{`{"a":1}`}, true},
		{"IS_JSON", []any{"not json"}, false},

		{"MAX_LENGTH", []any{"hello", float64(10)}, true},
		{"MAX_LENGTH", []any{"hello world", float64(5)}, false},
		{"MAX_LENGTH", []any{nil, float64(5)}, true},
	}

	for _, tt := range tests {
		t.Run(tt.fn, func(t *testing.T) {
			spec, ok := LookupFunction(tt.fn)
			if !ok {
				t.Fatalf("no registered function %q", tt.fn)
			}
			got, err := spec.Native(tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			b, ok := got.(bool)
			if !ok {
				t.Fatalf("expected bool result, got %T (%v)", got, got)
			}
			if b != tt.want {
				t.Errorf("%s(%v) = %v, want %v", tt.fn, tt.args, b, tt.want)
			}
		})
	}
}

// TestFieldFormatPredicates_ViaFuncCall proves these are reachable through
// the normal AdvancedEvaluator.Evaluate path (Expression -> FuncCall ->
// FieldRef), not just the raw registry - the shape the migrated rule_ast
// rows will actually use.
func TestFieldFormatPredicates_ViaFuncCall(t *testing.T) {
	ae := NewAdvancedEvaluator()

	node := RuleNode{
		Type: NodeTypeExpression,
		Expression: &Expression{
			Root: &FuncCall{
				Name: "IS_UUID",
				Args: []ExprNode{&FieldRef{Path: "id"}},
			},
		},
	}

	result, err := ae.Evaluate(node, map[string]interface{}{"id": "99e99e99-99e9-49e9-89e9-99e99e99e999"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Errorf("expected IS_UUID(id) to pass for a valid UUID field")
	}

	result, err = ae.Evaluate(node, map[string]interface{}{"id": "not-a-uuid"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Errorf("expected IS_UUID(id) to fail for an invalid UUID field")
	}
}
