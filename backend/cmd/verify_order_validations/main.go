// One-off verification for the completed validation engine package: the
// local `orm` schema, severity-driven enforcement (BLOCK rejects, WARN
// logs), the generalized context provider, and the Order BO rule set.
// Mirrors the verify_oracle_rule / verify_shadow_context pattern - a
// runnable, permanent proof, no login required.
//
// Authors 5 rules against the Order BO through the real
// ValidationRuleService, then drives real writes through the real
// BusinessObjectService.CreateBORecord/UpdateBORecord path:
//
//  1. LIMIT order without limit_price, enforcement ON -> BLOCK, write
//     rejected, row does not exist afterward.
//  2. A full, consistent order/allocation/execution chain, enforcement
//     ON -> every rule passes, nothing rejected, no violations persisted
//     for this order.
//  3. An order with incomplete allocations, enforcement OFF (shadow,
//     the default) -> write succeeds, but a BLOCK-severity violation is
//     still logged and persisted - proving shadow mode doesn't silently
//     drop what it finds.
//  4. A non-discretionary account with no manager_id set on the order,
//     enforcement ON -> BLOCK, rejected.
//  5. Oracle agreement: target_qty > 0 is both a live DB CHECK
//     (chk_order_target_qty_positive) and a rule - attempting
//     target_qty = 0 is rejected at the DB layer itself, and the rule
//     evaluates the same data as false.
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
	db     *sqlx.DB
	boSvc  *metadata.BusinessObjectService
	ctx    context.Context
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
		UserID: "verify_order_validations", Roles: []string{"global_admin"},
	})
	secCtx = &security.Context{TenantID: tenantID, UserID: "verify_order_validations"}
	boSvc = metadata.NewBusinessObjectService(db, nil, nil, nil)
	ruleSvc := analytics.NewValidationRuleService(db)

	seedAccounts()
	ruleIDs := authorOrderRules(ruleSvc)

	fmt.Println("\n=== Test 1: LIMIT order without limit_price, enforcement ON -> BLOCK ===")
	testLimitOrderRejected()

	fmt.Println("\n=== Test 2: full consistent order chain, enforcement ON -> everything passes ===")
	testCleanChainPasses()

	fmt.Println("\n=== Test 3: incomplete allocations, enforcement OFF (shadow default) -> logged, not blocked ===")
	testIncompleteAllocationsShadow()

	fmt.Println("\n=== Test 4: non-discretionary account, no manager_id, enforcement ON -> BLOCK ===")
	testComplianceRejected()

	fmt.Println("\n=== Test 5: oracle agreement on target_qty > 0 ===")
	testTargetQtyOracle(ruleSvc, ruleIDs["target_qty_positive"])

	fmt.Println("\n=== Test 6: rule authored against a semantic term (TargetQuantity, not target_qty) ===")
	testSemanticTermRule(ruleSvc)

	fmt.Println("\n=== Test 7: rule referencing an unresolvable term -> persisted as a rule error, not a silent pass ===")
	testUnresolvableTermFailsLoud(ruleSvc)

	fmt.Println("\nAll checks completed. See validation_rule_violations for the persisted record of every violation above.")
}

func seedAccounts() {
	_, err := db.ExecContext(ctx, `
		INSERT INTO orm.account (account_id, status, is_discretionary) VALUES
			('ACCT-DISC-ACTIVE', 'ACTIVE', true),
			('ACCT-NONDISC-ACTIVE', 'ACTIVE', false)
		ON CONFLICT (account_id) DO UPDATE SET status = EXCLUDED.status, is_discretionary = EXCLUDED.is_discretionary
	`)
	if err != nil {
		log.Fatalf("seed accounts: %v", err)
	}
	fmt.Println("Seeded orm.account: ACCT-DISC-ACTIVE (discretionary), ACCT-NONDISC-ACTIVE (non-discretionary)")
}

