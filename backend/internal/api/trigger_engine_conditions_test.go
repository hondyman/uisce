package api

import (
	"encoding/json"
	"testing"
)

// Trigger conditions are evaluated by internal/rules/vm.
func TestTriggerConditions_UseRuleEngine(t *testing.T) {
	e := &TriggerEngine{}
	data := map[string]interface{}{"status": "OPEN", "amount": 250.0, "flag": true, "note": "", "desk": "EQUITY"}
	for _, tc := range []struct {
		name   string
		cond   RuleCondition
		passed bool
		reason string
	}{
		{"equals", RuleCondition{"status", "equals", "OPEN"}, true, ""},
		{"notEquals", RuleCondition{"status", "notEquals", "CLOSED"}, true, ""},
		{"greaterThan", RuleCondition{"amount", "greaterThan", 100.0}, true, ""},
		{"lessThanOrEqual", RuleCondition{"amount", "lessThanOrEqual", 250.0}, true, ""},
		// The replaced evaluator's contains returned true for any value at
		// least as long as the search string.
		{"contains really checks containment", RuleCondition{"desk", "contains", "FX"}, false, "EQUITY contains FX is false"},
		{"contains match", RuleCondition{"desk", "contains", "quit"}, true, ""},
		// ...and its inList substring-matched the list's printed form.
		{"inList exact membership", RuleCondition{"status", "inList", []interface{}{"OPEN", "PENDING"}}, true, ""},
		{"inList no substring false positive", RuleCondition{"status", "inList", []interface{}{"REOPENED"}}, false, "OPEN inList [REOPENED] is false"},
		{"isEmpty", RuleCondition{"note", "isEmpty", nil}, true, ""},
		{"isTrue", RuleCondition{"flag", "isTrue", nil}, true, ""},
		{"missing field", RuleCondition{"absent", "equals", "x"}, false, "field not in event data"},
		{"unknown operator", RuleCondition{"status", "soundsLike", "x"}, false, "unknown operator: soundsLike"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := e.evaluateRule(tc.cond, data)
			if r["passed"] != tc.passed || r["reason"] != tc.reason {
				t.Errorf("got passed=%v reason=%q; want passed=%v reason=%q", r["passed"], r["reason"], tc.passed, tc.reason)
			}
		})
	}

	cfg, _ := json.Marshal([]RuleCondition{{"status", "equals", "OPEN"}, {"amount", "greaterThan", 1000.0}})
	met, _, err := e.evaluateConditions(cfg, data)
	if err != nil || met {
		t.Errorf("AND of conditions: met=%v err=%v; want false (second condition fails)", met, err)
	}
}
