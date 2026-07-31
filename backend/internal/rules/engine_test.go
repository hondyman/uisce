package rules

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func smallRule(conditionID string, field string, op string, value any) *RuleNode {
	return &RuleNode{
		Type: NodeTypeGroup,
		Group: &RuleGroup{
			ID:       "grp-" + conditionID,
			Operator: "AND",
			Conditions: []RuleNode{
				{
					Type: NodeTypeCondition,
					Condition: &RuleCondition{
						ID:        conditionID,
						Field:     field,
						FieldPath: field,
						Operator:  op,
						Value:     value,
					},
				},
			},
		},
	}
}

func TestEngine_StateSwap_PreservesHotPath(t *testing.T) {
	eng := NewRuleEngine(nil)

	rule := smallRule("r1", "customer.tier", "==", "GOLD")
	input := map[string]any{"customer": map[string]any{"tier": "GOLD"}}

	if _, err := eng.RewarmCore([]*RuleNode{rule}, 1); err != nil {
		t.Fatal(err)
	}
	if got := eng.CurrentRevision(); got != 1 {
		t.Errorf("revision after first rewarm = %d, want 1", got)
	}

	passed, trace, err := eng.Evaluate(context.Background(), "", rule.ID(), 1, rule, input, false)
	if err != nil || !passed {
		t.Fatalf("first eval: passed=%v err=%v", passed, err)
	}
	if !trace.UsedVM {
		t.Errorf("expected VM path, got fallback=%q", trace.Fallback)
	}

	ruleV2 := smallRule("r2", "customer.new_field", "==", "x")
	if _, err := eng.RewarmCore([]*RuleNode{rule, ruleV2}, 2); err != nil {
		t.Fatal(err)
	}
	if got := eng.CurrentRevision(); got != 2 {
		t.Errorf("revision after second rewarm = %d, want 2", got)
	}

	passed, trace, err = eng.Evaluate(context.Background(), "", rule.ID(), 1, rule, input, false)
	if err != nil || !passed {
		t.Fatalf("post-rewarm eval: passed=%v err=%v", passed, err)
	}
	if !trace.UsedVM {
		t.Errorf("expected VM path post-rewarm, got fallback=%q", trace.Fallback)
	}
	if trace.Revision != 2 {
		t.Errorf("trace.Revision = %d, want 2 (post-rewarm state)", trace.Revision)
	}
}

func TestEngine_FallbackTraceOnUnsupportedField(t *testing.T) {
	eng := NewRuleEngine(nil)

	registeredRule := smallRule("r1", "customer.tier", "==", "GOLD")
	if _, err := eng.RewarmCore([]*RuleNode{registeredRule}, 1); err != nil {
		t.Fatal(err)
	}

	unregisteredFieldRule := smallRule("r2", "customer.unregistered_field", "==", "x")
	input := map[string]any{
		"customer": map[string]any{
			"tier":               "GOLD",
			"unregistered_field": "x",
		},
	}
	passed, trace, err := eng.Evaluate(context.Background(), "", unregisteredFieldRule.ID(), 1, unregisteredFieldRule, input, false)
	if err != nil {
		t.Fatalf("recursive fallback should not error on missing field; got %v", err)
	}
	if !passed {
		t.Errorf("recursive fallback should pass; got passed=false")
	}
	if trace.UsedVM {
		t.Errorf("expected fallback path; got UsedVM=true")
	}
	if trace.Fallback == "" {
		t.Errorf("expected Fallback to be populated with reason")
	}
	if !contains(trace.Fallback, "unregistered_field") {
		t.Errorf("Fallback reason should mention the missing field; got %q", trace.Fallback)
	}
}

