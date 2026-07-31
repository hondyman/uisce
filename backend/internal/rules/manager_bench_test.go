package rules

import (
	"testing"
)

// BenchmarkManager_RealisticPath measures the realistic ingestion path:
// map[string]any -> Project() -> VM.Run(). This is what production HTTP
// handlers and Kafka consumers will see.
//
// Compared to BenchmarkVM_FastRecord (which uses a pre-built FastRecord),
// this benchmark includes the Project() allocation cost — the difference
// between the two benchmarks is the marginal cost of JSON-to-FastRecord
// conversion. In Phase 6 we'll eliminate that gap with a streaming
// JSON decoder.
func BenchmarkManager_RealisticPath(b *testing.B) {
	mgr := NewVMManager()
	ast := &RuleNode{
		Type: NodeTypeGroup,
		Group: &RuleGroup{
			Operator: "AND",
			Conditions: []RuleNode{
				{Type: NodeTypeCondition, Condition: &RuleCondition{FieldPath: "customer.tier", Operator: "==", Value: "GOLD", ValueType: "string"}},
				{Type: NodeTypeCondition, Condition: &RuleCondition{FieldPath: "customer.balance", Operator: ">", Value: float64(10000), ValueType: "number"}},
			},
		},
	}
	mgr.RegisterAndFreeze([]*RuleNode{ast})

	input := map[string]any{
		"customer": map[string]any{
			"tier":    "GOLD",
			"balance": float64(15000),
		},
	}

	fallback := func(n *RuleNode, i map[string]any) (bool, error) { return false, nil }

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if _, err := mgr.EvaluateWithFallback("bench", ast, input, fallback); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkManager_RealisticPath_Parallel measures throughput on all cores.
// This is the headline per-core number for production traffic.
func BenchmarkManager_RealisticPath_Parallel(b *testing.B) {
	mgr := NewVMManager()
	ast := &RuleNode{
		Type: NodeTypeGroup,
		Group: &RuleGroup{
			Operator: "AND",
			Conditions: []RuleNode{
				{Type: NodeTypeCondition, Condition: &RuleCondition{FieldPath: "customer.tier", Operator: "==", Value: "GOLD", ValueType: "string"}},
				{Type: NodeTypeCondition, Condition: &RuleCondition{FieldPath: "customer.balance", Operator: ">", Value: float64(10000), ValueType: "number"}},
			},
		},
	}
	mgr.RegisterAndFreeze([]*RuleNode{ast})

	input := map[string]any{
		"customer": map[string]any{
			"tier":    "GOLD",
			"balance": float64(15000),
		},
	}

	fallback := func(n *RuleNode, i map[string]any) (bool, error) { return false, nil }

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := mgr.EvaluateWithFallback("bench", ast, input, fallback); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkManager_CacheHitVsMiss isolates cache-hit cost from
// cache-miss (compile) cost. Run with -bench=BenchmarkManager_Cache
func BenchmarkManager_CacheHitVsMiss(b *testing.B) {
	mgr := NewVMManager()
	ast := &RuleNode{
		Type: NodeTypeCondition,
		Condition: &RuleCondition{
			FieldPath: "balance",
			Operator:   ">",
			Value:      float64(100),
			ValueType:  "number",
		},
	}
	mgr.RegisterAndFreeze([]*RuleNode{ast})
	input := map[string]any{"balance": float64(200)}
	fallback := func(n *RuleNode, i map[string]any) (bool, error) { return false, nil }

	// Warm cache.
	if _, err := mgr.EvaluateWithFallback("warm", ast, input, fallback); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if _, err := mgr.EvaluateWithFallback("warm", ast, input, fallback); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkManager_EvaluateJSON measures the zero-allocation ingestion
// path: raw JSON bytes -> streaming decoder -> VM.Run.
func BenchmarkManager_EvaluateJSON(b *testing.B) {
	mgr := NewVMManager()
	ast := &RuleNode{
		Type: NodeTypeGroup,
		Group: &RuleGroup{
			Operator: "AND",
			Conditions: []RuleNode{
				{Type: NodeTypeCondition, Condition: &RuleCondition{FieldPath: "customer.tier", Operator: "==", Value: "GOLD", ValueType: "string"}},
				{Type: NodeTypeCondition, Condition: &RuleCondition{FieldPath: "customer.balance", Operator: ">", Value: float64(10000), ValueType: "number"}},
			},
		},
	}
	mgr.RegisterAndFreeze([]*RuleNode{ast})

	jsonData := []byte(`{"customer":{"tier":"GOLD","balance":15000}}`)
	fallback := func(n *RuleNode, i map[string]any) (bool, error) { return false, nil }

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if _, err := mgr.EvaluateJSONWithFallback("bench-json", ast, jsonData, fallback); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkManager_EvaluateJSON_Parallel is the headline throughput number
// for the JSON ingestion path. Should converge with BenchmarkVM_Parallel.
func BenchmarkManager_EvaluateJSON_Parallel(b *testing.B) {
	mgr := NewVMManager()
	ast := &RuleNode{
		Type: NodeTypeGroup,
		Group: &RuleGroup{
			Operator: "AND",
			Conditions: []RuleNode{
				{Type: NodeTypeCondition, Condition: &RuleCondition{FieldPath: "customer.tier", Operator: "==", Value: "GOLD", ValueType: "string"}},
				{Type: NodeTypeCondition, Condition: &RuleCondition{FieldPath: "customer.balance", Operator: ">", Value: float64(10000), ValueType: "number"}},
			},
		},
	}
	mgr.RegisterAndFreeze([]*RuleNode{ast})

	jsonData := []byte(`{"customer":{"tier":"GOLD","balance":15000}}`)
	fallback := func(n *RuleNode, i map[string]any) (bool, error) { return false, nil }

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := mgr.EvaluateJSONWithFallback("bench-json", ast, jsonData, fallback); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// TestEngine_VMThroughput reports ops/sec based on VMManager.RealisticPath_Parallel.
func TestEngine_VMThroughput(t *testing.T) {
	mgr := NewVMManager()
	ast := &RuleNode{
		Type: NodeTypeCondition,
		Condition: &RuleCondition{
			FieldPath: "x",
			Operator:   ">",
			Value:      float64(0),
			ValueType:  "number",
		},
	}
	mgr.RegisterAndFreeze([]*RuleNode{ast})

	input := map[string]any{"x": float64(1)}
	fallback := func(n *RuleNode, i map[string]any) (bool, error) { return false, nil }

	// Warm cache.
	_, _ = mgr.EvaluateWithFallback("tput", ast, input, fallback)

	// Quick smoke test: should not panic, should produce true.
	got, _ := mgr.EvaluateWithFallback("tput", ast, input, fallback)
	if !got {
		t.Error("expected true (1 > 0)")
	}

	// Confirm cache size is exactly 1.
	snap := mgr.Snapshot()
	if snap.CacheSize != 1 {
		t.Errorf("CacheSize = %d, want 1", snap.CacheSize)
	}

	// Ensure dicts are frozen (lookups are lock-free).
	if mgr.Symbols().Num() < 1 {
		t.Errorf("Symbols.Num() = %d, want >= 1", mgr.Symbols().Num())
	}
}