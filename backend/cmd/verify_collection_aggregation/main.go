// One-off proof for collection aggregation: SUM(OrderAllocations.target_qty) =
// TargetQuantity, evaluated through the real write path and service stack.
//
// Case list:
//  1. Author SUM expression rule and legacy allocation_target_qty_sum rule together
//     (proves Decision 3: they coexist, both authored against the same BO).
//  2. Zero-allocation order -> both rules produce violations (not rule errors).
//     This is the integration-level proof of the empty-collection semantic:
//     OrderAllocations=[] -> SUM=0 -> 0 != target_qty -> violation.
//  3. Complete allocation (sum matches target) -> both rules pass.
//  4. Partial allocation (sum != target) -> both rules fail.
//
// Key signals checked at each step:
//   - Rule-authored-via-service: confirmed by no error on UpsertValidationRule.
//   - Expression evaluates: confirmed by violation appearing in analytics.ListViolations.
//   - Not a rule error: confirmed by rule_error=false on the violation.
//   - Empty collection = violation, not error: zero-allocation case proves it.
//
// Following the verify_order_validations pattern: local tenant, no login,
// real DB, real service stack.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/metadata"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const tenantID = "99e99e99-99e9-49e9-89e9-99e99e99e999"

var (
	db    *sqlx.DB
	boSvc *metadata.BusinessObjectService
	ctx   context.Context
	secCtx *security.Context
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	var err error
	db, err = sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close()

	ctx = security.WithAuthInfo(context.Background(), security.AuthInfo{
		UserID: "verify_collection_aggregation", Roles: []string{"global_admin"},
	})
	secCtx = &security.Context{TenantID: tenantID, UserID: "verify_collection_aggregation"}
	boSvc = metadata.NewBusinessObjectService(db, nil, nil, nil)
	ruleSvc := analytics.NewValidationRuleService(db)

	seedAccounts()
	ruleIDs := authorRules(ruleSvc)

	fmt.Println("\n=== Test 1: zero-allocation order -> violations from BOTH rules (empty-collection semantic) ===")
	testZeroAllocationsViolations(ruleSvc, ruleIDs)

	fmt.Println("\n=== Test 2: complete allocation (sum == target) -> both rules pass ===")
	testCompleteAllocationPasses(ruleSvc, ruleIDs)

	fmt.Println("\n=== Test 3: partial allocation (sum != target) -> both rules fail ===")
	testPartialAllocationViolations(ruleSvc, ruleIDs)

	fmt.Println("\nAll checks completed. Violations recorded in validation_rule_violations.")
}

func seedAccounts() {
	_, err := db.ExecContext(ctx, `
		INSERT INTO orm.account (account_id, status, is_discretionary) VALUES
			('ACCT-DISC-ACTIVE', 'ACTIVE', true)
		ON CONFLICT (account_id) DO UPDATE SET status = EXCLUDED.status, is_discretionary = EXCLUDED.is_discretionary
	`)
	if err != nil {
		log.Fatalf("seed accounts: %v", err)
	}
	fmt.Println("Seeded orm.account: ACCT-DISC-ACTIVE (discretionary, ACTIVE)")
}

func authorRules(svc *analytics.ValidationRuleService) map[string]uuid.UUID {
	// SUM expression rule: uses semantic term TargetQuantity (→ physical target_qty
	// via binding) and collection reference OrderAllocations (→ order_allocation rows
	// loaded by loadOrderContext). This is the expression the feature exists to enable.
	sumRuleAST := `{"type":"expression","root":{"op":"==","left":{"func":"SUM","args":[{"path":"OrderAllocations.target_qty"}]},"right":{"path":"TargetQuantity"}}}`
	sumDesc, err := svc.UpsertValidationRule(ctx, models.UpsertValidationRuleRequest{
		TenantID:    tenantID,
		BOName:      "order",
		Name:        "Allocation sum via collection expression (collection aggregation proof)",
		Description: "SUM(OrderAllocations.target_qty) = TargetQuantity — verifies collection aggregation through the real service stack",
		Severity:    models.ValidationRuleSeverityBlock,
		Timing:      models.ValidationRuleTimingPreWrite,
		Category:    "oms",
		RuleAST:     json.RawMessage(sumRuleAST),
	})
	if err != nil {
		log.Fatalf("author SUM expression rule: %v", err)
	}
	fmt.Printf("Authored SUM expression rule: id=%s\n", sumDesc.ID)

	// Legacy scalar rule: same semantic constraint (allocation sum must match target),
	// but using the pre-existing allocation_target_qty_sum scalar context key.
	// Proves Decision 3: both rules can be active simultaneously; the SUM expression
	// rule doesn't replace or break the legacy scalar rule.
	legacyRuleAST := `{"type":"expression","root":{"op":"==","left":{"path":"allocation_target_qty_sum"},"right":{"path":"target_qty"}}}`
	legacyDesc, err := svc.UpsertValidationRule(ctx, models.UpsertValidationRuleRequest{
		TenantID:    tenantID,
		BOName:      "order",
		Name:        "Allocation completeness (legacy scalar, coexistence proof)",
		Description: "allocation_target_qty_sum = target_qty — same constraint as SUM rule, scalar form; verifies legacy coexistence",
		Severity:    models.ValidationRuleSeverityBlock,
		Timing:      models.ValidationRuleTimingPreWrite,
		Category:    "oms",
		RuleAST:     json.RawMessage(legacyRuleAST),
	})
	if err != nil {
		log.Fatalf("author legacy scalar rule: %v", err)
	}
	fmt.Printf("Authored legacy scalar rule: id=%s\n", legacyDesc.ID)

	return map[string]uuid.UUID{
		"sum_expr":     sumDesc.ID,
		"legacy_scalar": legacyDesc.ID,
	}
}

