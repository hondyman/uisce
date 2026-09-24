package metadata

import (
	"testing"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// BO synthesis must only ever suggest rules the single rule engine can run.
func TestSynthesizedRulesAreRuleEngineExpressions(t *testing.T) {
	for _, r := range synthesizedRuleSuggestions() {
		if _, err := vm.ParseExpression(r.Script); err != nil {
			t.Errorf("rule %s: %q is not a rule-engine expression: %v", r.RuleName, r.Script, err)
		}
	}
}