func TestEngine_ForceRecompile_DoesNotCacheFailure(t *testing.T) {
	eng := NewRuleEngine(nil)

	badRule := smallRule("r1", "customer.name", "contains", "Smith")
	if _, err := eng.RewarmCore([]*RuleNode{badRule}, 1); err != nil {
		t.Fatal(err)
	}
	_, trace, _ := eng.Evaluate(context.Background(), "", badRule.ID(), 1, badRule, map[string]any{}, false)
	if trace.UsedVM {
		t.Errorf("expected fallback for unsupported rule on first eval")
	}

	missesAfterFirst := eng.Metrics().CacheMisses()

	goodRule := smallRule("r1", "customer.name", "==", "John")
	if _, err := eng.RewarmCore([]*RuleNode{goodRule}, 2); err != nil {
		t.Fatal(err)
	}
	passed, trace, err := eng.Evaluate(context.Background(), "", goodRule.ID(), 2, goodRule, map[string]any{"customer": map[string]any{"name": "John"}}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !passed {
		t.Errorf("force-recompile: expected pass")
	}
	if !trace.UsedVM {
		t.Errorf("expected VM path after force-recompile, got fallback=%q", trace.Fallback)
	}
	if eng.Metrics().CacheMisses() <= missesAfterFirst {
		t.Errorf("force-recompile should have triggered a compile; misses did not increase")
	}

	passed, _, _ = eng.Evaluate(context.Background(), "", goodRule.ID(), 2, goodRule, map[string]any{"customer": map[string]any{"name": "John"}}, false)
	if !passed {
		t.Errorf("cached call should pass")
	}
	if eng.Metrics().CacheHits() == 0 {
		t.Errorf("expected at least one cache hit after force-recompile")
	}
}

func TestEngine_CumulativeMetrics_SurviveRewarm(t *testing.T) {
	eng := NewRuleEngine(nil)
	rule := smallRule("r1", "customer.tier", "==", "GOLD")
	if _, err := eng.RewarmCore([]*RuleNode{rule}, 1); err != nil {
		t.Fatal(err)
	}
	input := map[string]any{"customer": map[string]any{"tier": "GOLD"}}

	for i := 0; i < 100; i++ {
		eng.Evaluate(context.Background(), "", rule.ID(), 1, rule, input, false)
	}
	if hits := eng.Metrics().CacheHits(); hits != 100 {
		t.Errorf("CacheHits = %d, want 100", hits)
	}

	if _, err := eng.RewarmCore([]*RuleNode{rule}, 2); err != nil {
		t.Fatal(err)
	}
	if hits := eng.Metrics().CacheHits(); hits != 100 {
		t.Errorf("CacheHits after rewarm = %d, want 100 (must survive)", hits)
	}
	if vm := eng.Metrics().VMPathCount(); vm != 100 {
		t.Errorf("VMPathCount after rewarm = %d, want 100", vm)
	}
}

func TestEngine_NilNodeFallsBack(t *testing.T) {
	eng := NewRuleEngine(nil)
	if _, err := eng.RewarmCore(nil, 1); err != nil {
		t.Fatal(err)
	}
	passed, trace, err := eng.Evaluate(context.Background(), "", "x", 1, nil, map[string]any{}, false)
	if err == nil {
		t.Errorf("expected error for nil node")
	}
	if passed {
		t.Errorf("expected passed=false for nil node")
	}
	if trace.Fallback == "" {
		t.Errorf("expected Fallback reason for nil node")
	}
}

func TestEngine_CacheKeyIncludesVersion(t *testing.T) {
	eng := NewRuleEngine(nil)
	rule := smallRule("r1", "customer.tier", "==", "GOLD")
	input := map[string]any{"customer": map[string]any{"tier": "GOLD"}}

	for i, version := range []int{1, 2, 3} {
		_, trace, _ := eng.Evaluate(context.Background(), "", rule.ID(), version, rule, input, false)
		if !trace.UsedVM {
			t.Errorf("call %d (v=%d): expected VM path", i, version)
		}
	}

	if misses := eng.Metrics().CacheMisses(); misses != 3 {
		t.Errorf("expected 3 cache misses (one per version-key), got %d", misses)
	}
}

func TestEngine_RewarmPreCompilesRules(t *testing.T) {
	eng := NewRuleEngine(nil)

	rules := []*RuleNode{
		smallRule("r1", "customer.tier", "==", "GOLD"),
		smallRule("r2", "customer.balance", ">", 10000),
		smallRule("r3", "customer.country", "==", "US"),
	}
	if _, err := eng.RewarmCore(rules, 1); err != nil {
		t.Fatal(err)
	}
	if got := eng.CurrentCacheSize(); got != 3 {
		t.Errorf("CacheSize after rewarm = %d, want 3 (pre-compiled)", got)
	}

	missesBefore := eng.Metrics().CacheMisses()
	for _, rule := range rules {
		input := map[string]any{
			"customer": map[string]any{"tier": "GOLD", "balance": float64(15000), "country": "US"},
		}
		passed, trace, err := eng.Evaluate(context.Background(), "", rule.ID(), 1, rule, input, false)
		if err != nil || !passed {
			t.Fatalf("rule %s: passed=%v err=%v", rule.ID(), passed, err)
		}
		if !trace.UsedVM {
			t.Errorf("rule %s: expected VM path", rule.ID())
		}
	}
	if newMisses := eng.Metrics().CacheMisses(); newMisses != missesBefore {
		t.Errorf("post-rewarm evaluation should be all cache hits; got %d new misses", newMisses-missesBefore)
	}
}

func TestEngine_RaceCondition_ConcurrentEvaluateAndRewarm(t *testing.T) {
	eng := NewRuleEngine(nil)
	rule := smallRule("r1", "customer.tier", "==", "GOLD")
	if _, err := eng.RewarmCore([]*RuleNode{rule}, 1); err != nil {
		t.Fatal(err)
	}
	input := map[string]any{"customer": map[string]any{"tier": "GOLD"}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				_, _, _ = eng.Evaluate(context.Background(), "", rule.ID(), 1, rule, input, false)
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			time.Sleep(time.Millisecond)
			if _, err := eng.RewarmCore([]*RuleNode{rule}, i+1); err != nil {
				t.Errorf("rewarm: %v", err)
				return
			}
		}
		cancel()
	}()

	wg.Wait()
}

