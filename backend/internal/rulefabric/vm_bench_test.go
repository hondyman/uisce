package rulefabric

import (
	"encoding/json"
	"testing"
)

func parseGroup(raw json.RawMessage) *ConditionGroup {
	var g ConditionGroup
	_ = jsonUnmarshal(raw, &g)
	return &g
}

// BenchmarkEvaluate_Recursive measures the legacy recursive evaluator
// before the VM migration (used as baseline).
func BenchmarkEvaluate_Recursive(b *testing.B) {
	rule := minimalRuleWithLogic("bench", "nested",
		`{"type":"group","operator":"AND","conditions":[
			{"type":"condition","field":"customer.tier","operator":"==","value":"GOLD"},
			{"type":"condition","field":"customer.balance","operator":">","value":10000},
			{"type":"condition","field":"customer.country","operator":"in","value":["US","CA","GB"]}
		]}`)

	re, err := NewRuleEvaluator(nil)
	if err != nil {
		b.Fatal(err)
	}
	group := parseGroup(rule.Logic.ConditionJSON)
	evalCtx := &EvaluationContext{
		Data: map[string]any{
			"customer": map[string]any{
				"tier":    "GOLD",
				"balance": float64(15000),
				"country": "US",
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := re.Evaluate(nil, rule, evalCtx); err != nil {
			b.Fatal(err)
		}
		_ = group // keep reference
	}
}

// BenchmarkEvaluate_VM measures the VM-backed path with the same fixture.
func BenchmarkEvaluate_VM(b *testing.B) {
	rule := minimalRuleWithLogic("bench-vm", "nested",
		`{"type":"group","operator":"AND","conditions":[
			{"type":"condition","field":"customer.tier","operator":"==","value":"GOLD"},
			{"type":"condition","field":"customer.balance","operator":">","value":10000},
			{"type":"condition","field":"customer.country","operator":"in","value":["US","CA","GB"]}
		]}`)

	mgr := NewVMManager()
	group := parseGroup(rule.Logic.ConditionJSON)
	extractRuleFabricPaths(group, mgr.syms, mgr.enums)
	mgr.syms.Freeze()
	mgr.enums.Freeze()

	re, err := NewRuleEvaluator(nil)
	if err != nil {
		b.Fatal(err)
	}
	re.vmManager = mgr
	evalCtx := &EvaluationContext{
		Data: map[string]any{
			"customer": map[string]any{
				"tier":    "GOLD",
				"balance": float64(15000),
				"country": "US",
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := re.Evaluate(nil, rule, evalCtx); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEvaluate_VM_Parallel measures throughput on all cores.
func BenchmarkEvaluate_VM_Parallel(b *testing.B) {
	rule := minimalRuleWithLogic("bench-vm-parallel", "nested",
		`{"type":"group","operator":"AND","conditions":[
			{"type":"condition","field":"customer.tier","operator":"==","value":"GOLD"},
			{"type":"condition","field":"customer.balance","operator":">","value":10000},
			{"type":"condition","field":"customer.country","operator":"in","value":["US","CA","GB"]}
		]}`)

	mgr := NewVMManager()
	group := parseGroup(rule.Logic.ConditionJSON)
	extractRuleFabricPaths(group, mgr.syms, mgr.enums)
	mgr.syms.Freeze()
	mgr.enums.Freeze()

	re, err := NewRuleEvaluator(nil)
	if err != nil {
		b.Fatal(err)
	}
	re.vmManager = mgr
	evalCtx := &EvaluationContext{
		Data: map[string]any{
			"customer": map[string]any{
				"tier":    "GOLD",
				"balance": float64(15000),
				"country": "US",
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := re.Evaluate(nil, rule, evalCtx); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// dummy import to keep uuid reference if needed in future
var _ = mustParseUUID
// BenchmarkEvaluate_VM_ManagedHotPath bypasses Evaluate's per-call
// JSON unmarshal and goes straight through VMManager.EvaluateWithFallback
// with a pre-parsed ConditionGroup. This reflects the production
// pattern where each rule is compiled once at startup and re-evaluated
// many times against incoming records.
func BenchmarkEvaluate_VM_ManagedHotPath(b *testing.B) {
	rule := minimalRuleWithLogic("bench-hot", "nested",
		`{"type":"group","operator":"AND","conditions":[
			{"type":"condition","field":"customer.tier","operator":"==","value":"GOLD"},
			{"type":"condition","field":"customer.balance","operator":">","value":10000},
			{"type":"condition","field":"customer.country","operator":"in","value":["US","CA","GB"]}
		]}`)

	mgr := NewVMManager()
	group := parseGroup(rule.Logic.ConditionJSON)
	extractRuleFabricPaths(group, mgr.syms, mgr.enums)
	mgr.syms.Freeze()
	mgr.enums.Freeze()

	// Pre-compile to amortise the cost of compile over the loop.
	_, _ = mgr.EvaluateWithFallback(rule, group, nil, nil,
		func() (bool, EvaluationDetails) { return false, EvaluationDetails{} })

	evalCtx := &EvaluationContext{
		Data: map[string]any{
			"customer": map[string]any{
				"tier":    "GOLD",
				"balance": float64(15000),
				"country": "US",
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = mgr.EvaluateWithFallback(rule, group, evalCtx.Data, nil,
			func() (bool, EvaluationDetails) { return false, EvaluationDetails{} })
	}
}

// BenchmarkEvaluate_VM_ManagedHotPath_Parallel is the headline
// throughput benchmark for the rulefabric VM path.
func BenchmarkEvaluate_VM_ManagedHotPath_Parallel(b *testing.B) {
	rule := minimalRuleWithLogic("bench-hot-par", "nested",
		`{"type":"group","operator":"AND","conditions":[
			{"type":"condition","field":"customer.tier","operator":"==","value":"GOLD"},
			{"type":"condition","field":"customer.balance","operator":">","value":10000},
			{"type":"condition","field":"customer.country","operator":"in","value":["US","CA","GB"]}
		]}`)

	mgr := NewVMManager()
	group := parseGroup(rule.Logic.ConditionJSON)
	extractRuleFabricPaths(group, mgr.syms, mgr.enums)
	mgr.syms.Freeze()
	mgr.enums.Freeze()

	_, _ = mgr.EvaluateWithFallback(rule, group, nil, nil,
		func() (bool, EvaluationDetails) { return false, EvaluationDetails{} })

	evalCtx := &EvaluationContext{
		Data: map[string]any{
			"customer": map[string]any{
				"tier":    "GOLD",
				"balance": float64(15000),
				"country": "US",
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = mgr.EvaluateWithFallback(rule, group, evalCtx.Data, nil,
				func() (bool, EvaluationDetails) { return false, EvaluationDetails{} })
		}
	})
}
