package services

import (
	"context"
	"encoding/json"
	"testing"
)

func TestEvaluateNestedConditions(t *testing.T) {
	// Set up engine with nil DB (we only need evaluation logic)
	// Pass nil for InstanceProvider as this test doesn't use traversal
	engine := NewValidationRuleEngine(nil, nil)

	// Define a complex nested condition:
	// (age >= 18 AND status = "active") OR (vip = true AND NOT (country = "US"))
	cond := map[string]interface{}{
		"or": []interface{}{
			map[string]interface{}{"and": []interface{}{
				map[string]interface{}{"field": "age", "operator": ">=", "value": 18},
				map[string]interface{}{"field": "status", "operator": "=", "value": "active"},
			}},
			map[string]interface{}{"and": []interface{}{
				map[string]interface{}{"field": "vip", "operator": "=", "value": true},
				map[string]interface{}{"not": map[string]interface{}{"field": "country", "operator": "=", "value": "US"}},
			}},
		},
	}

	raw, _ := json.Marshal(cond)

	rule := ValidationRuleDefinition{
		ID:            "test_rule_1",
		TenantID:      "t1",
		BPName:        "bp",
		StepName:      "step",
		ConditionJSON: raw,
		Enabled:       true,
	}

	// Case 1: age 20, status active -> should pass
	data1 := map[string]interface{}{"age": 20, "status": "active", "vip": false, "country": "US"}
	res1, err := engine.EvaluateRule(context.Background(), "t1", rule, data1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res1.Passed {
		t.Fatalf("expected rule to pass for data1 but it failed")
	}

	// Case 2: vip true but country US -> NOT(country=US) false -> overall false
	data2 := map[string]interface{}{"age": 16, "status": "inactive", "vip": true, "country": "US"}
	res2, err := engine.EvaluateRule(context.Background(), "t1", rule, data2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.Passed {
		t.Fatalf("expected rule to fail for data2 but it passed")
	}

	// Case 3: vip true and country not US -> should pass
	data3 := map[string]interface{}{"age": 16, "status": "inactive", "vip": true, "country": "CA"}
	res3, err := engine.EvaluateRule(context.Background(), "t1", rule, data3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res3.Passed {
		t.Fatalf("expected rule to pass for data3 but it failed")
	}
}
