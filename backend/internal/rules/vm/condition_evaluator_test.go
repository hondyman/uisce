package vm

import "testing"

// is_null/is_not_null were offered by AdvancedConditionBuilder.tsx's
// operator dropdown (number/date/boolean/enum field types) with no
// matching case in compareValues - selecting either produced "unknown
// operator" once a field resolved to a real value, or a silent false
// when the field was missing, never the actual null-check the label
// promised. Fixed in evaluateSimpleCondition, ahead of compareValues.
func TestEvaluateSimpleCondition_IsNull(t *testing.T) {
	simpleCond := func(field, operator string) map[string]interface{} {
		return map[string]interface{}{"type": "simple", "field": field, "operator": operator}
	}

	ce := NewConditionEvaluator()

	cases := []struct {
		name     string
		data     map[string]interface{}
		field    string
		operator string
		want     bool
	}{
		{"is_null true for missing field", map[string]interface{}{}, "risk_score", "is_null", true},
		{"is_null true for present-but-null field", map[string]interface{}{"risk_score": nil}, "risk_score", "is_null", true},
		{"is_null false for a present value", map[string]interface{}{"risk_score": 42.0}, "risk_score", "is_null", false},
		{"is_not_null false for missing field", map[string]interface{}{}, "risk_score", "is_not_null", false},
		{"is_not_null false for present-but-null field", map[string]interface{}{"risk_score": nil}, "risk_score", "is_not_null", false},
		{"is_not_null true for a present value", map[string]interface{}{"risk_score": 42.0}, "risk_score", "is_not_null", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ce.EvaluateWithHierarchy(simpleCond(c.field, c.operator), c.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// Every operator not special-cased still goes through compareValues
// unchanged - this is a narrow addition, not a rewrite of the shared
// "field not found" behavior for every other operator.
func TestEvaluateSimpleCondition_OtherOperatorsUnaffected(t *testing.T) {
	ce := NewConditionEvaluator()
	cond := map[string]interface{}{
		"type": "simple", "field": "missing_field", "operator": "equals", "value": 1.0,
	}
	got, err := ce.EvaluateWithHierarchy(cond, map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != false {
		t.Errorf("got %v, want false (missing field, non-null-check operator)", got)
	}
}
