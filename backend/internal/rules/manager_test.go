package rules

import (
	"testing"
)

func TestManager_EvaluateWithFallback_Supported(t *testing.T) {
	mgr := NewVMManager()

	ast := &RuleNode{
		Type: NodeTypeCondition,
		Condition: &RuleCondition{
			FieldPath: "customer.balance",
			Operator:   ">",
			Value:      float64(10000),
			ValueType:  "number",
		},
	}
	mgr.RegisterAndFreeze([]*RuleNode{ast})

	input := map[string]any{"customer": map[string]any{"balance": float64(15000)}}

	fallback := func(n *RuleNode, i map[string]any) (bool, error) {
		t.Fatal("fallback should not be called for supported rule")
		return false, nil
	}

	got, err := mgr.EvaluateWithFallback("rule-1", ast, input, fallback)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("expected true (balance > 10000), got false")
	}
}

func TestManager_EvaluateWithFallback_Unsupported(t *testing.T) {
	mgr := NewVMManager()

	// `contains` is unsupported in v1; the compile returns an error
	// and the manager must fall back to the recursive evaluator.
	ast := &RuleNode{
		Type: NodeTypeCondition,
		Condition: &RuleCondition{
			FieldPath: "customer.name",
			Operator:   "contains",
			Value:      "Smith",
			ValueType:  "string",
		},
	}
	mgr.RegisterAndFreeze([]*RuleNode{ast})

	input := map[string]any{"customer": map[string]any{"name": "John Smith"}}

	fallbackInvoked := false
	fallback := func(n *RuleNode, i map[string]any) (bool, error) {
		fallbackInvoked = true
		return true, nil
	}

	_, err := mgr.EvaluateWithFallback("rule-2", ast, input, fallback)
	if err != nil {
		t.Fatal(err)
	}
	if !fallbackInvoked {
		t.Error("fallback should have been invoked for unsupported operator")
	}

	snap := mgr.Snapshot()
	if snap.Fallbacks == 0 {
		t.Error("VMSnapshot.Fallbacks should be >= 1")
	}
	if snap.CompileErrors == 0 {
		t.Error("VMSnapshot.CompileErrors should be >= 1")
	}
}

func TestManager_CacheHit(t *testing.T) {
	mgr := NewVMManager()
	ast := &RuleNode{
		Type: NodeTypeCondition,
		Condition: &RuleCondition{
			FieldPath: "a",
			Operator:   ">",
			Value:      float64(1),
			ValueType:  "number",
		},
	}
	mgr.RegisterAndFreeze([]*RuleNode{ast})

	fallback := func(n *RuleNode, i map[string]any) (bool, error) { return false, nil }

	// First call compiles.
	got1, err := mgr.EvaluateWithFallback("rule-cache", ast, map[string]any{"a": 2.0}, fallback)
	if err != nil {
		t.Fatal(err)
	}
	if !got1 {
		t.Error("expected true (2 > 1)")
	}

	snap1 := mgr.Snapshot()
	if snap1.CacheMisses == 0 {
		t.Error("first call should be a cache miss")
	}

	// Second call should hit cache.
	got2, err := mgr.EvaluateWithFallback("rule-cache", ast, map[string]any{"a": 0.0}, fallback)
	if err != nil {
		t.Fatal(err)
	}
	if got2 {
		t.Error("expected false (0 > 1)")
	}

	snap2 := mgr.Snapshot()
	if snap2.CacheHits == 0 {
		t.Error("second call should be a cache hit")
	}
	if snap2.CacheSize != 1 {
		t.Errorf("CacheSize = %d, want 1", snap2.CacheSize)
	}
}

func TestManager_StickyUnsupported(t *testing.T) {
	mgr := NewVMManager()
	ast := &RuleNode{
		Type: NodeTypeCondition,
		Condition: &RuleCondition{
			FieldPath: "a",
			Operator:   "matches_regex",
			Value:      ".*",
			ValueType:  "string",
		},
	}
	mgr.RegisterAndFreeze([]*RuleNode{ast})

	fallback := func(n *RuleNode, i map[string]any) (bool, error) { return false, nil }

	// First call: compile fails -> fallback.
	_, _ = mgr.EvaluateWithFallback("rule-regex", ast, map[string]any{"a": "x"}, fallback)
	snap1 := mgr.Snapshot()

	// Second call: must hit cache (sticky unsupported), not re-compile.
	_, _ = mgr.EvaluateWithFallback("rule-regex", ast, map[string]any{"a": "y"}, fallback)
	snap2 := mgr.Snapshot()

	if snap2.CacheMisses != snap1.CacheMisses {
		t.Errorf("expected no additional miss after sticky unsupported; got %d (was %d)",
			snap2.CacheMisses, snap1.CacheMisses)
	}
	if snap2.Fallbacks != snap1.Fallbacks+1 {
		t.Errorf("expected fallback count to increment by 1; got %d (was %d)",
			snap2.Fallbacks, snap1.Fallbacks)
	}
}

func TestManager_ParityWithRecursive(t *testing.T) {
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

	fallback := func(n *RuleNode, i map[string]any) (bool, error) {
		ae := NewAdvancedEvaluator()
		return ae.Evaluate(*n, i)
	}

	inputs := []map[string]any{
		{"customer": map[string]any{"tier": "GOLD", "balance": float64(15000)}},
		{"customer": map[string]any{"tier": "SILVER", "balance": float64(15000)}},
		{"customer": map[string]any{"tier": "GOLD", "balance": float64(5000)}},
		{"customer": map[string]any{"tier": "GOLD", "balance": float64(10000)}},
	}

	for i, input := range inputs {
		vmGot, err := mgr.EvaluateWithFallback("parity", ast, input, fallback)
		if err != nil {
			t.Fatalf("iter %d: vm err: %v", i, err)
		}
		recGot, err := fallback(ast, input)
		if err != nil {
			t.Fatalf("iter %d: recursive err: %v", i, err)
		}
		if vmGot != recGot {
			t.Errorf("iter %d: parity mismatch vm=%v recursive=%v", i, vmGot, recGot)
		}
	}
}

func TestManager_EvaluateJSON_Parity(t *testing.T) {
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

	fallback := func(n *RuleNode, i map[string]any) (bool, error) {
		ae := NewAdvancedEvaluator()
		return ae.Evaluate(*n, i)
	}

	cases := []struct {
		name string
		json []byte
		want bool
	}{
		{"GOLD+15000 passes", []byte(`{"customer":{"tier":"GOLD","balance":15000}}`), true},
		{"SILVER fails tier", []byte(`{"customer":{"tier":"SILVER","balance":15000}}`), false},
		{"GOLD+5000 fails balance", []byte(`{"customer":{"tier":"GOLD","balance":5000}}`), false},
		{"GOLD+10000 boundary", []byte(`{"customer":{"tier":"GOLD","balance":10000}}`), false}, // > not >=
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mgr.EvaluateJSONWithFallback("json-parity", ast, tc.json, fallback)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("got=%v want=%v", got, tc.want)
			}
		})
	}
}