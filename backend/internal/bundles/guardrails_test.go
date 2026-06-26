package bundles

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestEvaluateGuardrails_YAMLFallback(t *testing.T) {
	// create a temporary guardrails.yaml in working dir
	yaml := `sod_pairs:
  - ["orders_write", "billing_view"]
  - ["orders_write", "customer_segment"]
certified:
  - "billing_view"
`
	fpath := "guardrails.yaml"
	if err := os.WriteFile(fpath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write yaml: %v", err)
	}
	defer os.Remove(fpath)

	// build a proposal details payload with conflicting claims
	details := map[string]interface{}{
		"claims":      []string{"orders_write", "billing_view"},
		"description": "test",
	}
	var db *sqlx.DB = nil
	b, _ := json.Marshal(details)
	ok, reasons, err := evaluateGuardrails(db, b)
	if err != nil {
		t.Fatalf("evaluateGuardrails returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected guardrail to fail but passed")
	}
	if len(reasons) == 0 {
		t.Fatalf("expected reasons for failure")
	}
}

func TestLoadGuardrails_NoFile(t *testing.T) {
	// Ensure no YAML exists
	_ = os.Remove("guardrails.yaml")
	cfg, _, err := loadGuardrails(nil)
	if err != nil {
		t.Fatalf("loadGuardrails error: %v", err)
	}
	// default config should be empty
	if cfg == nil {
		t.Fatalf("expected non-nil cfg")
	}
	if len(cfg.SoDPairs) != 0 || len(cfg.Certified) != 0 {
		t.Fatalf("expected empty guardrails when no config present")
	}
}
