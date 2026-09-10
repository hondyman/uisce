// One-off verification: authors the "Filled Quantity Within Bounds" rule
// (filled_qty >= 0 AND filled_qty <= quantity) against the Order BO,
// persists it as a validation_rule catalog node via the real
// ValidationRuleService (the same path the routed editor's Save action
// will use), evaluates it against every live oms.orders row via the real
// unified engine (internal/rules/vm.AdvancedEvaluator), and compares each
// result against oms.orders' live CHECK constraint chk_orders_filled_qty
// (CHECK ((filled_qty <= quantity) AND (filled_qty >= 0))) - the ground
// truth oracle.
//
// Note: the originally proposed oracle (exec_price > 0 on Execution) does
// not exist as a live CHECK constraint - only execution_exec_price_not_null
// (NOT NULL) does; grepped every migration file, found no trace of a
// "> 0" check ever being defined. oms.orders.chk_orders_filled_qty is a
// real, live, richer constraint (two conditions, one cross-field) found
// by querying pg_constraint directly instead of trusting the claim.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const tenantID = "99e99e99-99e9-49e9-89e9-99e99e99e999"

func main() {
	dsn := os.Getenv("DATABASE_URL")
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close()

	svc := analytics.NewValidationRuleService(db)
	ctx := context.Background()

	ruleAST := json.RawMessage(`{
		"type": "group",
		"operator": "AND",
		"conditions": [
			{"type": "condition", "field": "filled_qty", "operator": "greater_equal", "value": 0},
			{"type": "expression", "root": {"op": "<=", "left": {"path": "filled_qty"}, "right": {"path": "quantity"}}}
		]
	}`)

	desc, err := svc.UpsertValidationRule(ctx, models.UpsertValidationRuleRequest{
		TenantID:    tenantID,
		BOName:      "order",
		Name:        "Filled Quantity Within Bounds",
		Description: "filled_qty >= 0 AND filled_qty <= quantity - oracle rule mirroring the live DB CHECK chk_orders_filled_qty",
		Severity:    models.ValidationRuleSeverityBlock,
		Timing:      models.ValidationRuleTimingPreWrite,
		Category:    "capacity",
		RuleAST:     ruleAST,
	})
	if err != nil {
		log.Fatalf("UpsertValidationRule failed: %v", err)
	}
	fmt.Printf("Persisted validation rule: id=%s name=%s bo=%s\n", desc.ID, desc.Name, desc.BOName)

	// Pull every live order row and evaluate the persisted rule against
	// each one via the real unified engine, comparing against what the
	// DB's own CHECK constraint already enforces (every row in the table
	// necessarily satisfies it - the DB wouldn't have accepted a row that
	// didn't - so every evaluation here MUST return true; that is the
	// verification).
	type orderRow struct {
		ID        string  `db:"id"`
		FilledQty float64 `db:"filled_qty"`
		Quantity  float64 `db:"quantity"`
	}
	var rows []orderRow
	if err := db.SelectContext(ctx, &rows, `SELECT id, filled_qty, quantity FROM oms.orders`); err != nil {
		log.Fatalf("querying oms.orders: %v", err)
	}

	allPassed := true
	for _, row := range rows {
		data := map[string]interface{}{
			"filled_qty": row.FilledQty,
			"quantity":   row.Quantity,
		}
		result, err := svc.Evaluate(ctx, desc.ID, data)
		if err != nil {
			log.Fatalf("evaluate row %s: %v", row.ID, err)
		}
		status := "PASS"
		if !result {
			status = "FAIL"
			allPassed = false
		}
		fmt.Printf("  order %s: filled_qty=%.2f quantity=%.2f -> engine=%v (DB CHECK: satisfied by construction) [%s]\n",
			row.ID, row.FilledQty, row.Quantity, result, status)
	}

	if !allPassed {
		log.Fatalf("MISMATCH: at least one row the unified engine evaluated as FAIL, despite satisfying the live DB CHECK constraint by construction - engine/constraint disagree")
	}
	fmt.Printf("\nAll %d live rows: unified engine agrees with the DB CHECK constraint (chk_orders_filled_qty).\n", len(rows))

	// Agreement on live data alone doesn't prove the rule can detect a
	// violation - the DB CHECK guarantees every stored row already
	// satisfies it, so a rule that always returns true would "pass" this
	// far too. Prove the negative case directly: synthetic data the DB
	// constraint would have rejected, evaluated by the same persisted
	// rule via the same engine, must fail.
	violations := []struct {
		name string
		data map[string]interface{}
	}{
		{"filled_qty > quantity (over-fill)", map[string]interface{}{"filled_qty": 600.0, "quantity": 500.0}},
		{"filled_qty < 0 (negative fill)", map[string]interface{}{"filled_qty": -1.0, "quantity": 100.0}},
	}
	for _, v := range violations {
		result, err := svc.Evaluate(ctx, desc.ID, v.data)
		if err != nil {
			log.Fatalf("evaluate violation case %q: %v", v.name, err)
		}
		if result {
			log.Fatalf("MISMATCH: engine returned PASS for a case the DB CHECK would reject: %s", v.name)
		}
		fmt.Printf("  violation case %q: engine=false (correctly detected) [PASS]\n", v.name)
	}

	fmt.Println("\nEnd-to-end proof, both directions: authored in the AST -> persisted as a catalog_node -> evaluated by internal/rules/vm.AdvancedEvaluator -> agrees with ground truth on real data AND correctly rejects synthetic violations.")
}