func setEnforce(on bool) {
	if on {
		os.Setenv("VALIDATION_RULES_ENFORCE", "true")
	} else {
		os.Unsetenv("VALIDATION_RULES_ENFORCE")
	}
}

func violationsForOrder(orderID string) []analytics.ViolationSummary {
	violations, err := analytics.ListViolations(ctx, db, tenantID, "order", 500)
	if err != nil {
		log.Fatalf("list violations: %v", err)
	}
	var ours []analytics.ViolationSummary
	for _, v := range violations {
		if v.RecordID == orderID {
			ours = append(ours, v)
		}
	}
	return ours
}

// testZeroAllocationsViolations: create an order with no allocations (OrderAllocations=[]).
// Expected: SUM(OrderAllocations.target_qty) = 0; TargetQuantity (target_qty) = 100;
// 0 != 100 -> violation from SUM rule. Legacy rule: allocation_target_qty_sum = 0 != 100 ->
// also a violation. Both violations must have rule_error=false (evaluation, not broken context).
func testZeroAllocationsViolations(ruleSvc *analytics.ValidationRuleService, ruleIDs map[string]uuid.UUID) {
	setEnforce(false)
	// Use unique target_qty so concurrent runs don't collide on the duplicate-order rule.
	targetQty := float64(100 + time.Now().UnixNano()%100000)
	order, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"sec_id": 1, "side": "BUY", "order_type": "MARKET",
			"target_qty": targetQty, "executed_qty": 0, "leaves_qty": targetQty,
			"trade_date": time.Now().Format("2006-01-02"),
		},
	}, "verify_collection_aggregation")
	if err != nil {
		log.Fatalf("create order (zero allocations): %v", err)
	}
	orderID := fmt.Sprintf("%v", order["id"])
	fmt.Printf("Created order %s with target_qty=%.0f, zero allocations\n", orderID, targetQty)

	// Trigger rule evaluation on the order (even in shadow mode, violations are persisted).
	// A status update re-runs evaluateAndEnforceRules with the current context.
	_, err = boSvc.UpdateBORecord(ctx, secCtx, "order", orderID, models.BOCrudRecordRequest{
		Record: map[string]interface{}{"status": "DRAFT"},
	}, "verify_collection_aggregation")
	if err != nil {
		log.Fatalf("update order to trigger re-evaluation: %v", err)
	}

	violations := violationsForOrder(orderID)
	var sumViolation, legacyViolation *analytics.ViolationSummary
	for _, v := range violations {
		if v.RuleID == ruleIDs["sum_expr"] {
			sumViolation = &v
		}
		if v.RuleID == ruleIDs["legacy_scalar"] {
			legacyViolation = &v
		}
	}

	if sumViolation == nil {
		log.Fatalf("SUM expression rule produced no violation for zero-allocation order (expected violation: SUM=0 != target_qty=%.0f)", targetQty)
	}
	if sumViolation.RuleError {
		log.Fatalf("SUM rule violation has rule_error=true — empty collection should evaluate to 0 (violation), not error")
	}
	fmt.Printf("SUM rule violation OK: rule_error=false, message=%q\n", sumViolation.Message)

	if legacyViolation == nil {
		log.Fatalf("legacy scalar rule produced no violation for zero-allocation order (expected violation: sum=0 != target_qty=%.0f)", targetQty)
	}
	if legacyViolation.RuleError {
		log.Fatalf("legacy rule violation has rule_error=true — scalar evaluation should work, not error")
	}
	fmt.Printf("Legacy rule violation OK: rule_error=false, message=%q\n", legacyViolation.Message)
	fmt.Printf("PASS: zero-allocation order -> both rules produce violations (not errors). Empty-collection semantic confirmed.\n")
}

