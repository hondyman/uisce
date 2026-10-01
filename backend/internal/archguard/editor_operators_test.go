package archguard

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// TestEditorOperatorsAreImplemented fails when the rule editor offers an
// operator the engine cannot evaluate - a rule an analyst can author but
// that errors ("unknown operator") at enforcement time.
func TestEditorOperatorsAreImplemented(t *testing.T) {
	editor := filepath.Join(filepath.Dir(backendRoot(t)), "frontend", "src", "components", "ExpressionBuilder", "AdvancedConditionBuilder.tsx")
	src, err := os.ReadFile(editor)
	if err != nil {
		t.Fatalf("rule editor not found: %v", err)
	}
	ops := map[string]bool{}
	for _, m := range regexp.MustCompile(`\{ value: '([a-z_]+)', label:`).FindAllStringSubmatch(string(src), -1) {
		ops[m[1]] = true
	}
	if len(ops) < 20 {
		t.Fatalf("parsed only %d editor operators - has the operator list's shape changed?", len(ops))
	}

	data := map[string]interface{}{"f": "2026-01-01"}
	for op := range ops {
		node := vm.RuleNode{Type: vm.NodeTypeCondition, Condition: &vm.RuleCondition{
			Field: "f", FieldPath: "f", Operator: op, Value: []interface{}{1, 2},
		}}
		if _, err := vm.NewAdvancedEvaluator().Evaluate(node, data); err != nil && strings.Contains(err.Error(), "unknown operator") {
			t.Errorf("editor offers %q but the rule engine does not implement it", op)
		}
	}
}
