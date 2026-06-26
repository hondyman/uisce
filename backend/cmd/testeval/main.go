package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hondyman/semlayer/backend/internal/services"
)

func main() {
	engine := services.NewValidationRuleEngine(nil, nil)

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

	rule := services.ValidationRuleDefinition{
		ID:            "test_rule_1",
		TenantID:      "t1",
		BPName:        "bp",
		StepName:      "step",
		ConditionJSON: raw,
		Enabled:       true,
	}

	cases := []struct {
		data   map[string]interface{}
		expect bool
	}{
		{map[string]interface{}{"age": 20, "status": "active", "vip": false, "country": "US"}, true},
		{map[string]interface{}{"age": 16, "status": "inactive", "vip": true, "country": "US"}, false},
		{map[string]interface{}{"age": 16, "status": "inactive", "vip": true, "country": "CA"}, true},
	}

	for i, c := range cases {
		res, err := engine.EvaluateRule(context.Background(), "t1", rule, c.data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "case %d error: %v\n", i+1, err)
			os.Exit(2)
		}
		if res.Passed != c.expect {
			fmt.Fprintf(os.Stderr, "case %d expected %v got %v\n", i+1, c.expect, res.Passed)
			os.Exit(3)
		}
		fmt.Printf("case %d passed as expected\n", i+1)
	}

	fmt.Println("All evaluation cases passed")
}