// testCompleteAllocationPasses: order with full allocation (sum == target_qty).
// Both rules should pass silently.
func testCompleteAllocationPasses(ruleSvc *analytics.ValidationRuleService, ruleIDs map[string]uuid.UUID) {
	setEnforce(false)
	targetQty := float64(100 + time.Now().UnixNano()%100000)
	order, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"sec_id": 1, "side": "BUY", "order_type": "MARKET",
			"target_qty": targetQty, "executed_qty": 0, "leaves_qty": targetQty,
			"trade_date": time.Now().Format("2006-01-02"),
		},
	}, "verify_collection_aggregation")
	if err != nil {
		log.Fatalf("create order: %v", err)
	}
	orderID := fmt.Sprintf("%v", order["id"])

	// Full allocation: create one row with target_qty == order.target_qty.
	_, err = boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"order_id": orderID, "account_id": "ACCT-DISC-ACTIVE",
			"target_qty": targetQty, "allocated_qty": targetQty,
		},
	}, "verify_collection_aggregation")
	if err != nil {
		log.Fatalf("create allocation: %v", err)
	}
	fmt.Printf("Created order %s with full allocation (target_qty=%.0f)\n", orderID, targetQty)

	before := violationCountForOrder(orderID)
	setEnforce(true)
	_, err = boSvc.UpdateBORecord(ctx, secCtx, "order", orderID, models.BOCrudRecordRequest{
		Record: map[string]interface{}{"status": "ALLOCATED"},
	}, "verify_collection_aggregation")
	if err != nil {
		log.Fatalf("update order (enforcement ON, full allocation) failed unexpectedly: %v", err)
	}

	after := violationCountForOrder(orderID)
	if after != before {
		log.Fatalf("order with complete allocation produced new violations: %d -> %d (expected 0 new)", before, after)
	}
	fmt.Printf("PASS: order with complete allocation (sum %.0f == target %.0f) -> zero violations. Both rules confirmed passing.\n", targetQty, targetQty)
}

// testPartialAllocationViolations: order with partial allocation (sum != target_qty).
// Both rules should produce violations.
func testPartialAllocationViolations(ruleSvc *analytics.ValidationRuleService, ruleIDs map[string]uuid.UUID) {
	setEnforce(false)
	targetQty := float64(100 + time.Now().UnixNano()%100000)
	partialQty := targetQty * 0.6 // 60 of 100 allocated
	order, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"sec_id": 1, "side": "SELL", "order_type": "MARKET",
			"target_qty": targetQty, "executed_qty": 0, "leaves_qty": targetQty,
			"trade_date": time.Now().Format("2006-01-02"),
		},
	}, "verify_collection_aggregation")
	if err != nil {
		log.Fatalf("create order: %v", err)
	}
	orderID := fmt.Sprintf("%v", order["id"])

	// Partial allocation: only partialQty allocated.
	_, err = boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"order_id": orderID, "account_id": "ACCT-DISC-ACTIVE",
			"target_qty": partialQty, "allocated_qty": 0,
		},
	}, "verify_collection_aggregation")
	if err != nil {
		log.Fatalf("create partial allocation: %v", err)
	}
	fmt.Printf("Created order %s with partial allocation (%.0f of %.0f)\n", orderID, partialQty, targetQty)

	setEnforce(true)
	_, err = boSvc.UpdateBORecord(ctx, secCtx, "order", orderID, models.BOCrudRecordRequest{
		Record: map[string]interface{}{"status": "PARTIALLY_ALLOCATED"},
	}, "verify_collection_aggregation")
	// Shadow mode off: enforcement is ON, so a BLOCK violation causes an error.
	if err == nil {
		log.Fatalf("expected update to be rejected (partial allocation, both rules should fire BLOCK), but it succeeded")
	}
	fmt.Printf("Update correctly blocked: %v\n", err)

	violations := violationsForOrder(orderID)
	var sumViolation, legacyViolation *analytics.ViolationSummary
	for _, v := range violations {
		if v.RuleID == ruleIDs["sum_expr"] {
			sumViolation = &v
		}
		if v.RuleID == ruleIDs["legacy_scalar"] {
			legacyViolation = &v
		}
	}

	if sumViolation == nil {
		log.Fatalf("SUM rule produced no violation for partial allocation (sum=%.0f != target=%.0f)", partialQty, targetQty)
	}
	if legacyViolation == nil {
		log.Fatalf("legacy rule produced no violation for partial allocation (sum=%.0f != target=%.0f)", partialQty, targetQty)
	}
	if sumViolation.RuleError || legacyViolation.RuleError {
		log.Fatalf("a partial-allocation violation has rule_error=true — should be a real violation, not an evaluation error")
	}
	fmt.Printf("PASS: partial allocation (%.0f of %.0f) -> both rules blocked correctly. SUM message=%q\n", partialQty, targetQty, sumViolation.Message)
}

func violationCountForOrder(orderID string) int {
	violations, err := analytics.ListViolations(ctx, db, tenantID, "order", 500)
	if err != nil {
		log.Fatalf("list violations: %v", err)
	}
	n := 0
	for _, v := range violations {
		if v.RecordID == orderID {
			n++
		}
	}
	return n
}