func authorOrderRules(svc *analytics.ValidationRuleService) map[string]uuid.UUID {
	rules := []struct {
		key      string
		name     string
		severity string
		ast      string
	}{
		{
			"limit_needs_price",
			"LIMIT order requires limit_price",
			models.ValidationRuleSeverityBlock,
			`{"type":"group","operator":"OR","conditions":[
				{"type":"condition","field":"order_type","operator":"not_equals","value":"LIMIT"},
				{"type":"expression","root":{"func":"NOT_EMPTY","args":[{"path":"limit_price"}]}}
			]}`,
		},
		{
			"allocation_completeness",
			"Allocation quantities must sum to order target_qty",
			models.ValidationRuleSeverityBlock,
			`{"type":"expression","root":{"op":"==","left":{"path":"allocation_target_qty_sum"},"right":{"path":"target_qty"}}}`,
		},
		{
			"reconciliation_leaves_qty",
			"leaves_qty must equal target_qty minus executed_qty",
			models.ValidationRuleSeverityBlock,
			`{"type":"expression","root":{"op":"==","left":{"path":"leaves_qty"},"right":{"op":"-","left":{"path":"target_qty"},"right":{"path":"executed_qty"}}}}`,
		},
		{
			"account_compliance",
			"Account must be ACTIVE; non-discretionary accounts require manager_id",
			models.ValidationRuleSeverityBlock,
			`{"type":"group","operator":"AND","conditions":[
				{"type":"condition","field":"account_status","operator":"equals","value":"ACTIVE"},
				{"type":"group","operator":"OR","conditions":[
					{"type":"condition","field":"account_is_discretionary","operator":"equals","value":true},
					{"type":"expression","root":{"func":"NOT_EMPTY","args":[{"path":"manager_id"}]}}
				]}
			]}`,
		},
		{
			"target_qty_positive",
			"target_qty must be positive",
			models.ValidationRuleSeverityBlock,
			`{"type":"condition","field":"target_qty","operator":"greater_than","value":0}`,
		},
	}

	ids := make(map[string]uuid.UUID)
	for _, r := range rules {
		desc, err := svc.UpsertValidationRule(ctx, models.UpsertValidationRuleRequest{
			TenantID:    tenantID,
			BOName:      "order",
			Name:        r.name,
			Description: "Order BO validation set - verify_order_validations proof",
			Severity:    r.severity,
			Timing:      models.ValidationRuleTimingPreWrite,
			Category:    "oms",
			RuleAST:     json.RawMessage(r.ast),
		})
		if err != nil {
			log.Fatalf("author rule %s: %v", r.key, err)
		}
		ids[r.key] = desc.ID
		fmt.Printf("Authored rule %q: id=%s severity=%s\n", r.name, desc.ID, r.severity)
	}
	return ids
}

func setEnforce(on bool) {
	if on {
		os.Setenv("VALIDATION_RULES_ENFORCE", "true")
	} else {
		os.Unsetenv("VALIDATION_RULES_ENFORCE")
	}
}

