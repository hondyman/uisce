// Phase 1 of the OMS validation implementation spec
// (docs/unified-rule-engine-handoff.md): the 20 new rules across
// Placement, Execution, ExecutionAllocation, and OrderAllocation, plus
// the 2 new rules on Order (over-placement, duplicate-order soft-check).
// Mirrors cmd/verify_order_validations' discipline - every rule proven
// with both a passing case and a violating case, through the real
// BusinessObjectService.CreateBORecord/UpdateBORecord write path, real
// ValidationRuleService authoring, real persisted violations.
//
// Does NOT re-author the 5 Order rules verify_order_validations already
// proves (LIMIT-needs-price, target_qty-positive, allocation-completeness,
// account-compliance, leaves-reconciliation) - the spec's own table
// mislabels "Leaves reconciliation" as NEW when reconciliation_leaves_qty
// already exists; checked against authorOrderRules before writing this,
// not assumed from the spec's table.
//
// Honesty note on "oracle" rules: only Order.target_qty > 0 has a live DB
// CHECK constraint today (chk_order_target_qty_positive, proven by
// verify_order_validations). Checked pg_constraint for the orm schema
// before writing this file - Placement.RoutedQuantity > 0,
// Execution.ExecQuantity/ExecPrice > 0, ExecutionAllocation's
// AllocationExecQuantity > 0, and OrderAllocation.TargetQuantity > 0 have
// NO corresponding DB CHECK constraints in this schema. They're still
// authored and proven here via pass/fail test cases, just not claimed as
// oracle-agreement proofs the way Order's target_qty rule is - adding new
// CHECK constraints was out of this spec's scope.
package main

import (
	"context"
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
	ruleS  *analytics.ValidationRuleService
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
		UserID: "verify_oms_validations", Roles: []string{"global_admin"},
	})
	secCtx = &security.Context{TenantID: tenantID, UserID: "verify_oms_validations"}
	boSvc = metadata.NewBusinessObjectService(db, nil, nil, nil)
	ruleS = analytics.NewValidationRuleService(db)

	seedReferenceData()

	fmt.Println("\n=== Order: 2 new rules (over-placement, duplicate-order) ===")
	testOrderNewRules()

	fmt.Println("\n=== Placement: 5 rules ===")
	testPlacementRules()

	fmt.Println("\n=== Execution: 6 rules ===")
	testExecutionRules()

	fmt.Println("\n=== ExecutionAllocation + OrderAllocation: 7 rules (shared chain) ===")
	testAllocationRules()

	fmt.Println("\nAll OMS validation checks completed. See validation_rule_violations for the persisted record of every case above.")
}

func setEnforce(on bool) {
	if on {
		os.Setenv("VALIDATION_RULES_ENFORCE", "true")
	} else {
		os.Unsetenv("VALIDATION_RULES_ENFORCE")
	}
}

