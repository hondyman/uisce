package services

import (
	"context"
	"testing"
)

func TestStarlarkEngine_EvaluateUserRule_OkStyle(t *testing.T) {
	engine := NewStarlarkEngine(nil)

	script := `
ok = eq(field("account", "account_type"), "ADVISORY") and gt(num_field("account", "aum"), 100)
message = "check passed"
`

	data := map[string]interface{}{
		"account": map[string]interface{}{
			"account_type": "ADVISORY",
			"aum":          150,
		},
	}

	res, err := engine.EvaluateUserRule(context.Background(), script, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatalf("expected non-nil result")
	}
	if !res.IsValid {
		t.Fatalf("expected IsValid=true, got false (message=%q)", res.Message)
	}
	if res.Message != "check passed" {
		t.Fatalf("expected message %q, got %q", "check passed", res.Message)
	}
}