func testLimitOrderRejected() {
	setEnforce(true)
	before := countOrders()

	// Supply the id ourselves (CreateBORecord auto-generates one only if
	// missing) so a rejected write's id is still known - the write's own
	// return value is nil on rejection, but the violation row needs a
	// known id to look up afterward.
	rejectedID := uuid.New().String()
	_, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"id": rejectedID, "sec_id": 1, "side": "BUY", "order_type": "LIMIT",
			"target_qty": 100, "executed_qty": 0, "leaves_qty": 100,
			"trade_date": time.Now().Format("2006-01-02"),
		},
	}, "verify_order_validations")
	after := countOrders()
	if err == nil {
		log.Fatalf("expected CreateBORecord to be rejected (LIMIT without limit_price), but it succeeded")
	}
	if after != before {
		log.Fatalf("write was rejected (err=%v) but orm.order row count still changed: %d -> %d", err, before, after)
	}
	fmt.Printf("Write correctly rejected: %v\nOrder count unchanged (%d) - the row was never persisted.\n", err, after)

	// Pin the subtle case: does a BLOCK rejection's own violation record
	// survive the transaction rollback that the rejection itself caused?
	// If PersistViolation shared the write's transaction, the rollback
	// would erase the rejection's own evidence - a BLOCK write would
	// leave no row AND no audit trail. It's persisted via s.db, outside
	// the transaction, specifically so this doesn't happen - verify that
	// design decision against the real database, not just the source.
	violations, err := analytics.ListViolations(ctx, db, tenantID, "order", 500)
	if err != nil {
		log.Fatalf("list violations: %v", err)
	}
	found := false
	for _, v := range violations {
		if v.RecordID == rejectedID && v.WriteBlocked {
			found = true
		}
	}
	if !found {
		log.Fatalf("BLOCK rejection for order %s left no write_blocked=true violation record - "+
			"the rollback erased its own evidence, an enforcement mechanism whose rejections "+
			"aren't auditable", rejectedID)
	}
	fmt.Printf("Confirmed: the rejection's own violation record survived the rollback (record_id=%s, write_blocked=true) - BLOCK rejections are auditable.\n", rejectedID)
}

func testCleanChainPasses() {
	// A freshly created order has no allocations yet - that's expected,
	// transient incompleteness, not something to enforce over. Build the
	// chain with enforcement off, then flip it on for the final update
	// once the order is genuinely complete, and check that update alone
	// (not the whole order's history) produced zero new violations.
	setEnforce(false)
	order, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"sec_id": 1, "side": "BUY", "order_type": "MARKET",
			"target_qty": 100, "executed_qty": 0, "leaves_qty": 100,
			"trade_date": time.Now().Format("2006-01-02"),
		},
	}, "verify_order_validations")
	if err != nil {
		log.Fatalf("clean order creation failed unexpectedly: %v", err)
	}
	orderID := fmt.Sprintf("%v", order["id"])
	fmt.Printf("Created order %s (target_qty=100)\n", orderID)

	_, err = boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"order_id": orderID, "account_id": "ACCT-DISC-ACTIVE",
			"target_qty": 100, "allocated_qty": 0,
		},
	}, "verify_order_validations")
	if err != nil {
		log.Fatalf("allocation creation failed: %v", err)
	}
	fmt.Println("Created order_allocation: 100 to ACCT-DISC-ACTIVE (discretionary, ACTIVE) - sum matches target_qty")

	before := violationCountForOrder(orderID)
	setEnforce(true)
	_, err = boSvc.UpdateBORecord(ctx, secCtx, "order", orderID, models.BOCrudRecordRequest{
		Record: map[string]interface{}{"executed_qty": 100, "leaves_qty": 0},
	}, "verify_order_validations")
	if err != nil {
		log.Fatalf("order update (full fill) failed unexpectedly, with the order now fully allocated: %v", err)
	}
	fmt.Println("Updated order to executed_qty=100, leaves_qty=0, enforcement ON - reconciliation, allocation-completeness, and compliance rules all passed silently.")

	after := violationCountForOrder(orderID)
	if after != before {
		log.Fatalf("the final, complete-state update unexpectedly produced a new violation: %d -> %d", before, after)
	}
	fmt.Println("Confirmed: the final update produced zero new violations - the completed chain really is clean.")
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

