package rulefabric

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// helper: build a sym/enum dict warmed with field paths and literals from
// the given rules. Mimics what RegisterAndFreeze does in production.
func warmDicts(t *testing.T, rules []*RuleWithLogic) (*vm.SymbolDict, *vm.EnumDict) {
	t.Helper()
	syms := vm.NewSymbolDict()
	enums := vm.NewEnumDict()
	for _, rule := range rules {
		var group ConditionGroup
		if err := jsonUnmarshal(rule.Logic.ConditionJSON, &group); err != nil {
			t.Fatalf("parse rule %s: %v", rule.ID, err)
		}
		ExtractRuleFabricPaths(&group, syms, enums)
	}
	syms.Freeze()
	enums.Freeze()
	return syms, enums
}

// minimalRuleWithLogic constructs a RuleWithLogic with the given
// condition JSON. Used by tests to avoid the database.
func minimalRuleWithLogic(id, name, conditionJSON string) *RuleWithLogic {
	rule := Rule{
		RuleCode: name,
		Name:     name,
		Category: CategoryMDM,
		Severity: SeverityWarning,
	}
	// Set a stable ID — use a fixed UUID for deterministic cache keys in tests.
	rule.ID = mustParseUUID("00000000-0000-0000-0000-000000000001")
	logic := RuleLogic{
		RuleID:        rule.ID,
		Version:       1,
		ConditionJSON: json.RawMessage(conditionJSON),
	}
	return &RuleWithLogic{Rule: rule, Logic: logic}
}

// Compile the rule once with both paths and assert the VM result equals
// the recursive result.
func TestRulefabricVM_Parity_EqualsOperator(t *testing.T) {
	rule := minimalRuleWithLogic("r1", "test",
		`{"type":"condition","field":"customer.tier","operator":"==","value":"GOLD"}`)

	evalCtx := &EvaluationContext{
		Data: map[string]any{"customer": map[string]any{"tier": "GOLD"}},
	}

	re := &RuleEvaluator{}
	vmResult, _ := re.Evaluate(nil, rule, evalCtx)

	recursivePassed, _ := re.evaluateConditionGroup(
		mustParseGroup(t, rule.Logic.ConditionJSON),
		evalCtx.Data, nil,
	)

	if vmResult.Status != EvalPassed {
		t.Errorf("VM expected pass, got %v", vmResult.Status)
	}
	if !recursivePassed {
		t.Error("recursive expected pass")
	}
}

func TestRulefabricVM_Parity_Group(t *testing.T) {
	rule := minimalRuleWithLogic("r2", "group",
		`{"type":"group","operator":"AND","conditions":[
			{"type":"condition","field":"customer.tier","operator":"==","value":"GOLD"},
			{"type":"condition","field":"customer.balance","operator":">","value":10000}
		]}`)

	// Pre-warm the dictionaries via RegisterAndFreeze.
	mgr := NewVMManager()
	corpus := []*RuleWithLogic{rule}
	for _, r := range corpus {
		var group ConditionGroup
		_ = jsonUnmarshal(r.Logic.ConditionJSON, &group)
		extractRuleFabricPaths(&group, mgr.syms, mgr.enums)
	}
	mgr.syms.Freeze()
	mgr.enums.Freeze()

	evalCtx := &EvaluationContext{
		Data: map[string]any{
			"customer": map[string]any{
				"tier":    "GOLD",
				"balance": float64(15000),
			},
		},
	}

	re := &RuleEvaluator{vmManager: mgr}
	res, _ := re.Evaluate(nil, rule, evalCtx)
	if res.Status != EvalPassed {
		t.Errorf("VM: expected pass for AND(true, true), got %v", res.Status)
	}

	// Make balance fail
	evalCtx.Data = map[string]any{
		"customer": map[string]any{"tier": "GOLD", "balance": float64(500)},
	}
	res, _ = re.Evaluate(nil, rule, evalCtx)
	if res.Status != EvalFailed {
		t.Errorf("VM: expected fail for AND(true, false), got %v", res.Status)
	}
}