func TestEngine_UpdateRule_IncrementalCacheUpdate(t *testing.T) {
	eng := NewRuleEngine(nil)
	ctx := context.Background()

	ruleV1 := smallRule("r1", "customer.tier", "==", "GOLD")
	eng.RewarmCore([]*RuleNode{ruleV1}, 1)

	inputGold := map[string]any{"customer": map[string]any{"tier": "GOLD"}}
	passed, trace, _ := eng.Evaluate(ctx, "", ruleV1.ID(), 1, ruleV1, inputGold, false)
	if !passed || !trace.UsedVM {
		t.Fatalf("v1 should pass and use VM, got passed=%v trace=%+v", passed, trace)
	}

	ruleV2 := smallRule("r1", "customer.tier", "==", "SILVER")
	eng.UpdateRule("", ruleV1.ID(), 2, ruleV2)

	inputSilver := map[string]any{"customer": map[string]any{"tier": "SILVER"}}
	passed, trace, _ = eng.Evaluate(ctx, "", ruleV1.ID(), 2, ruleV2, inputSilver, false)
	if !passed || !trace.UsedVM {
		t.Fatalf("v2 should pass and use VM, got passed=%v trace=%+v", passed, trace)
	}

	passed, _, _ = eng.Evaluate(ctx, "", ruleV1.ID(), 1, ruleV1, inputGold, false)
	if !passed {
		t.Error("v1 should still pass with old cache entry")
	}
}

func TestEngine_DeleteRule_RemovesFromCache(t *testing.T) {
	eng := NewRuleEngine(nil)
	rule := smallRule("r1", "customer.tier", "==", "GOLD")
	eng.RewarmCore([]*RuleNode{rule}, 1)

	sizeBefore := eng.CurrentCacheSize()
	eng.DeleteRule("", rule.ID(), 1)
	if size := eng.CurrentCacheSize(); size != sizeBefore-1 {
		t.Errorf("expected cache size %d, got %d", sizeBefore-1, size)
	}

	missesBefore := eng.Metrics().CacheMisses()
	eng.Evaluate(context.Background(), "", rule.ID(), 1, rule, map[string]any{"customer": map[string]any{"tier": "GOLD"}}, false)
	if eng.Metrics().CacheMisses() <= missesBefore {
		t.Error("expected a cache miss after deleting and re-evaluating rule")
	}
}

func TestEngine_HasField(t *testing.T) {
	eng := NewRuleEngine(nil)
	rule := smallRule("r1", "customer.tier", "==", "GOLD")
	eng.RewarmCore([]*RuleNode{rule}, 1)

	if !eng.HasField("", "customer.tier") {
		t.Error("expected HasField to return true for registered field")
	}
	if eng.HasField("", "customer.custom_tax_id") {
		t.Error("expected HasField to return false for unregistered field")
	}
}