func testIncompleteAllocationsShadow() {
	setEnforce(false)
	order, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"sec_id": 1, "side": "SELL", "order_type": "MARKET",
			"target_qty": 100, "executed_qty": 0, "leaves_qty": 100,
			"trade_date": time.Now().Format("2006-01-02"),
		},
	}, "verify_order_validations")
	if err != nil {
		log.Fatalf("order creation failed unexpectedly (shadow mode never blocks): %v", err)
	}
	orderID := fmt.Sprintf("%v", order["id"])

	_, err = boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"order_id": orderID, "account_id": "ACCT-DISC-ACTIVE",
			"target_qty": 40, "allocated_qty": 0, // only 40 of 100 allocated
		},
	}, "verify_order_validations")
	if err != nil {
		log.Fatalf("allocation creation failed: %v", err)
	}

	// Touch the order again so evaluateAndEnforceRules re-runs against the
	// current (still-incomplete) allocation sum.
	_, err = boSvc.UpdateBORecord(ctx, secCtx, "order", orderID, models.BOCrudRecordRequest{
		Record: map[string]interface{}{"status": "PARTIALLY_ALLOCATED"},
	}, "verify_order_validations")
	if err != nil {
		log.Fatalf("shadow-mode update was rejected, but shadow mode must never block: %v", err)
	}
	fmt.Printf("Order %s updated successfully despite incomplete allocations (40 of 100) - shadow mode did not block it.\n", orderID)

	violations, err := analytics.ListViolations(ctx, db, tenantID, "order", 500)
	if err != nil {
		log.Fatalf("list violations: %v", err)
	}
	found := false
	for _, v := range violations {
		if v.RecordID == orderID && v.RuleName == "Allocation quantities must sum to order target_qty" {
			found = true
			fmt.Printf("Found persisted violation: severity=%s write_blocked=%v message=%q\n", v.Severity, v.WriteBlocked, v.Message)
		}
	}
	if !found {
		log.Fatalf("expected a persisted allocation-completeness violation for order %s, found none", orderID)
	}
}

func testComplianceRejected() {
	// Same sequencing as testCleanChainPasses: build with enforcement off
	// (no account context exists until an allocation does), flip it on
	// only for the write that should actually get caught. Allocation sum
	// matches target_qty here on purpose, so allocation_completeness
	// passes and account_compliance is isolated as the one rule failing.
	setEnforce(false)
	order, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"sec_id": 1, "side": "BUY", "order_type": "MARKET",
			"target_qty": 50, "executed_qty": 0, "leaves_qty": 50,
			"trade_date": time.Now().Format("2006-01-02"),
			// manager_id deliberately omitted
		},
	}, "verify_order_validations")
	if err != nil {
		log.Fatalf("order creation failed unexpectedly: %v", err)
	}
	orderID := fmt.Sprintf("%v", order["id"])

	_, err = boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"order_id": orderID, "account_id": "ACCT-NONDISC-ACTIVE",
			"target_qty": 50, "allocated_qty": 0,
		},
	}, "verify_order_validations")
	if err != nil {
		log.Fatalf("allocation creation failed: %v", err)
	}

	// Now the account_compliance rule has an account to check: non-discretionary,
	// no manager_id on the order -> BLOCK on the next order write.
	setEnforce(true)
	_, err = boSvc.UpdateBORecord(ctx, secCtx, "order", orderID, models.BOCrudRecordRequest{
		Record: map[string]interface{}{"status": "ALLOCATED"},
	}, "verify_order_validations")
	if err == nil {
		log.Fatalf("expected the account-compliance rule to reject this update (non-discretionary account, no manager_id), but it succeeded")
	}
	fmt.Printf("Write correctly rejected: %v\n", err)
}

func testTargetQtyOracle(svc *analytics.ValidationRuleService, ruleID uuid.UUID) {
	setEnforce(true)
	before := countOrders()
	_, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"sec_id": 1, "side": "BUY", "order_type": "MARKET",
			"target_qty": 0, "executed_qty": 0, "leaves_qty": 0,
			"trade_date": time.Now().Format("2006-01-02"),
		},
	}, "verify_order_validations")
	after := countOrders()
	if err == nil {
		log.Fatalf("expected target_qty=0 to be rejected by the live DB CHECK (chk_order_target_qty_positive), but the write succeeded")
	}
	if after != before {
		log.Fatalf("DB CHECK should have prevented any row from landing, but count changed: %d -> %d", before, after)
	}
	fmt.Printf("DB CHECK chk_order_target_qty_positive rejected target_qty=0, as expected: %v\n", err)

	result, err := svc.Evaluate(ctx, ruleID, map[string]interface{}{"target_qty": 0.0})
	if err != nil {
		log.Fatalf("rule evaluation errored: %v", err)
	}
	if result {
		log.Fatalf("MISMATCH: unified engine evaluated target_qty=0 as PASS, but the live DB CHECK rejects it - engine/constraint disagree")
	}
	fmt.Println("Unified engine agrees: target_qty=0 evaluates to FAIL, matching the live DB CHECK constraint.")
}