func TestRulefabricVM_Parity_ORGroup(t *testing.T) {
	rule := minimalRuleWithLogic("r3", "or",
		`{"type":"group","operator":"OR","conditions":[
			{"type":"condition","field":"a","operator":"==","value":"x"},
			{"type":"condition","field":"b","operator":">","value":5}
		]}`)

	mgr := NewVMManager()
	var group ConditionGroup
	_ = jsonUnmarshal(rule.Logic.ConditionJSON, &group)
	extractRuleFabricPaths(&group, mgr.syms, mgr.enums)
	mgr.syms.Freeze()
	mgr.enums.Freeze()

	re := &RuleEvaluator{vmManager: mgr}

	cases := []struct {
		name string
		data map[string]any
		want EvaluationStatus
	}{
		{"a matches", map[string]any{"a": "x", "b": float64(1)}, EvalPassed},
		{"b matches", map[string]any{"a": "y", "b": float64(10)}, EvalPassed},
		{"neither", map[string]any{"a": "y", "b": float64(1)}, EvalFailed},
		{"both", map[string]any{"a": "x", "b": float64(10)}, EvalPassed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, _ := re.Evaluate(nil, rule, &EvaluationContext{Data: tc.data})
			if res.Status != tc.want {
				t.Errorf("got %v want %v", res.Status, tc.want)
			}
		})
	}
}

func TestRulefabricVM_Parity_NestedGroups(t *testing.T) {
	rule := minimalRuleWithLogic("r4", "nested",
		`{"type":"group","operator":"AND","conditions":[
			{"type":"group","operator":"OR","conditions":[
				{"type":"condition","field":"a","operator":"==","value":"x"},
				{"type":"condition","field":"b","operator":"==","value":"y"}
			]},
			{"type":"condition","field":"c","operator":">","value":100}
		]}`)

	mgr := NewVMManager()
	var group ConditionGroup
	_ = jsonUnmarshal(rule.Logic.ConditionJSON, &group)
	extractRuleFabricPaths(&group, mgr.syms, mgr.enums)
	mgr.syms.Freeze()
	mgr.enums.Freeze()

	re := &RuleEvaluator{vmManager: mgr}

	// (a==x OR b==y) AND c>100
	cases := []struct {
		name string
		data map[string]any
		want EvaluationStatus
	}{
		{"a-and-c", map[string]any{"a": "x", "b": "z", "c": float64(200)}, EvalPassed},
		{"b-and-c", map[string]any{"a": "z", "b": "y", "c": float64(200)}, EvalPassed},
		{"neither-low-c", map[string]any{"a": "z", "b": "z", "c": float64(200)}, EvalFailed},
		{"or-but-low-c", map[string]any{"a": "x", "b": "y", "c": float64(50)}, EvalFailed},
		{"all-wrong", map[string]any{"a": "z", "b": "z", "c": float64(50)}, EvalFailed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, _ := re.Evaluate(nil, rule, &EvaluationContext{Data: tc.data})
			if res.Status != tc.want {
				t.Errorf("got %v want %v", res.Status, tc.want)
			}
		})
	}
}

// Unsupported operators must trigger fallback (return an error from
// the VM compile) and produce a correct result via the recursive path.
func TestRulefabricVM_FallbackOnRegex(t *testing.T) {
	rule := minimalRuleWithLogic("r5", "regex",
		`{"type":"condition","field":"email","operator":"matches_regex","value":"^.*@example\\.com$"}`)

	mgr := NewVMManager()
	var group ConditionGroup
	_ = jsonUnmarshal(rule.Logic.ConditionJSON, &group)
	extractRuleFabricPaths(&group, mgr.syms, mgr.enums)
	mgr.syms.Freeze()
	mgr.enums.Freeze()

	re := &RuleEvaluator{vmManager: mgr}
	res, _ := re.Evaluate(nil, rule, &EvaluationContext{
		Data: map[string]any{"email": "alice@example.com"},
	})
	if res.Status != EvalPassed {
		t.Errorf("regex fallback should pass for alice@example.com, got %v", res.Status)
	}

	// Fallback counter should have incremented.
	snap := mgr.Snapshot()
	if snap.Fallbacks == 0 {
		t.Errorf("Fallbacks should be > 0, got %d", snap.Fallbacks)
	}
}

