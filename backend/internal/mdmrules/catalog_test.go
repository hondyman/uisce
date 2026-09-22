package mdmrules

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

func refs(n vm.RuleNode, out map[string]bool) {
	switch n.Type {
	case vm.NodeTypeGroup:
		for _, c := range n.Group.Conditions {
			refs(c, out)
		}
	case vm.NodeTypeCondition:
		out[n.Condition.Field] = true
	case vm.NodeTypeExpression:
		var walk func(e vm.ExprNode)
		walk = func(e vm.ExprNode) {
			switch x := e.(type) {
			case *vm.BinaryExpr:
				walk(x.Left)
				walk(x.Right)
			case *vm.FieldRef:
				out[x.Path] = true
			case *vm.FuncCall:
				for _, a := range x.Args {
					walk(a)
				}
			}
		}
		walk(n.Expression.Root)
	}
}

// Every term a rule references must exist on its BO: a typo would otherwise surface as a rule_error on
// the first write, not at authoring time.
func TestEveryReferencedTermIsInTheBOsVocabulary(t *testing.T) {
	for _, r := range Catalog() {
		vocab := map[string]bool{}
		for _, term := range Vocabulary[r.BO] {
			vocab[term] = true
		}
		if len(vocab) == 0 {
			t.Errorf("%s: BO %q has no vocabulary", r.Name, r.BO)
			continue
		}
		used := map[string]bool{}
		refs(r.AST, used)
		for term := range used {
			if !vocab[term] {
				t.Errorf("%s references %q, which is not a term of %s", r.Name, term, r.BO)
			}
		}
	}
}

// Each rule must accept its passing examples and reject its failing ones. This is what makes a rule
// that "looks wired up but never fires" show up here instead of in production.
func TestEveryRuleAcceptsAndRejectsItsExamples(t *testing.T) {
	ev := vm.NewAdvancedEvaluator()
	for _, r := range Catalog() {
		if len(r.Cases) == 0 {
			t.Errorf("%s has no examples", r.Name)
		}
		var sawPass, sawFail bool
		for _, c := range r.Cases {
			got, err := ev.Evaluate(r.AST, c.Record)
			if err != nil {
				t.Errorf("%s / %s: evaluation error: %v", r.Name, c.Name, err)
				continue
			}
			if got != c.Pass {
				t.Errorf("%s / %s: got pass=%v, want %v", r.Name, c.Name, got, c.Pass)
			}
			sawPass = sawPass || c.Pass
			sawFail = sawFail || !c.Pass
		}
		if !sawPass || !sawFail {
			t.Errorf("%s needs at least one passing and one failing example", r.Name)
		}
	}
}

// The AST is stored as JSON in the catalog node; it must survive the round trip unchanged in behaviour.
func TestEveryASTRoundTripsThroughJSON(t *testing.T) {
	ev := vm.NewAdvancedEvaluator()
	for _, r := range Catalog() {
		b, err := json.Marshal(r.AST)
		if err != nil {
			t.Fatalf("%s: marshal: %v", r.Name, err)
		}
		var back vm.RuleNode
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("%s: unmarshal: %v", r.Name, err)
		}
		for _, c := range r.Cases {
			want, _ := ev.Evaluate(r.AST, c.Record)
			got, err := ev.Evaluate(back, c.Record)
			if err != nil || got != want {
				t.Errorf("%s / %s: after round trip got (%v, %v), want %v", r.Name, c.Name, got, err, want)
			}
		}
	}
}

func TestNamesAreUniqueAndASTsDoNotDuplicateWithinABO(t *testing.T) {
	names := map[string]bool{}
	asts := map[string]string{}
	for _, r := range Catalog() {
		if names[r.Name] {
			t.Errorf("duplicate rule name %s", r.Name)
		}
		names[r.Name] = true
		b, _ := json.Marshal(r.AST)
		key := r.BO + "|" + string(b)
		if other, ok := asts[key]; ok {
			// UpsertValidationRule rejects a rule whose AST equals another rule's on the same BO.
			t.Errorf("%s and %s have identical conditions on %s", r.Name, other, r.BO)
		}
		asts[key] = r.Name
	}
}

func TestOnlySourcingRulesAreBindingScoped(t *testing.T) {
	var scoped []string
	for _, r := range Catalog() {
		if r.Scope == ScopeCurrentBindings {
			scoped = append(scoped, r.Name)
			if r.Category != catSourcing {
				t.Errorf("%s is binding-scoped but not a sourcing rule", r.Name)
			}
		}
	}
	if want := []string{"mdm.issuer.sourced_from_a_system"}; !reflect.DeepEqual(scoped, want) {
		t.Errorf("binding-scoped rules = %v; want %v", scoped, want)
	}
}

// Guard rails from the design: no rule on enumerations/UX-controlled terms, and none on the ambiguous IssuerId.
func TestNoRulesOnStatusPriorityIsActiveOrIssuerId(t *testing.T) {
	for _, r := range Catalog() {
		used := map[string]bool{}
		refs(r.AST, used)
		for term := range used {
			if term == "Status" || term == "Priority" || term == "IssuerId" || strings.HasSuffix(term, "IsActive") {
				t.Errorf("%s references %s, which rules must not cover", r.Name, term)
			}
		}
	}
}

func TestSeverityTimingAndCategoryAreValid(t *testing.T) {
	for _, r := range Catalog() {
		if r.Severity != "BLOCK" && r.Severity != "WARN" {
			t.Errorf("%s: bad severity %q", r.Name, r.Severity)
		}
		if r.Timing != "pre_write" && r.Timing != "reconcile" {
			t.Errorf("%s: bad timing %q", r.Name, r.Timing)
		}
		if r.Category == "" || r.Description == "" {
			t.Errorf("%s: missing category or description", r.Name)
		}
	}
}
