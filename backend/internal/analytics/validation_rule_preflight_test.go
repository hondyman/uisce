package analytics

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hondyman/uisce/backend/internal/models"
)

func specWithAST(key, field string, deps ...string) models.PortableRuleSpec {
	ast, _ := json.Marshal(map[string]any{
		"type": "condition", "id": "c1", "field": field, "operator": "greater_than", "value": 100,
	})
	return models.PortableRuleSpec{
		RuleKey: key, Name: key, BOName: "order", Domain: "validation",
		Severity: "BLOCK", Timing: "pre_write", GovernanceStatus: "draft",
		RuleAST: ast, DependsOn: deps,
	}
}

func TestTopoSortOrdersDependencies(t *testing.T) {
	specs := []models.PortableRuleSpec{
		specWithAST("b", "LimitPrice", "a"),
		specWithAST("a", "TargetQuantity"),
	}
	ordered, errs := topoSortRules(specs)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if ordered[0].RuleKey != "a" || ordered[1].RuleKey != "b" {
		t.Fatalf("dependency not ordered first: %s, %s", ordered[0].RuleKey, ordered[1].RuleKey)
	}
}

func TestTopoSortDetectsCycle(t *testing.T) {
	specs := []models.PortableRuleSpec{
		specWithAST("a", "F1", "b"),
		specWithAST("b", "F2", "a"),
	}
	_, errs := topoSortRules(specs)
	found := false
	for _, e := range errs {
		if e.Code == models.ImportErrDepCycle {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected ERR_DEPENDENCY_CYCLE, got %v", errs)
	}
}

func TestUnresolvedTermsFailLoud(t *testing.T) {
	spec := specWithAST("r", "NonExistentTerm")
	ts := &boTermSet{
		boExists: true,
		fields:   map[string]struct{}{"TargetQuantity": {}, "LimitPrice": {}},
	}
	errs := unresolvedTerms(&spec, ts)
	if len(errs) != 1 {
		t.Fatalf("expected 1 unresolved term error, got %v", errs)
	}
	if !strings.Contains(errs[0], "NonExistentTerm") {
		t.Fatalf("error must name the term: %s", errs[0])
	}
}

func TestResolvedTermsPass(t *testing.T) {
	spec := specWithAST("r", "TargetQuantity")
	ts := &boTermSet{boExists: true, fields: map[string]struct{}{"TargetQuantity": {}}}
	if errs := unresolvedTerms(&spec, ts); len(errs) != 0 {
		t.Fatalf("expected resolved, got %v", errs)
	}
}

func TestDiffActions(t *testing.T) {
	spec := specWithAST("r", "TargetQuantity")
	spec.Severity = "BLOCK"

	// CREATE
	if action, _ := diffAgainstExisting(&spec, existingRule{}, false); action != "CREATE" {
		t.Fatalf("want CREATE, got %s", action)
	}

	// NO_OP (identical)
	existing := existingRule{
		RuleKey: "r", Name: "r", BOName: "order", Domain: "validation",
		Severity: "BLOCK", Timing: "pre_write", Governance: "draft",
		CanonicalAST: canonicalForDiff(spec.RuleAST),
	}
	if action, changes := diffAgainstExisting(&spec, existing, false); action != "NO_OP" {
		t.Fatalf("want NO_OP, got %s (%v)", action, changes)
	}

	// UPDATE (severity differs)
	existing.Severity = "WARN"
	action, changes := diffAgainstExisting(&spec, existing, false)
	if action != "UPDATE" || len(changes) != 1 || changes[0] != "severity" {
		t.Fatalf("want UPDATE[severity], got %s %v", action, changes)
	}
}

func TestCoreShadowDetection(t *testing.T) {
	idx := &existingRuleIndex{byKey: map[string]existingRule{
		"core-qty-limit": {RuleKey: "core-qty-limit", Name: "Qty Limit", BOName: "order"},
	}}
	if !idx.shadows("core-qty-limit", "Anything", "order") {
		t.Fatal("key shadow not detected")
	}
	if !idx.shadows("other-key", "Qty Limit", "order") {
		t.Fatal("name shadow not detected")
	}
	if idx.shadows("other-key", "Other Name", "order") {
		t.Fatal("false-positive shadow")
	}
	if idx.shadows("core-qty-limit", "Anything", "trade") { // different BO: not a shadow
		t.Fatal("same key on different BO flagged as shadow")
	}
}