func countOrders() int {
	var n int
	if err := db.GetContext(ctx, &n, `SELECT count(*) FROM orm."order"`); err != nil {
		log.Fatalf("count orders: %v", err)
	}
	return n
}

// testSemanticTermRule authors a rule against "TargetQuantity" - the
// semantic term (business_object_fields.field_name), not "target_qty"
// (the physical column it's currently bound to) - and proves it
// evaluates identically to the physical-named rule. This is the whole
// point of the semantic-term retrofit: the rule stays valid if the BO's
// binding ever points TargetQuantity at a different physical column.
func testSemanticTermRule(svc *analytics.ValidationRuleService) {
	desc, err := svc.UpsertValidationRule(ctx, models.UpsertValidationRuleRequest{
		TenantID: tenantID, BOName: "order",
		Name:        "target_qty positive, authored via semantic term",
		Description: "Same constraint as target_qty_positive, authored against TargetQuantity (semantic) instead of target_qty (physical) - verify_order_validations proof",
		Severity:    models.ValidationRuleSeverityBlock,
		Timing:      models.ValidationRuleTimingPreWrite,
		Category:    "oms",
		RuleAST:     json.RawMessage(`{"type":"condition","field":"TargetQuantity","operator":"greater_than","value":0}`),
	})
	if err != nil {
		log.Fatalf("author semantic-term rule: %v", err)
	}
	fmt.Printf("Authored rule against semantic term \"TargetQuantity\": id=%s\n", desc.ID)

	// svc.Evaluate has no BO/binding context at all - it evaluates the
	// rule_ast against exactly the data map it's handed, no alias
	// resolution. Supplying "TargetQuantity" directly here (not
	// "target_qty") is the honest sanity check for that path: it proves
	// the rule_ast itself is well-formed and matches its own field name,
	// independent of the binding-resolution machinery proven below.
	pass, err := svc.Evaluate(ctx, desc.ID, map[string]interface{}{"TargetQuantity": 100.0})
	if err != nil {
		log.Fatalf("evaluate semantic-term rule: %v", err)
	}
	if !pass {
		log.Fatalf("MISMATCH: semantic-term rule should PASS when its own field name is supplied directly")
	}
	fmt.Println("svc.Evaluate (direct, no binding resolution) with TargetQuantity=100 -> PASS - the rule_ast itself is well-formed.")

	// A freshly created order has no allocations yet, which independently
	// trips the (unrelated) allocation-completeness and account-compliance
	// rules - same staging as testCleanChainPasses. What this test cares
	// about is only whether the semantic-term rule itself fires
	// correctly, so create with enforcement off and check that specific
	// rule's absence from the violations list, rather than requiring the
	// whole write to succeed outright.
	setEnforce(false)
	order, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"sec_id": 1, "side": "BUY", "order_type": "MARKET",
			"target_qty": 100, "executed_qty": 0, "leaves_qty": 100,
			"trade_date": time.Now().Format("2006-01-02"),
		},
	}, "verify_order_validations")
	if err != nil {
		log.Fatalf("order creation failed unexpectedly (shadow mode never blocks): %v", err)
	}
	orderID := fmt.Sprintf("%v", order["id"])

	violations, err := analytics.ListViolations(ctx, db, tenantID, "order", 500)
	if err != nil {
		log.Fatalf("list violations: %v", err)
	}
	for _, v := range violations {
		if v.RecordID == orderID && v.RuleID == desc.ID {
			log.Fatalf("semantic-term rule fired against order %s (target_qty=100 > 0 should pass): %s", orderID, v.Message)
		}
	}
	fmt.Printf("Created order %s through the real write path - the semantic-term rule did not fire, proving the binding-resolution alias (\"TargetQuantity\" -> target_qty's value) worked for a real write, not just a hand-built payload.\n", orderID)
}