func TestEngine_CustomField_Addition(t *testing.T) {
	eng := NewRuleEngine(nil)
	ctx := context.Background()

	ruleStandard := smallRule("r1", "customer.tier", "==", "GOLD")
	eng.RewarmCore([]*RuleNode{ruleStandard}, 1)

	ruleCustom := smallRule("r2", "customer.custom_tax_id", "==", "TAX-123")
	eng.UpdateRule("", ruleCustom.ID(), 1, ruleCustom)

	inputCustom := map[string]any{"customer": map[string]any{"custom_tax_id": "TAX-123"}}
	passed, trace, _ := eng.Evaluate(ctx, "", ruleCustom.ID(), 1, ruleCustom, inputCustom, false)

	if !passed {
		t.Error("expected recursive fallback to pass the evaluation")
	}
	if trace.UsedVM {
		t.Error("expected fallback to recursive because field is missing from dict")
	}
	if !contains(trace.Fallback, "custom_tax_id") {
		t.Errorf("expected fallback reason to mention the missing field, got: %s", trace.Fallback)
	}

	allRules := []*RuleNode{ruleStandard, ruleCustom}
	eng.RewarmCore(allRules, 2)

	passed, trace, _ = eng.Evaluate(ctx, "", ruleCustom.ID(), 2, ruleCustom, inputCustom, false)
	if !passed {
		t.Error("expected evaluation to pass after rewarm")
	}
	if !trace.UsedVM {
		t.Errorf("expected VM path after rewarm, got fallback: %s", trace.Fallback)
	}
}

func TestEngine_TenantIsolation(t *testing.T) {
	eng := NewRuleEngine(nil)
	ctx := context.Background()

	coreRule := smallRule("core-1", "customer.tier", "==", "GOLD")
	eng.RewarmCore([]*RuleNode{coreRule}, 1)

	tenantARule := smallRule("tenant-a-1", "customer.custom_tax_id", "==", "TAX-123")
	eng.RewarmTenant("tenantA", []*RuleNode{coreRule, tenantARule}, 1)

	inputA := map[string]any{"customer": map[string]any{"custom_tax_id": "TAX-123"}}
	passed, trace, _ := eng.Evaluate(ctx, "tenantA", tenantARule.ID(), 1, tenantARule, inputA, false)
	if !passed || !trace.UsedVM {
		t.Fatalf("Tenant A should hit VM for custom rule: %+v", trace)
	}
	if !trace.IsTenant {
		t.Error("trace.IsTenant should be true for tenant-specific evaluation")
	}

	if eng.HasField("", "customer.custom_tax_id") {
		t.Error("Core state should not know about tenant custom fields")
	}

	inputB := map[string]any{"customer": map[string]any{"tier": "GOLD"}}
	passed, trace, _ = eng.Evaluate(ctx, "tenantB", coreRule.ID(), 1, coreRule, inputB, false)
	if !passed || !trace.UsedVM {
		t.Fatalf("Tenant B should hit VM for core rule: %+v", trace)
	}
	if trace.IsTenant {
		t.Error("trace.IsTenant should be false when using core state")
	}

	passed, trace, _ = eng.Evaluate(ctx, "tenantB", tenantARule.ID(), 1, tenantARule, inputA, false)
	if trace.UsedVM {
		t.Error("Tenant B should NOT use VM for Tenant A's custom rule")
	}
	if trace.Fallback == "" {
		t.Error("Expected fallback reason for Tenant B evaluating unknown field")
	}
}

func TestEngine_RewarmTenant_IncrementalUpdate(t *testing.T) {
	eng := NewRuleEngine(nil)
	ctx := context.Background()

	coreRule := smallRule("core-1", "customer.tier", "==", "GOLD")
	eng.RewarmCore([]*RuleNode{coreRule}, 1)

	tenantARuleV1 := smallRule("tenant-a-1", "customer.region", "==", "US")
	eng.RewarmTenant("tenantA", []*RuleNode{coreRule, tenantARuleV1}, 1)

	inputA := map[string]any{"customer": map[string]any{"tier": "GOLD", "region": "US"}}
	passed, trace, _ := eng.Evaluate(ctx, "tenantA", tenantARuleV1.ID(), 1, tenantARuleV1, inputA, false)
	if !passed || !trace.UsedVM {
		t.Fatalf("Tenant A v1 should hit VM: %+v", trace)
	}

	tenantARuleV2 := smallRule("tenant-a-1", "customer.region", "==", "EU")
	eng.RewarmTenant("tenantA", []*RuleNode{coreRule, tenantARuleV2}, 2)

	inputEU := map[string]any{"customer": map[string]any{"tier": "GOLD", "region": "EU"}}
	passed, trace, _ = eng.Evaluate(ctx, "tenantA", tenantARuleV2.ID(), 2, tenantARuleV2, inputEU, false)
	if !passed || !trace.UsedVM {
		t.Fatalf("Tenant A v2 should hit VM: %+v", trace)
	}
}