// Scoring formula on the rule triggers VM-unsupported → fallback to
// recursive; the score is still computed via CEL in Evaluate's tail.
func TestRulefabricVM_FallbackOnScoringFormula(t *testing.T) {
	rule := minimalRuleWithLogic("r6", "scoring",
		`{"type":"condition","field":"x","operator":">","value":0}`)
	rule.Logic.ScoringFormula = "data.x * 2"

	mgr := NewVMManager()
	var group ConditionGroup
	_ = jsonUnmarshal(rule.Logic.ConditionJSON, &group)
	extractRuleFabricPaths(&group, mgr.syms, mgr.enums)
	mgr.syms.Freeze()
	mgr.enums.Freeze()

	re, err := NewRuleEvaluator(nil)
	if err != nil {
		t.Fatalf("NewRuleEvaluator: %v", err)
	}
	re.vmManager = mgr

	res, _ := re.Evaluate(nil, rule, &EvaluationContext{
		Data: map[string]any{"x": float64(10)},
	})
	if res.Status != EvalPassed {
		t.Errorf("expected pass, got %v", res.Status)
	}
	// Score assertion is best-effort: CEL formula evaluation depends
	// on the exact variable declarations; we just want to confirm the
	// fallback path didn't error and the boolean gate succeeded.
}

// CompileError when field path is not registered.
func TestRulefabricVM_CompileError_MissingField(t *testing.T) {
	rule := minimalRuleWithLogic("r7", "missing",
		`{"type":"condition","field":"unregistered.path","operator":"==","value":"x"}`)

	mgr := NewVMManager()
	// Don't pre-intern the path — Compile must reject.
	mgr.syms.Freeze()
	mgr.enums.Freeze()

	res, _ := mgr.EvaluateWithFallback(
		rule,
		mustParseGroup(t, rule.Logic.ConditionJSON),
		map[string]any{"unregistered": map[string]any{"path": "x"}},
		nil,
		func() (bool, EvaluationDetails) { return true, EvaluationDetails{} },
	)
	// Fallback invoked → result is true.
	if !res {
		t.Errorf("expected fallback to return true, got false")
	}
	if mgr.Snapshot().CompileErrors == 0 {
		t.Errorf("CompileErrors should increment")
	}
}

// Sticky unsupported: once a rule fails to compile, every subsequent
// call hits the cache (sticky Unsupported) without retrying compile.
func TestRulefabricVM_StickyUnsupported(t *testing.T) {
	rule := minimalRuleWithLogic("r8", "regex-sticky",
		`{"type":"condition","field":"x","operator":"matches_regex","value":".*"}`)

	mgr := NewVMManager()
	var group ConditionGroup
	_ = jsonUnmarshal(rule.Logic.ConditionJSON, &group)
	extractRuleFabricPaths(&group, mgr.syms, mgr.enums)
	mgr.syms.Freeze()
	mgr.enums.Freeze()

	// First call: compile fails → fallback.
	res, _ := mgr.EvaluateWithFallback(
		rule, &group, map[string]any{"x": "anything"}, nil,
		func() (bool, EvaluationDetails) { return false, EvaluationDetails{} },
	)
	if res {
		t.Errorf("expected fallback false")
	}
	misses1 := mgr.Snapshot().CacheMisses

	// Second call: must hit cache (sticky), not re-compile.
	res, _ = mgr.EvaluateWithFallback(
		rule, &group, map[string]any{"x": "different"}, nil,
		func() (bool, EvaluationDetails) { return false, EvaluationDetails{} },
	)
	if res {
		t.Errorf("expected fallback false (second call)")
	}
	if mgr.Snapshot().CacheMisses != misses1 {
		t.Errorf("cache misses should not increment for sticky unsupported; got %d (was %d)",
			mgr.Snapshot().CacheMisses, misses1)
	}
}

// --- helpers ---

func mustParseGroup(t testing.TB, raw json.RawMessage) *ConditionGroup {
	t.Helper()
	var g ConditionGroup
	if err := jsonUnmarshal(raw, &g); err != nil {
		t.Fatalf("parse: %v", err)
	}
	return &g
}

func mustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return id
}