// testUnresolvableTermFailsLoud authors a rule against a semantic term
// that doesn't exist on the Order BO's field list, and proves the
// engagement's recurring failure mode (a rule that looks wired up but
// silently never fires) can't happen here: the rule evaluation errors,
// and that error is persisted as a rule_error=true violation - visible
// and auditable - not swallowed as a skipped rule or a false pass.
func testUnresolvableTermFailsLoud(svc *analytics.ValidationRuleService) {
	// Unique name per run (not a fixed one) - UpsertValidationRule's
	// ON CONFLICT DO UPDATE doesn't reset is_active, so reusing a fixed
	// name across runs risks a previous run's self-retirement (see below)
	// leaving this probe rule inactive - or, worse, active for the
	// *entire* run (Tests 1-6 too) if a previous run's leftover row gets
	// manually reactivated in between. A fresh name each run means a
	// fresh row, active only for the duration of this one test, which
	// this function retires at the end regardless.
	probeName := fmt.Sprintf("Deliberately unresolvable term (proof, %d)", time.Now().UnixNano())
	desc, err := svc.UpsertValidationRule(ctx, models.UpsertValidationRuleRequest{
		TenantID: tenantID, BOName: "order",
		Name:        probeName,
		Description: "References a semantic term the Order BO has no MAPS_TO binding for - must fail loud, not silently pass. verify_order_validations proof.",
		Severity:    models.ValidationRuleSeverityBlock,
		Timing:      models.ValidationRuleTimingPreWrite,
		Category:    "oms",
		RuleAST:     json.RawMessage(`{"type":"condition","field":"NoSuchTerm","operator":"equals","value":"anything"}`),
	})
	if err != nil {
		log.Fatalf("author unresolvable-term rule: %v", err)
	}
	fmt.Printf("Authored rule against nonexistent term \"NoSuchTerm\": id=%s\n", desc.ID)

	setEnforce(true)
	orderID := uuid.New().String()
	_, err = boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"id": orderID, "sec_id": 1, "side": "BUY", "order_type": "MARKET",
			"target_qty": 100, "executed_qty": 0, "leaves_qty": 100,
			"trade_date": time.Now().Format("2006-01-02"),
		},
	}, "verify_order_validations")
	if err == nil {
		log.Fatalf("expected the write to be rejected: an unresolvable BLOCK-severity rule must not silently pass")
	}
	fmt.Printf("Write correctly rejected (rule error treated as at-least-as-serious as a real BLOCK): %v\n", err)

	violations, err := analytics.ListViolations(ctx, db, tenantID, "order", 500)
	if err != nil {
		log.Fatalf("list violations: %v", err)
	}
	found := false
	for _, v := range violations {
		if v.RecordID == orderID && v.RuleID == desc.ID && v.RuleError {
			found = true
			fmt.Printf("Found persisted rule-error violation: rule_error=true message=%q\n", v.Message)
		}
	}
	if !found {
		log.Fatalf("expected a persisted rule_error=true violation for the unresolvable-term rule against order %s, found none - it was silently skipped instead", orderID)
	}

	// Retire the probe rule (same is_active=false convention as the
	// 233-rule corpus retirement) so a second run of this script isn't
	// polluted by a permanently-broken rule left active in the catalog -
	// this rule exists only to prove the fail-loud behavior once.
	if _, err := db.ExecContext(ctx, `UPDATE catalog_node SET is_active = false WHERE id = $1`, desc.ID); err != nil {
		log.Fatalf("failed to retire probe rule %s: %v", desc.ID, err)
	}
	fmt.Printf("Retired probe rule %s (is_active = false) so it won't affect a future run.\n", desc.ID)
}