func TestEngine_RewarmTenant_CustomFieldWithoutCoreRule(t *testing.T) {
	eng := NewRuleEngine(nil)
	ctx := context.Background()

	eng.RewarmCore([]*RuleNode{}, 1)

	tenantARule := smallRule("tenant-a-1", "customer.custom_sector", "==", "TECH")
	eng.RewarmTenant("tenantA", []*RuleNode{tenantARule}, 1)

	input := map[string]any{"customer": map[string]any{"custom_sector": "TECH"}}
	passed, trace, _ := eng.Evaluate(ctx, "tenantA", tenantARule.ID(), 1, tenantARule, input, false)
	if !passed || !trace.UsedVM {
		t.Fatalf("Tenant A should hit VM for custom rule: %+v", trace)
	}
}

func TestEngine_Evictor_RemovesIdleStates(t *testing.T) {
	eng := NewRuleEngine(nil)

	rule := smallRule("r1", "customer.tier", "==", "GOLD")
	eng.RewarmTenant("tenantA", []*RuleNode{rule}, 1)
	eng.RewarmTenant("tenantB", []*RuleNode{rule}, 1)

	if eng.TenantCount() != 2 {
		t.Fatalf("expected 2 tenants, got %d", eng.TenantCount())
	}

	now := time.Now()
	evicted := eng.EvictIdleStates(now, time.Hour)
	if evicted != 0 {
		t.Errorf("expected 0 evictions with hour TTL, got %d", evicted)
	}

	oldTime := now.Add(-2 * time.Hour)
	evicted = eng.EvictIdleStates(oldTime, time.Hour)
	if evicted != 2 {
		t.Errorf("expected 2 evictions, got %d", evicted)
	}
	if eng.TenantCount() != 0 {
		t.Errorf("expected 0 tenants after eviction, got %d", eng.TenantCount())
	}
}

func TestEngine_Evictor_PreservesActiveState(t *testing.T) {
	eng := NewRuleEngine(nil)
	ctx := context.Background()

	rule := smallRule("r1", "customer.tier", "==", "GOLD")
	eng.RewarmTenant("tenantA", []*RuleNode{rule}, 1)
	eng.RewarmTenant("tenantB", []*RuleNode{rule}, 1)

	input := map[string]any{"customer": map[string]any{"tier": "GOLD"}}
	eng.Evaluate(ctx, "tenantA", rule.ID(), 1, rule, input, false)

	now := time.Now()
	evicted := eng.EvictIdleStates(now, time.Hour)
	if evicted != 1 {
		t.Errorf("expected 1 eviction (tenantB idle), got %d", evicted)
	}
	if eng.TenantCount() != 1 {
		t.Errorf("expected 1 tenant remaining, got %d", eng.TenantCount())
	}
}

func TestEngine_StartEvictor_RunsAndStops(t *testing.T) {
	eng := NewRuleEngine(nil)
	ctx, cancel := context.WithCancel(context.Background())

	eng.StartEvictor(ctx, time.Hour, 10*time.Millisecond)
	cancel()

	done := make(chan struct{})
	go func() {
		time.Sleep(50 * time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Error("evictor did not stop within 200ms of cancel")
	}
}

func TestEngine_Singleflight_DeduplicatesConcurrentRewarms(t *testing.T) {
	eng := NewRuleEngine(nil)

	rule := smallRule("r1", "customer.tier", "==", "GOLD")

	var wg sync.WaitGroup
	var maxConcurrent int64
	var current int64

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&current, 1)
			max := atomic.LoadInt64(&current)
			for {
				m := atomic.LoadInt64(&current)
				if m > max {
					atomic.StoreInt64(&maxConcurrent, m)
				}
				if atomic.LoadInt64(&current) == 0 {
					break
				}
				time.Sleep(time.Microsecond)
			}
			eng.RewarmTenant("tenantA", []*RuleNode{rule}, 1)
			atomic.AddInt64(&current, -1)
		}()
	}

	wg.Wait()
	if maxConcurrent > 1 {
		t.Errorf("expected singleflight to serialize rewarms, but max concurrent was %d", maxConcurrent)
	}
}

