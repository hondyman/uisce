package services

import (
	"context"
	"fmt"
	"runtime"
	"testing"
)

func BenchmarkStarlarkEngine_EvaluateUserRule_OkStyle_CachedProgram(b *testing.B) {
	engine := NewStarlarkEngine(nil)
	script := `
# ok-style rule
ok = eq(field("account", "account_type"), "ADVISORY")
message = "Account must be ADVISORY"
`
	data := map[string]interface{}{
		"account": map[string]interface{}{
			"account_type": "ADVISORY",
		},
	}

	// Warm compile/cache.
	_, _ = engine.EvaluateUserRule(context.Background(), script, data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err := engine.EvaluateUserRule(context.Background(), script, data)
		if err != nil {
			b.Fatal(err)
		}
		if res == nil || !res.IsValid {
			b.Fatalf("unexpected result: %+v", res)
		}
	}
}

func BenchmarkStarlarkEngine_EvaluateUserRuleBatch_OkStyle(b *testing.B) {
	engine := NewStarlarkEngine(nil)
	script := `
ok = eq(field("account", "account_type"), "ADVISORY")
message = "Account must be ADVISORY"
`

	records := make([]map[string]interface{}, 1000)
	for i := 0; i < len(records); i++ {
		accountType := "BROKERAGE"
		if i%5 == 0 {
			accountType = "ADVISORY"
		}
		records[i] = map[string]interface{}{"account": map[string]interface{}{"account_type": accountType}}
	}

	// Warm compile/cache.
	_, _ = engine.EvaluateUserRuleBatch(context.Background(), script, records[:1], 1)

	workers := runtime.GOMAXPROCS(0)
	if workers <= 0 {
		workers = 1
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err := engine.EvaluateUserRuleBatch(context.Background(), script, records, workers)
		if err != nil {
			b.Fatal(err)
		}
		if len(res) != len(records) {
			b.Fatalf("len(res)=%d want %d", len(res), len(records))
		}
	}
}

func BenchmarkStarlarkEngine_EvaluateOkRuleBundleBatch(b *testing.B) {
	engine := NewStarlarkEngine(nil)

	// 20 simple rules; 1 fails for non-ADVISORY.
	rules := make([]OkRule, 0, 20)
	rules = append(rules, OkRule{ID: "r0", Script: `ok = eq(field("account", "account_type"), "ADVISORY")`})
	for i := 1; i < 20; i++ {
		rules = append(rules, OkRule{ID: fmt.Sprintf("r%d", i), Script: `ok = True`})
	}

	records := make([]map[string]interface{}, 1000)
	for i := 0; i < len(records); i++ {
		accountType := "BROKERAGE"
		if i%5 == 0 {
			accountType = "ADVISORY"
		}
		records[i] = map[string]interface{}{"account": map[string]interface{}{"account_type": accountType}}
	}

	// Warm compile/cache.
	_, _ = engine.EvaluateOkRuleBundleBatch(context.Background(), rules, records[:1], 1, false)

	workers := runtime.GOMAXPROCS(0)
	if workers <= 0 {
		workers = 1
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err := engine.EvaluateOkRuleBundleBatch(context.Background(), rules, records, workers, false)
		if err != nil {
			b.Fatal(err)
		}
		if len(res) != len(records) {
			b.Fatalf("len(res)=%d want %d", len(res), len(records))
		}
	}
}

func BenchmarkStarlarkEngine_EvaluateOkRuleBundleBatch_BigPayload(b *testing.B) {
	engine := NewStarlarkEngine(nil)

	// 20 simple rules; 1 checks account_type.
	rules := make([]OkRule, 0, 20)
	rules = append(rules, OkRule{ID: "r0", Script: `ok = eq(field("account", "account_type"), "ADVISORY")`})
	for i := 1; i < 20; i++ {
		rules = append(rules, OkRule{ID: fmt.Sprintf("r%d", i), Script: `ok = True`})
	}

	bigPositions := make([]interface{}, 0, 50)
	for i := 0; i < 50; i++ {
		bigPositions = append(bigPositions, map[string]interface{}{
			"id":   i,
			"lots": []interface{}{1, 2, 3, 4, 5, 6, 7, 8},
		})
	}

	records := make([]map[string]interface{}, 1000)
	for i := 0; i < len(records); i++ {
		accountType := "BROKERAGE"
		if i%5 == 0 {
			accountType = "ADVISORY"
		}
		records[i] = map[string]interface{}{
			"account":   map[string]interface{}{"account_type": accountType},
			"positions": bigPositions,
		}
	}

	_, _ = engine.EvaluateOkRuleBundleBatch(context.Background(), rules, records[:1], 1, false)

	workers := runtime.GOMAXPROCS(0)
	if workers <= 0 {
		workers = 1
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err := engine.EvaluateOkRuleBundleBatch(context.Background(), rules, records, workers, false)
		if err != nil {
			b.Fatal(err)
		}
		if len(res) != len(records) {
			b.Fatalf("len(res)=%d want %d", len(res), len(records))
		}
	}
}

func BenchmarkStarlarkEngine_EvaluateOkRuleBundleBatchWithMeta_Projected(b *testing.B) {
	engine := NewStarlarkEngine(nil)

	rules := make([]OkRuleWithMeta, 0, 20)
	rules = append(rules, OkRuleWithMeta{
		OkRule: OkRule{ID: "r0", Script: `ok = eq(field("account", "account_type"), "ADVISORY")`},
		Meta:   OkRuleMeta{Cost: 1, FailureLikelihood: 0.4, RequiredFieldPaths: []string{"account.account_type"}},
	})
	for i := 1; i < 20; i++ {
		rules = append(rules, OkRuleWithMeta{
			OkRule: OkRule{ID: fmt.Sprintf("r%d", i), Script: `ok = True`},
			Meta:   OkRuleMeta{Cost: 10, FailureLikelihood: 0.01},
		})
	}

	// Large unused payload that should be dropped by projection.
	bigPositions := make([]interface{}, 0, 50)
	for i := 0; i < 50; i++ {
		bigPositions = append(bigPositions, map[string]interface{}{
			"id":   i,
			"lots": []interface{}{1, 2, 3, 4, 5, 6, 7, 8},
		})
	}

	records := make([]map[string]interface{}, 1000)
	for i := 0; i < len(records); i++ {
		accountType := "BROKERAGE"
		if i%5 == 0 {
			accountType = "ADVISORY"
		}
		records[i] = map[string]interface{}{
			"account":   map[string]interface{}{"account_type": accountType},
			"positions": bigPositions,
		}
	}

	_, _ = engine.EvaluateOkRuleBundleBatchWithMeta(context.Background(), rules, records[:1], 1, false)

	workers := runtime.GOMAXPROCS(0)
	if workers <= 0 {
		workers = 1
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err := engine.EvaluateOkRuleBundleBatchWithMeta(context.Background(), rules, records, workers, false)
		if err != nil {
			b.Fatal(err)
		}
		if len(res) != len(records) {
			b.Fatalf("len(res)=%d want %d", len(res), len(records))
		}
	}
}
