package services

import (
	"context"
	"testing"
)

func TestStarlarkEngine_EvaluateUserRuleBatch_OkStyle_OrderPreserved(t *testing.T) {
	engine := NewStarlarkEngine(nil)
	script := `
ok = eq(field("account", "account_type"), "ADVISORY")
message = "Account must be ADVISORY"
`

	records := []map[string]interface{}{
		{"account": map[string]interface{}{"account_type": "BROKERAGE"}},
		{"account": map[string]interface{}{"account_type": "ADVISORY"}},
		{"account": map[string]interface{}{"account_type": "IRA"}},
	}

	results, err := engine.EvaluateUserRuleBatch(context.Background(), script, records, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != len(records) {
		t.Fatalf("len(results)=%d want %d", len(results), len(records))
	}

	if results[0] == nil || results[0].IsValid {
		t.Fatalf("results[0]=%+v want invalid", results[0])
	}
	if results[1] == nil || !results[1].IsValid {
		t.Fatalf("results[1]=%+v want valid", results[1])
	}
	if results[2] == nil || results[2].IsValid {
		t.Fatalf("results[2]=%+v want invalid", results[2])
	}

	// Message should be present when provided.
	if results[0].Message == "" {
		t.Fatalf("results[0].Message empty; want non-empty")
	}
}

func TestStarlarkEngine_EvaluateOkRuleBundleBatch_ShortCircuit(t *testing.T) {
	engine := NewStarlarkEngine(nil)

	rules := []OkRule{
		{ID: "r1", Script: `ok = eq(field("account", "account_type"), "ADVISORY")`},
		{ID: "r2", Script: `ok = True`},
		{ID: "r3", Script: `ok = True`},
	}

	records := []map[string]interface{}{
		{"account": map[string]interface{}{"account_type": "BROKERAGE"}},
	}

	results, err := engine.EvaluateOkRuleBundleBatch(context.Background(), rules, records, 1, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results)=%d want 1", len(results))
	}
	if len(results[0]) != len(rules) {
		t.Fatalf("len(results[0])=%d want %d", len(results[0]), len(rules))
	}

	if results[0][0] == nil || results[0][0].IsValid {
		t.Fatalf("results[0][0]=%+v want invalid", results[0][0])
	}
	if results[0][1] != nil || results[0][2] != nil {
		t.Fatalf("expected short-circuited results to be nil, got r2=%+v r3=%+v", results[0][1], results[0][2])
	}
}

func TestStarlarkEngine_EvaluateOkRuleBundleBatchWithMeta_OrdersByCostAndShortCircuits(t *testing.T) {
	engine := NewStarlarkEngine(nil)

	// Both rules fail; ordering decides which result becomes non-nil under short-circuit.
	rules := []OkRuleWithMeta{
		{
			OkRule: OkRule{ID: "expensive", Script: `ok = False`},
			Meta:   OkRuleMeta{Cost: 10, FailureLikelihood: 0.9, RequiredFieldPaths: []string{"account.account_type"}},
		},
		{
			OkRule: OkRule{ID: "cheap", Script: `ok = False`},
			Meta:   OkRuleMeta{Cost: 1, FailureLikelihood: 0.1, RequiredFieldPaths: []string{"account.account_type"}},
		},
	}

	records := []map[string]interface{}{
		{
			"account": map[string]interface{}{"account_type": "BROKERAGE"},
			// Big unused payload that should be dropped by projection.
			"positions": []interface{}{map[string]interface{}{"id": 1, "lots": []interface{}{1, 2, 3}}},
		},
	}

	results, err := engine.EvaluateOkRuleBundleBatchWithMeta(context.Background(), rules, records, 1, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0][1] == nil || results[0][1].IsValid {
		t.Fatalf("expected cheap rule (index 1) to run first and fail; got %+v", results[0][1])
	}
	if results[0][0] != nil {
		t.Fatalf("expected expensive rule (index 0) to be skipped; got %+v", results[0][0])
	}
}
