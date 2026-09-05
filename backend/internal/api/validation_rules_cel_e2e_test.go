package api

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/hondyman/uisce/backend/internal/rulefabric"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// TestEvaluateValidationRuleCEL_EndToEnd exercises the code path that
// replaced CUE/Starlark execution (fetchTenantRuleForExecution +
// evaluateValidationRuleCEL) against a real Postgres instance running the
// actual catalog_validation_rules/catalog_validation_rule_cores schema, for
// all three inheritance modes: custom, inherit, and extend.
//
// Skipped unless VALIDATION_RULES_TEST_DSN is set, since it needs a real
// database with these tables (this repo doesn't run migrations in CI for
// this package). See the schema in backend/migrations/misc/create_validation_rules.sql
// + backend/migrations/20251220_core_validation_rule_templates.sql +
// backend/migrations/20260206_120100_drop_starlark_column.sql.
func TestEvaluateValidationRuleCEL_EndToEnd(t *testing.T) {
	dsn := os.Getenv("VALIDATION_RULES_TEST_DSN")
	if dsn == "" {
		t.Skip("VALIDATION_RULES_TEST_DSN not set; skipping end-to-end DB test")
	}

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer sqlDB.Close()

	evaluator, err := rulefabric.NewRuleEvaluator(sqlx.NewDb(sqlDB, "postgres"))
	if err != nil {
		t.Fatalf("NewRuleEvaluator: %v", err)
	}
	h := &validationRulesHandler{db: sqlDB, ruleEvaluator: evaluator}

	const tenantID = "22222222-2222-2222-2222-222222222222"
	const datasourceID = "66666666-6666-6666-6666-666666666666"

	cases := []struct {
		name      string
		ruleID    string
		record    map[string]interface{}
		wantValid bool
	}{
		{"custom mode, passes", "11111111-1111-1111-1111-111111111111", map[string]interface{}{"age": 25.0}, true},
		{"custom mode, fails", "11111111-1111-1111-1111-111111111111", map[string]interface{}{"age": 10.0}, false},
		{"inherit mode, passes", "44444444-4444-4444-4444-444444444444", map[string]interface{}{"kyc_status": "VERIFIED"}, true},
		{"inherit mode, fails", "44444444-4444-4444-4444-444444444444", map[string]interface{}{"kyc_status": "PENDING"}, false},
		{"extend mode, both satisfied", "55555555-5555-5555-5555-555555555555", map[string]interface{}{"kyc_status": "VERIFIED", "age": 25.0}, true},
		{"extend mode, core ok but extension fails", "55555555-5555-5555-5555-555555555555", map[string]interface{}{"kyc_status": "VERIFIED", "age": 18.0}, false},
		{"extend mode, core fails", "55555555-5555-5555-5555-555555555555", map[string]interface{}{"kyc_status": "PENDING", "age": 25.0}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row, err := fetchTenantRuleForExecution(context.Background(), h.db, tc.ruleID, tenantID, datasourceID)
			if err != nil {
				t.Fatalf("fetchTenantRuleForExecution: %v", err)
			}
			valid, message, err := h.evaluateValidationRuleCEL(context.Background(), row, tc.record)
			if err != nil {
				t.Fatalf("evaluateValidationRuleCEL error: %v", err)
			}
			if valid != tc.wantValid {
				t.Errorf("got valid=%v (message=%q), want valid=%v", valid, message, tc.wantValid)
			}
		})
	}
}