func seedReferenceData() {
	if _, err := db.ExecContext(ctx, `
		INSERT INTO orm.account (account_id, status, is_discretionary) VALUES
			('ACCT-DISC-ACTIVE', 'ACTIVE', true),
			('ACCT-OMS-INACTIVE', 'INACTIVE', true)
		ON CONFLICT (account_id) DO UPDATE SET status = EXCLUDED.status, is_discretionary = EXCLUDED.is_discretionary
	`); err != nil {
		log.Fatalf("seed accounts: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO orm.broker (broker_id, status) VALUES
			('BRK-ACTIVE', 'ACTIVE'),
			('BRK-INACTIVE', 'INACTIVE')
		ON CONFLICT (broker_id) DO UPDATE SET status = EXCLUDED.status
	`); err != nil {
		log.Fatalf("seed brokers: %v", err)
	}
	fmt.Println("Seeded orm.account (ACCT-DISC-ACTIVE, ACCT-OMS-INACTIVE) and orm.broker (BRK-ACTIVE, BRK-INACTIVE).")
}

func author(boName, name, severity, timing, ast string) uuid.UUID {
	desc, err := ruleS.UpsertValidationRule(ctx, models.UpsertValidationRuleRequest{
		TenantID: tenantID, BOName: boName, Name: name,
		Description: "OMS validation spec, Phase 1 - verify_oms_validations proof",
		Severity:    severity, Timing: timing, Category: "oms",
		RuleAST: []byte(ast),
	})
	if err != nil {
		log.Fatalf("author rule %q on %s: %v", name, boName, err)
	}
	fmt.Printf("  authored %q (%s/%s): id=%s\n", name, severity, timing, desc.ID)
	return desc.ID
}

func createOrder(targetQty float64, side, orderType string, limitPrice *float64) string {
	rec := map[string]interface{}{
		"sec_id": 1, "side": side, "order_type": orderType,
		"target_qty": targetQty, "executed_qty": 0, "leaves_qty": targetQty,
		"trade_date": time.Now().Format("2006-01-02"),
	}
	if limitPrice != nil {
		rec["limit_price"] = *limitPrice
	}
	order, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{Record: rec}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("create order: %v", err)
	}
	return fmt.Sprintf("%v", order["id"])
}

func violationsFor(boKey, recordID string) []analytics.ViolationSummary {
	all, err := analytics.ListViolations(ctx, db, tenantID, boKey, 1000)
	if err != nil {
		log.Fatalf("list violations for %s: %v", boKey, err)
	}
	var out []analytics.ViolationSummary
	for _, v := range all {
		if v.RecordID == recordID {
			out = append(out, v)
		}
	}
	return out
}

func hasViolationNamed(boKey, recordID, ruleName string) bool {
	for _, v := range violationsFor(boKey, recordID) {
		if v.RuleName == ruleName {
			return true
		}
	}
	return false
}

// ---------- Order: 2 new rules ----------

func testOrderNewRules() {
	overPlacementID := author("order", "Placement routed quantity must not exceed order target_qty",
		models.ValidationRuleSeverityBlock, models.ValidationRuleTimingPreWrite,
		`{"type":"expression","root":{"op":"<=","left":{"path":"placement_routed_sum"},"right":{"path":"TargetQuantity"}}}`)
	_ = overPlacementID
	dupID := author("order", "Duplicate order soft-check",
		models.ValidationRuleSeverityWarn, models.ValidationRuleTimingPreWrite,
		`{"type":"condition","field":"duplicate_order_count","operator":"equals","value":0}`)
	_ = dupID

	// Over-placement: isolate it from the order's other already-active
	// rules (allocation-completeness, account-compliance - both BLOCK,
	// both fail on any order with no matching allocation) by staging a
	// full, otherwise-clean chain first, same discipline as
	// verify_order_validations' testCleanChainPasses - a matching
	// order_allocation to a discretionary ACTIVE account, built with
	// enforcement off, so only the placement-routed-sum variable differs
	// between the FAIL and PASS cases.
	setEnforce(false)
	orderID := createOrder(100, "BUY", "MARKET", nil)
	if _, err := boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"order_id": orderID, "account_id": "ACCT-DISC-ACTIVE", "target_qty": 100, "allocated_qty": 0},
	}, "verify_oms_validations"); err != nil {
		log.Fatalf("create matching order_allocation for over-placement isolation: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO orm.placement (id, order_id, broker_id, routed_qty, executed_qty, leaves_qty, status)
		VALUES ($1, $2, 'BRK-ACTIVE', 150, 0, 150, 'NEW')`, uuid.New().String(), orderID); err != nil {
		log.Fatalf("direct-insert oversized placement: %v", err)
	}
	setEnforce(true)
	_, err := boSvc.UpdateBORecord(ctx, secCtx, "order", orderID, models.BOCrudRecordRequest{
		Record: map[string]interface{}{"status": "PARTIALLY_ROUTED"},
	}, "verify_oms_validations")
	if err == nil {
		log.Fatalf("expected the order update to be rejected: placement_routed_sum (150) > target_qty (100)")
	}
	fmt.Printf("FAIL case (over-placement): correctly rejected - %v\n", err)

	setEnforce(false)
	orderID2 := createOrder(100, "BUY", "MARKET", nil)
	if _, err := boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"order_id": orderID2, "account_id": "ACCT-DISC-ACTIVE", "target_qty": 100, "allocated_qty": 0},
	}, "verify_oms_validations"); err != nil {
		log.Fatalf("create matching order_allocation for over-placement isolation: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO orm.placement (id, order_id, broker_id, routed_qty, executed_qty, leaves_qty, status)
		VALUES ($1, $2, 'BRK-ACTIVE', 100, 0, 100, 'NEW')`, uuid.New().String(), orderID2); err != nil {
		log.Fatalf("direct-insert exact placement: %v", err)
	}
	setEnforce(true)
	_, err = boSvc.UpdateBORecord(ctx, secCtx, "order", orderID2, models.BOCrudRecordRequest{
		Record: map[string]interface{}{"status": "FULLY_ROUTED"},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("PASS case (over-placement) unexpectedly rejected: %v", err)
	}
	fmt.Println("PASS case (over-placement): placement_routed_sum == target_qty, update succeeded.")

	// Duplicate-order soft-check (WARN, never blocks): two orders with
	// identical sec_id/side/target_qty/trade_date/manager_id. Created with
	// enforcement off, same reason as everywhere else a bare order (no
	// allocation yet) is created in this file - allocation-completeness/
	// account-compliance are pre-existing BLOCK rules that fail on any
	// order with no allocation, and WARN-severity violations are recorded
	// identically regardless of the enforcement flag (WARN never blocks
	// either way), so enforcement being off doesn't weaken this test.
	setEnforce(false)
	firstID := createOrder(77, "SELL", "MARKET", nil)
	secondID := createOrder(77, "SELL", "MARKET", nil)
	if !hasViolationNamed("order", secondID, "Duplicate order soft-check") {
		log.Fatalf("FAIL case (duplicate order): expected a WARN violation on the second identical order, found none")
	}
	fmt.Printf("FAIL case (duplicate order): second order (%s) correctly flagged as a duplicate of %s (WARN, not blocked).\n", secondID, firstID)

	// A run-unique target_qty (not a fixed literal like 77/78 above) so
	// this assertion holds no matter how many times this script has run
	// against the same database before - a fixed value would eventually
	// collide with a PRIOR run's own "non-duplicate" order and start
	// failing this exact check, the irony of a duplicate-detection test
	// not being idempotent about its own non-duplicate fixture.
	uniqueTargetQty := float64(1000 + time.Now().UnixNano()%100000)
	thirdID := createOrder(uniqueTargetQty, "SELL", "MARKET", nil)
	if hasViolationNamed("order", thirdID, "Duplicate order soft-check") {
		log.Fatalf("PASS case (duplicate order): order with a different target_qty was incorrectly flagged as a duplicate")
	}
	fmt.Println("PASS case (duplicate order): an order with a different target_qty was not flagged.")
}

// ---------- Placement: 5 rules ----------

func testPlacementRules() {
	author("placement", "Routed quantity must be positive",
		models.ValidationRuleSeverityBlock, models.ValidationRuleTimingPreWrite,
		`{"type":"condition","field":"RoutedQuantity","operator":"greater_than","value":0}`)
	author("placement", "Sibling routed quantity must not exceed order target_qty",
		models.ValidationRuleSeverityBlock, models.ValidationRuleTimingPreWrite,
		`{"type":"expression","root":{"op":"<=","left":{"path":"sibling_routed_sum"},"right":{"path":"order_target_qty"}}}`)
	author("placement", "Broker must be ACTIVE",
		models.ValidationRuleSeverityBlock, models.ValidationRuleTimingPreWrite,
		`{"type":"condition","field":"broker_status","operator":"equals","value":"ACTIVE"}`)
	author("placement", "Placement leaves_qty must equal routed_qty minus executed_qty",
		models.ValidationRuleSeverityBlock, models.ValidationRuleTimingReconcile,
		`{"type":"expression","root":{"op":"==","left":{"path":"LeavesQuantity"},"right":{"op":"-","left":{"path":"RoutedQuantity"},"right":{"path":"ExecutedQuantity"}}}}`)
	author("placement", "FixClOrdID must be present once routed",
		models.ValidationRuleSeverityWarn, models.ValidationRuleTimingPreWrite,
		`{"type":"group","operator":"OR","conditions":[
			{"type":"condition","field":"Status","operator":"not_equals","value":"ROUTED"},
			{"type":"expression","root":{"func":"NOT_EMPTY","args":[{"path":"FixClordid"}]}}
		]}`)

	setEnforce(false)
	orderID := createOrder(10000, "BUY", "MARKET", nil)

	// Positive routed: FAIL then PASS.
	setEnforce(true)
	_, err := boSvc.CreateBORecord(ctx, secCtx, "placement", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"order_id": orderID, "broker_id": "BRK-ACTIVE", "routed_qty": 0, "executed_qty": 0, "leaves_qty": 0},
	}, "verify_oms_validations")
	if err == nil {
		log.Fatalf("FAIL case (positive routed): expected rejection for routed_qty=0")
	}
	fmt.Printf("FAIL case (positive routed): correctly rejected - %v\n", err)

	p1, err := boSvc.CreateBORecord(ctx, secCtx, "placement", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"order_id": orderID, "broker_id": "BRK-ACTIVE", "routed_qty": 400, "executed_qty": 0, "leaves_qty": 400},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("PASS case (positive routed) unexpectedly rejected: %v", err)
	}
	p1ID := fmt.Sprintf("%v", p1["id"])
	fmt.Printf("PASS case (positive routed): placement %s created with routed_qty=400.\n", p1ID)

	// Over-placement (placement side): FAIL then PASS (order target_qty=10000, 400 already routed).
	_, err = boSvc.CreateBORecord(ctx, secCtx, "placement", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"order_id": orderID, "broker_id": "BRK-ACTIVE", "routed_qty": 9700, "executed_qty": 0, "leaves_qty": 9700},
	}, "verify_oms_validations")
	if err == nil {
		log.Fatalf("FAIL case (placement over-placement): expected rejection (400+9700 > 10000)")
	}
	fmt.Printf("FAIL case (placement over-placement): correctly rejected - %v\n", err)

	p2, err := boSvc.CreateBORecord(ctx, secCtx, "placement", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"order_id": orderID, "broker_id": "BRK-ACTIVE", "routed_qty": 9600, "executed_qty": 0, "leaves_qty": 9600},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("PASS case (placement over-placement) unexpectedly rejected: %v", err)
	}
	fmt.Printf("PASS case (placement over-placement): placement %v created, 400+9600=10000 == target_qty.\n", p2["id"])

	// Broker active: FAIL then PASS. Order now fully routed (10000), so
	// use a fresh order to leave room. createOrder itself always needs
	// enforcement off - a bare order with no allocation yet always trips
	// the pre-existing allocation-completeness/account-compliance BLOCK
	// rules on Order, unrelated to whatever this section is testing.
	setEnforce(false)
	orderID2 := createOrder(500, "BUY", "MARKET", nil)
	setEnforce(true)
	_, err = boSvc.CreateBORecord(ctx, secCtx, "placement", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"order_id": orderID2, "broker_id": "BRK-INACTIVE", "routed_qty": 100, "executed_qty": 0, "leaves_qty": 100},
	}, "verify_oms_validations")
	if err == nil {
		log.Fatalf("FAIL case (broker active): expected rejection for BRK-INACTIVE")
	}
	fmt.Printf("FAIL case (broker active): correctly rejected - %v\n", err)

	_, err = boSvc.CreateBORecord(ctx, secCtx, "placement", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"order_id": orderID2, "broker_id": "BRK-ACTIVE", "routed_qty": 100, "executed_qty": 0, "leaves_qty": 100},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("PASS case (broker active) unexpectedly rejected: %v", err)
	}
	fmt.Println("PASS case (broker active): BRK-ACTIVE placement succeeded.")

	// Leaves reconciliation: FAIL then PASS.
	setEnforce(false)
	orderID3 := createOrder(500, "BUY", "MARKET", nil)
	setEnforce(true)
	_, err = boSvc.CreateBORecord(ctx, secCtx, "placement", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"order_id": orderID3, "broker_id": "BRK-ACTIVE", "routed_qty": 100, "executed_qty": 0, "leaves_qty": 50},
	}, "verify_oms_validations")
	if err == nil {
		log.Fatalf("FAIL case (placement leaves reconciliation): expected rejection (leaves=50, should be 100)")
	}
	fmt.Printf("FAIL case (placement leaves reconciliation): correctly rejected - %v\n", err)

	_, err = boSvc.CreateBORecord(ctx, secCtx, "placement", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"order_id": orderID3, "broker_id": "BRK-ACTIVE", "routed_qty": 100, "executed_qty": 0, "leaves_qty": 100},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("PASS case (placement leaves reconciliation) unexpectedly rejected: %v", err)
	}
	fmt.Println("PASS case (placement leaves reconciliation): leaves_qty=100=routed-executed, succeeded.")

	// FixClOrdID present when ROUTED: WARN, never blocks.
	setEnforce(false)
	orderID4 := createOrder(500, "BUY", "MARKET", nil)
	setEnforce(true)
	p4, err := boSvc.CreateBORecord(ctx, secCtx, "placement", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"order_id": orderID4, "broker_id": "BRK-ACTIVE", "routed_qty": 100, "executed_qty": 0, "leaves_qty": 100, "status": "ROUTED"},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("ROUTED placement without fix_clordid unexpectedly rejected (rule is WARN, must never block): %v", err)
	}
	p4ID := fmt.Sprintf("%v", p4["id"])
	if !hasViolationNamed("placement", p4ID, "FixClOrdID must be present once routed") {
		log.Fatalf("FAIL case (fix_clordid): expected a WARN violation for a ROUTED placement with no fix_clordid, found none")
	}
	fmt.Println("FAIL case (fix_clordid): ROUTED placement with no fix_clordid correctly logged a WARN (not blocked).")

	p5, err := boSvc.CreateBORecord(ctx, secCtx, "placement", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"order_id": orderID4, "broker_id": "BRK-ACTIVE", "routed_qty": 100, "executed_qty": 0, "leaves_qty": 100, "status": "ROUTED", "fix_clordid": "CLORD-1"},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("PASS case (fix_clordid) unexpectedly rejected: %v", err)
	}
	p5ID := fmt.Sprintf("%v", p5["id"])
	if hasViolationNamed("placement", p5ID, "FixClOrdID must be present once routed") {
		log.Fatalf("PASS case (fix_clordid): a ROUTED placement WITH fix_clordid was incorrectly flagged")
	}
	fmt.Println("PASS case (fix_clordid): ROUTED placement with fix_clordid set produced no violation.")
}

// ---------- Execution: 6 rules ----------

func testExecutionRules() {
	author("execution", "Overfill guard: sibling exec qty must not exceed placement routed_qty",
		models.ValidationRuleSeverityBlock, models.ValidationRuleTimingPreWrite,
		`{"type":"expression","root":{"op":"<=","left":{"path":"sibling_qty_sum"},"right":{"path":"routed_qty"}}}`)
	author("execution", "Execution quantity and price must be positive",
		models.ValidationRuleSeverityBlock, models.ValidationRuleTimingPreWrite,
		`{"type":"group","operator":"AND","conditions":[
			{"type":"condition","field":"ExecQuantity","operator":"greater_than","value":0},
			{"type":"condition","field":"ExecPrice","operator":"greater_than","value":0}
		]}`)
	author("execution", "Execution price must respect the order's limit price",
		models.ValidationRuleSeverityWarn, models.ValidationRuleTimingPreWrite,
		`{"type":"group","operator":"OR","conditions":[
			{"type":"condition","field":"order_limit_price","operator":"is_null"},
			{"type":"group","operator":"AND","conditions":[
				{"type":"group","operator":"OR","conditions":[
					{"type":"condition","field":"order_side","operator":"not_equals","value":"BUY"},
					{"type":"expression","root":{"op":"<=","left":{"path":"ExecPrice"},"right":{"path":"order_limit_price"}}}
				]},
				{"type":"group","operator":"OR","conditions":[
					{"type":"condition","field":"order_side","operator":"not_equals","value":"SELL"},
					{"type":"expression","root":{"op":">=","left":{"path":"ExecPrice"},"right":{"path":"order_limit_price"}}}
				]}
			]}
		]}`)
	author("execution", "Execution time must not precede its placement",
		models.ValidationRuleSeverityWarn, models.ValidationRuleTimingPreWrite,
		`{"type":"condition","field":"causality_ok","operator":"equals","value":true}`)
	author("execution", "LastCapacity must be a recognized value",
		models.ValidationRuleSeverityWarn, models.ValidationRuleTimingPreWrite,
		`{"type":"group","operator":"OR","conditions":[
			{"type":"condition","field":"LastCapacity","operator":"is_null"},
			{"type":"condition","field":"LastCapacity","operator":"equals","value":"P"},
			{"type":"condition","field":"LastCapacity","operator":"equals","value":"A"}
		]}`)
	author("execution", "BrokerExecID must be unique per broker",
		models.ValidationRuleSeverityWarn, models.ValidationRuleTimingPreWrite,
		`{"type":"condition","field":"duplicate_broker_exec_count","operator":"equals","value":0}`)

	setEnforce(false)
	limit := 100.0
	orderID := createOrder(1000, "BUY", "LIMIT", &limit)
	placement, err := boSvc.CreateBORecord(ctx, secCtx, "placement", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"order_id": orderID, "broker_id": "BRK-ACTIVE", "routed_qty": 1000, "executed_qty": 0, "leaves_qty": 1000},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("create placement for execution tests: %v", err)
	}
	placementID := fmt.Sprintf("%v", placement["id"])

	// Overfill guard: FAIL then PASS.
	setEnforce(true)
	_, err = boSvc.CreateBORecord(ctx, secCtx, "execution", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"placement_id": placementID, "order_id": orderID, "exec_qty": 1500, "exec_price": 99, "exec_time": time.Now().UTC(), "transact_time": time.Now().UTC(), "status": "FILLED"},
	}, "verify_oms_validations")
	if err == nil {
		log.Fatalf("FAIL case (overfill guard): expected rejection (1500 > routed_qty 1000)")
	}
	fmt.Printf("FAIL case (overfill guard): correctly rejected - %v\n", err)

	e1, err := boSvc.CreateBORecord(ctx, secCtx, "execution", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"placement_id": placementID, "order_id": orderID, "exec_qty": 600, "exec_price": 99, "exec_time": time.Now().UTC(), "transact_time": time.Now().UTC(), "status": "FILLED", "last_capacity": "P"},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("PASS case (overfill guard) unexpectedly rejected: %v", err)
	}
	fmt.Printf("PASS case (overfill guard): execution %v (qty=600) within routed_qty=1000.\n", e1["id"])

	// Positive qty/price: FAIL then PASS.
	_, err = boSvc.CreateBORecord(ctx, secCtx, "execution", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"placement_id": placementID, "order_id": orderID, "exec_qty": 0, "exec_price": 99, "exec_time": time.Now().UTC(), "transact_time": time.Now().UTC(), "status": "FILLED"},
	}, "verify_oms_validations")
	if err == nil {
		log.Fatalf("FAIL case (positive qty/price): expected rejection for exec_qty=0")
	}
	fmt.Printf("FAIL case (positive qty/price): correctly rejected - %v\n", err)

	e2, err := boSvc.CreateBORecord(ctx, secCtx, "execution", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"placement_id": placementID, "order_id": orderID, "exec_qty": 100, "exec_price": 99.5, "exec_time": time.Now().UTC(), "transact_time": time.Now().UTC(), "status": "FILLED", "last_capacity": "A"},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("PASS case (positive qty/price) unexpectedly rejected: %v", err)
	}
	e2ID := fmt.Sprintf("%v", e2["id"])
	fmt.Printf("PASS case (positive qty/price): execution %s (qty=100, price=99.5) succeeded.\n", e2ID)

	// Price-vs-limit (WARN): BUY order, limit=100. exec_price=105 > limit -> WARN.
	e3, err := boSvc.CreateBORecord(ctx, secCtx, "execution", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"placement_id": placementID, "order_id": orderID, "exec_qty": 50, "exec_price": 105, "exec_time": time.Now().UTC(), "transact_time": time.Now().UTC(), "status": "FILLED"},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("price-vs-limit FAIL case unexpectedly rejected (rule is WARN): %v", err)
	}
	e3ID := fmt.Sprintf("%v", e3["id"])
	if !hasViolationNamed("execution", e3ID, "Execution price must respect the order's limit price") {
		log.Fatalf("FAIL case (price-vs-limit): expected a WARN violation for BUY exec_price=105 > limit_price=100, found none")
	}
	fmt.Println("FAIL case (price-vs-limit): BUY execution above the limit price correctly logged a WARN.")

	e4, err := boSvc.CreateBORecord(ctx, secCtx, "execution", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"placement_id": placementID, "order_id": orderID, "exec_qty": 50, "exec_price": 98, "exec_time": time.Now().UTC(), "transact_time": time.Now().UTC(), "status": "FILLED"},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("PASS case (price-vs-limit) unexpectedly rejected: %v", err)
	}
	e4ID := fmt.Sprintf("%v", e4["id"])
	if hasViolationNamed("execution", e4ID, "Execution price must respect the order's limit price") {
		log.Fatalf("PASS case (price-vs-limit): a BUY execution at/under the limit price was incorrectly flagged")
	}
	fmt.Println("PASS case (price-vs-limit): BUY execution at 98 (<= limit 100) produced no violation.")

	// Causality (WARN): exec_time before the placement's created_at.
	past := time.Now().Add(-48 * time.Hour).UTC()
	e5, err := boSvc.CreateBORecord(ctx, secCtx, "execution", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"placement_id": placementID, "order_id": orderID, "exec_qty": 10, "exec_price": 99, "exec_time": past, "transact_time": past, "status": "FILLED"},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("causality FAIL case unexpectedly rejected (rule is WARN): %v", err)
	}
	e5ID := fmt.Sprintf("%v", e5["id"])
	if !hasViolationNamed("execution", e5ID, "Execution time must not precede its placement") {
		log.Fatalf("FAIL case (causality): expected a WARN violation for exec_time 48h before the placement, found none")
	}
	fmt.Println("FAIL case (causality): an execution timestamped before its placement correctly logged a WARN.")

	e6, err := boSvc.CreateBORecord(ctx, secCtx, "execution", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"placement_id": placementID, "order_id": orderID, "exec_qty": 10, "exec_price": 99, "exec_time": time.Now().UTC(), "transact_time": time.Now().UTC(), "status": "FILLED"},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("PASS case (causality) unexpectedly rejected: %v", err)
	}
	e6ID := fmt.Sprintf("%v", e6["id"])
	if hasViolationNamed("execution", e6ID, "Execution time must not precede its placement") {
		log.Fatalf("PASS case (causality): an execution timestamped after its placement was incorrectly flagged")
	}
	fmt.Println("PASS case (causality): an execution timestamped after its placement produced no violation.")

	// LastCapacity valid (WARN).
	e7, err := boSvc.CreateBORecord(ctx, secCtx, "execution", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"placement_id": placementID, "order_id": orderID, "exec_qty": 5, "exec_price": 99, "exec_time": time.Now().UTC(), "transact_time": time.Now().UTC(), "status": "FILLED", "last_capacity": "X"},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("last_capacity FAIL case unexpectedly rejected (rule is WARN): %v", err)
	}
	e7ID := fmt.Sprintf("%v", e7["id"])
	if !hasViolationNamed("execution", e7ID, "LastCapacity must be a recognized value") {
		log.Fatalf("FAIL case (last_capacity): expected a WARN violation for last_capacity='X', found none")
	}
	fmt.Println("FAIL case (last_capacity): last_capacity='X' correctly logged a WARN.")

	e8, err := boSvc.CreateBORecord(ctx, secCtx, "execution", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"placement_id": placementID, "order_id": orderID, "exec_qty": 5, "exec_price": 99, "exec_time": time.Now().UTC(), "transact_time": time.Now().UTC(), "status": "FILLED", "last_capacity": "P"},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("PASS case (last_capacity) unexpectedly rejected: %v", err)
	}
	e8ID := fmt.Sprintf("%v", e8["id"])
	if hasViolationNamed("execution", e8ID, "LastCapacity must be a recognized value") {
		log.Fatalf("PASS case (last_capacity): last_capacity='P' was incorrectly flagged")
	}
	fmt.Println("PASS case (last_capacity): last_capacity='P' produced no violation.")

	// Duplicate broker exec (WARN): same broker_id + broker_exec_id twice.
	// broker_exec_id values are run-unique (a nanosecond suffix), not
	// fixed literals - a fixed "UNIQUE-EXEC-1" would eventually collide
	// with a PRIOR run's own execution and start failing the PASS
	// assertion below, the same non-idempotency trap the duplicate-order
	// test above already had to be fixed for.
	dupExecID := fmt.Sprintf("DUP-EXEC-%d", time.Now().UnixNano())
	e9, err := boSvc.CreateBORecord(ctx, secCtx, "execution", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"placement_id": placementID, "order_id": orderID, "exec_qty": 5, "exec_price": 99, "exec_time": time.Now().UTC(), "transact_time": time.Now().UTC(), "status": "FILLED", "broker_id": "BRK-ACTIVE", "broker_exec_id": dupExecID},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("create first exec for dup-broker-exec test: %v", err)
	}
	fmt.Printf("Created first execution %v with broker_exec_id=%s.\n", e9["id"], dupExecID)

	e10, err := boSvc.CreateBORecord(ctx, secCtx, "execution", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"placement_id": placementID, "order_id": orderID, "exec_qty": 5, "exec_price": 99, "exec_time": time.Now().UTC(), "transact_time": time.Now().UTC(), "status": "FILLED", "broker_id": "BRK-ACTIVE", "broker_exec_id": dupExecID},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("dup-broker-exec FAIL case unexpectedly rejected (rule is WARN): %v", err)
	}
	e10ID := fmt.Sprintf("%v", e10["id"])
	if !hasViolationNamed("execution", e10ID, "BrokerExecID must be unique per broker") {
		log.Fatalf("FAIL case (dup broker exec): expected a WARN violation for a repeated broker_exec_id, found none")
	}
	fmt.Println("FAIL case (dup broker exec): a second execution with the same broker_id+broker_exec_id correctly logged a WARN.")

	uniqueExecID := fmt.Sprintf("UNIQUE-EXEC-%d", time.Now().UnixNano())
	e11, err := boSvc.CreateBORecord(ctx, secCtx, "execution", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"placement_id": placementID, "order_id": orderID, "exec_qty": 5, "exec_price": 99, "exec_time": time.Now().UTC(), "transact_time": time.Now().UTC(), "status": "FILLED", "broker_id": "BRK-ACTIVE", "broker_exec_id": uniqueExecID},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("PASS case (dup broker exec) unexpectedly rejected: %v", err)
	}
	e11ID := fmt.Sprintf("%v", e11["id"])
	if hasViolationNamed("execution", e11ID, "BrokerExecID must be unique per broker") {
		log.Fatalf("PASS case (dup broker exec): a unique broker_exec_id was incorrectly flagged")
	}
	fmt.Println("PASS case (dup broker exec): a unique broker_exec_id produced no violation.")
}

// ---------- ExecutionAllocation + OrderAllocation: shared chain ----------

func testAllocationRules() {
	author("execution_allocation", "Distributed quantity must not exceed the execution's exec_qty",
		models.ValidationRuleSeverityBlock, models.ValidationRuleTimingPreWrite,
		`{"type":"expression","root":{"op":"<=","left":{"path":"sibling_alloc_sum"},"right":{"path":"parent_exec_qty"}}}`)
	author("execution_allocation", "Allocation price must match the parent execution's price",
		models.ValidationRuleSeverityBlock, models.ValidationRuleTimingPreWrite,
		`{"type":"expression","root":{"op":"==","left":{"path":"AllocationExecPrice"},"right":{"path":"parent_exec_price"}}}`)
	author("execution_allocation", "Allocation must link to the same order as its execution",
		models.ValidationRuleSeverityBlock, models.ValidationRuleTimingPreWrite,
		`{"type":"condition","field":"same_order_ok","operator":"equals","value":true}`)
	author("execution_allocation", "Allocation execution quantity must be positive",
		models.ValidationRuleSeverityBlock, models.ValidationRuleTimingPreWrite,
		`{"type":"condition","field":"AllocationExecQuantity","operator":"greater_than","value":0}`)

	author("order_allocation", "Allocation target quantity must be positive",
		models.ValidationRuleSeverityBlock, models.ValidationRuleTimingPreWrite,
		`{"type":"condition","field":"TargetQuantity","operator":"greater_than","value":0}`)
	author("order_allocation", "Allocation account must be ACTIVE",
		models.ValidationRuleSeverityBlock, models.ValidationRuleTimingPreWrite,
		`{"type":"condition","field":"account_status","operator":"equals","value":"ACTIVE"}`)
	author("order_allocation", "Allocated quantity must equal distributed execution-allocation fills",
		models.ValidationRuleSeverityBlock, models.ValidationRuleTimingReconcile,
		`{"type":"expression","root":{"op":"==","left":{"path":"AllocatedQuantity"},"right":{"path":"alloc_fill_sum"}}}`)

	// --- OrderAllocation: positive target, account active ---
	setEnforce(false)
	orderID := createOrder(1000, "BUY", "MARKET", nil)
	setEnforce(true)

	_, err := boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"order_id": orderID, "account_id": "ACCT-DISC-ACTIVE", "target_qty": 0, "allocated_qty": 0},
	}, "verify_oms_validations")
	if err == nil {
		log.Fatalf("FAIL case (order_allocation positive target): expected rejection for target_qty=0")
	}
	fmt.Printf("FAIL case (order_allocation positive target): correctly rejected - %v\n", err)

	_, err = boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"order_id": orderID, "account_id": "ACCT-OMS-INACTIVE", "target_qty": 100, "allocated_qty": 0},
	}, "verify_oms_validations")
	if err == nil {
		log.Fatalf("FAIL case (order_allocation account active): expected rejection for an INACTIVE account")
	}
	fmt.Printf("FAIL case (order_allocation account active): correctly rejected - %v\n", err)

	oa, err := boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"order_id": orderID, "account_id": "ACCT-DISC-ACTIVE", "target_qty": 1000, "allocated_qty": 0},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("PASS case (order_allocation positive target + account active) unexpectedly rejected: %v", err)
	}
	oaID := fmt.Sprintf("%v", oa["id"])
	fmt.Printf("PASS case (order_allocation positive target + account active): order_allocation %s created.\n", oaID)

	// --- Build a placement/execution to distribute against ---
	placement, err := boSvc.CreateBORecord(ctx, secCtx, "placement", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"order_id": orderID, "broker_id": "BRK-ACTIVE", "routed_qty": 1000, "executed_qty": 0, "leaves_qty": 1000},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("create placement for allocation tests: %v", err)
	}
	placementID := fmt.Sprintf("%v", placement["id"])

	execution, err := boSvc.CreateBORecord(ctx, secCtx, "execution", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"placement_id": placementID, "order_id": orderID, "exec_qty": 300, "exec_price": 50, "exec_time": time.Now().UTC(), "transact_time": time.Now().UTC(), "status": "FILLED", "last_capacity": "P"},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("create execution for allocation tests: %v", err)
	}
	executionID := fmt.Sprintf("%v", execution["id"])

	// A second order, unrelated to orderID, to prove the same-order-linkage rule.
	setEnforce(false)
	otherOrderID := createOrder(1000, "BUY", "MARKET", nil)
	otherOA, err := boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"order_id": otherOrderID, "account_id": "ACCT-DISC-ACTIVE", "target_qty": 1000, "allocated_qty": 0},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("create unrelated order_allocation for linkage test: %v", err)
	}
	otherOAID := fmt.Sprintf("%v", otherOA["id"])
	setEnforce(true)

	// --- ExecutionAllocation: same-order linkage ---
	_, err = boSvc.CreateBORecord(ctx, secCtx, "execution_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"execution_id": executionID, "order_allocation_id": otherOAID, "alloc_exec_qty": 50, "alloc_exec_price": 50},
	}, "verify_oms_validations")
	if err == nil {
		log.Fatalf("FAIL case (same-order linkage): expected rejection - execution belongs to a different order than the order_allocation")
	}
	fmt.Printf("FAIL case (same-order linkage): correctly rejected - %v\n", err)

	// --- ExecutionAllocation: price consistency ---
	_, err = boSvc.CreateBORecord(ctx, secCtx, "execution_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"execution_id": executionID, "order_allocation_id": oaID, "alloc_exec_qty": 50, "alloc_exec_price": 51},
	}, "verify_oms_validations")
	if err == nil {
		log.Fatalf("FAIL case (price consistency): expected rejection - alloc_exec_price 51 != parent exec_price 50")
	}
	fmt.Printf("FAIL case (price consistency): correctly rejected - %v\n", err)

	// --- ExecutionAllocation: positive alloc qty ---
	_, err = boSvc.CreateBORecord(ctx, secCtx, "execution_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"execution_id": executionID, "order_allocation_id": oaID, "alloc_exec_qty": 0, "alloc_exec_price": 50},
	}, "verify_oms_validations")
	if err == nil {
		log.Fatalf("FAIL case (positive alloc qty): expected rejection for alloc_exec_qty=0")
	}
	fmt.Printf("FAIL case (positive alloc qty): correctly rejected - %v\n", err)

	// --- ExecutionAllocation: distribution completeness ---
	_, err = boSvc.CreateBORecord(ctx, secCtx, "execution_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"execution_id": executionID, "order_allocation_id": oaID, "alloc_exec_qty": 400, "alloc_exec_price": 50},
	}, "verify_oms_validations")
	if err == nil {
		log.Fatalf("FAIL case (distribution completeness): expected rejection - 400 > parent execution's exec_qty 300")
	}
	fmt.Printf("FAIL case (distribution completeness): correctly rejected - %v\n", err)

	// --- ExecutionAllocation: everything passing ---
	ea, err := boSvc.CreateBORecord(ctx, secCtx, "execution_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{"execution_id": executionID, "order_allocation_id": oaID, "alloc_exec_qty": 300, "alloc_exec_price": 50},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("PASS case (execution_allocation, all 4 rules) unexpectedly rejected: %v", err)
	}
	fmt.Printf("PASS case (execution_allocation, all 4 rules): execution_allocation %v created (qty=300=exec_qty, price=50=parent, same order).\n", ea["id"])

	// --- OrderAllocation: alloc reconcile (AllocatedQuantity == alloc_fill_sum) ---
	_, err = boSvc.UpdateBORecord(ctx, secCtx, "order_allocation", oaID, models.BOCrudRecordRequest{
		Record: map[string]interface{}{"allocated_qty": 999},
	}, "verify_oms_validations")
	if err == nil {
		log.Fatalf("FAIL case (alloc reconcile): expected rejection - allocated_qty 999 != alloc_fill_sum 300")
	}
	fmt.Printf("FAIL case (alloc reconcile): correctly rejected - %v\n", err)

	_, err = boSvc.UpdateBORecord(ctx, secCtx, "order_allocation", oaID, models.BOCrudRecordRequest{
		Record: map[string]interface{}{"allocated_qty": 300},
	}, "verify_oms_validations")
	if err != nil {
		log.Fatalf("PASS case (alloc reconcile) unexpectedly rejected: %v", err)
	}
	fmt.Println("PASS case (alloc reconcile): allocated_qty=300 matches alloc_fill_sum (300 distributed) - update succeeded.")
}