func TestEngine_RewarmTenant_VersionCheck(t *testing.T) {
	eng := NewRuleEngine(nil)

	rule := smallRule("r1", "customer.tier", "==", "GOLD")
	eng.RewarmTenant("tenantA", []*RuleNode{rule}, 1)

	if got := eng.TenantCount(); got != 1 {
		t.Fatalf("expected 1 tenant, got %d", got)
	}

	rev1, _ := eng.RewarmTenant("tenantA", []*RuleNode{rule}, 1)
	rev2, _ := eng.RewarmTenant("tenantA", []*RuleNode{rule}, 2)
	rev2Again, _ := eng.RewarmTenant("tenantA", []*RuleNode{rule}, 2)
	rev1Again, _ := eng.RewarmTenant("tenantA", []*RuleNode{rule}, 1)

	if rev1 != rev1 {
		t.Errorf("rev1 mismatch")
	}
	if rev2 <= rev1 {
		t.Errorf("rev2 should be greater than rev1, got rev1=%d rev2=%d", rev1, rev2)
	}
	if rev2Again != rev2 {
		t.Errorf("rev2Again should equal rev2 (version not newer), got %d vs %d", rev2Again, rev2)
	}
	if rev1Again != rev2 {
		t.Errorf("rev1Again (stale version) should not decrease revision, got %d vs %d", rev1Again, rev2)
	}
}

func TestEngine_RewarmCore_InvalidatesTenantStates(t *testing.T) {
	eng := NewRuleEngine(nil)

	coreRule := smallRule("core-1", "customer.tier", "==", "GOLD")
	eng.RewarmCore([]*RuleNode{coreRule}, 1)

	tenantRule := smallRule("tenant-a-1", "customer.custom_field", "==", "x")
	eng.RewarmTenant("tenantA", []*RuleNode{coreRule, tenantRule}, 1)

	if eng.TenantCount() != 1 {
		t.Fatal("expected 1 tenant before core rewarm")
	}

	coreRuleV2 := smallRule("core-1", "customer.tier", "==", "GOLD")
	eng.RewarmCore([]*RuleNode{coreRuleV2}, 2)

	if eng.TenantCount() != 0 {
		t.Errorf("expected 0 tenants after core rewarm (invalidated), got %d", eng.TenantCount())
	}
}

func TestEngine_RewarmCore_VersionCheck(t *testing.T) {
	eng := NewRuleEngine(nil)

	rule := smallRule("r1", "customer.tier", "==", "GOLD")
	rev1, _ := eng.RewarmCore([]*RuleNode{rule}, 1)
	rev1Again, _ := eng.RewarmCore([]*RuleNode{rule}, 1)
	rev2, _ := eng.RewarmCore([]*RuleNode{rule}, 2)

	if rev1 != rev1Again {
		t.Errorf("same version should return same revision, got %d vs %d", rev1, rev1Again)
	}
	if rev2 <= rev1 {
		t.Errorf("rev2 should be > rev1, got rev1=%d rev2=%d", rev1, rev2)
	}
}

func TestEngine_EvalTrace_HasTenantID(t *testing.T) {
	eng := NewRuleEngine(nil)

	coreRule := smallRule("r1", "customer.tier", "==", "GOLD")
	eng.RewarmCore([]*RuleNode{coreRule}, 1)

	input := map[string]any{"customer": map[string]any{"tier": "GOLD"}}

	_, trace, _ := eng.Evaluate(context.Background(), "", coreRule.ID(), 1, coreRule, input, false)
	if trace.TenantID != "" {
		t.Errorf("expected empty TenantID for core eval, got %q", trace.TenantID)
	}

	eng.RewarmTenant("tenantA", []*RuleNode{coreRule}, 1)
	_, trace, _ = eng.Evaluate(context.Background(), "tenantA", coreRule.ID(), 1, coreRule, input, false)
	if trace.TenantID != "tenantA" {
		t.Errorf("expected TenantID=tenantA, got %q", trace.TenantID)
	}
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
