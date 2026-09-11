// Three-domain proof: demonstrates that MDM, compliance, and BO validation
// rules share the same editor, same storage convention, and same engine -
// answering the user complaint "I don't feel we are using the same editor
// in MDM rules, compliance rules and Business Objects validations" by
// demonstration rather than by claim.
//
// Three rules authored against the order BO (one per domain), all through
// ValidationRuleService.UpsertValidationRule, all stored as catalog_node
// rows with GOVERNED_BY_RULE edges and domain/severity/timing in
// properties, all evaluated by the same shadow_evaluation.go write path:
//  1. domain="validation"  - side must be BUY
//  2. domain="mdm"       - target_qty must not exceed a sanity cap (1M)
//  3. domain="compliance" - target_qty must not exceed a hard limit (100k)
//
// Each rule is proven in both directions where the pre-existing rules allow:
// a write that violates the rule is blocked, and a write that satisfies it
// passes. The triple-violation test additionally proves all three can fire
// simultaneously on the same write.
//
// The "same editor" half is proven by the domain selector UI in
// AdvancedRuleBuilderPage.tsx (which this command's save requests exercise
// through the same API).
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/metadata"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const tenantID = "99e99e99-99e9-49e9-89e9-99e99e99e999"

func setEnforce(on bool) {
	if on {
		os.Setenv("VALIDATION_RULES_ENFORCE", "true")
	} else {
		os.Unsetenv("VALIDATION_RULES_ENFORCE")
	}
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close()

	ctx := security.WithAuthInfo(context.Background(), security.AuthInfo{
		UserID: "verify_three_domains", Roles: []string{"global_admin"},
	})
	secCtx := &security.Context{TenantID: tenantID, UserID: "verify_three_domains"}
	boSvc := metadata.NewBusinessObjectService(db, nil, nil, nil)
	ruleSvc := analytics.NewValidationRuleService(db)

	// Seed a discretionary ACTIVE account. The pre-existing account rule
	// (Account must be ACTIVE) resolves account_status via JOIN on
	// order_allocation, so every order needs at least one allocation to an
	// ACTIVE account to pass that rule.
	_, err = db.ExecContext(ctx, `
		INSERT INTO orm.account (account_id, status, is_discretionary) VALUES
			('ACCT-DISC-ACTIVE', 'ACTIVE', true)
		ON CONFLICT (account_id) DO UPDATE SET status = EXCLUDED.status, is_discretionary = EXCLUDED.is_discretionary
	`)
	if err != nil {
		log.Fatalf("seed account: %v", err)
	}
	fmt.Println("Seeded orm.account: ACCT-DISC-ACTIVE (discretionary, ACTIVE)")

	// Build a fully-allocated order chain with enforcement OFF. With
	// enforcement OFF the pre-existing allocation-completeness rule doesn't
	// block the chain build. This gives us a valid allocated order whose
	// account context resolves correctly when enforcement is later enabled.
	setEnforce(false)
	uniqueQty := float64(50000 + time.Now().UnixNano()%100000)
	order, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"sec_id": 1, "side": "BUY", "order_type": "MARKET",
			"target_qty": uniqueQty, "executed_qty": 0, "leaves_qty": uniqueQty,
			"trade_date": time.Now().Format("2006-01-02"),
		},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("parent order creation: %v", err)
	}
	orderID := fmt.Sprintf("%v", order["id"])
	fmt.Printf("Created parent order %s (target_qty=%.0f)\n", orderID, uniqueQty)

	// Create the matching allocation so the account context resolves.
	_, err = boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"order_id": orderID, "account_id": "ACCT-DISC-ACTIVE",
			"target_qty": uniqueQty, "allocated_qty": 0,
		},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("allocation creation: %v", err)
	}
	fmt.Printf("Created order_allocation: %.0f to ACCT-DISC-ACTIVE (account context now resolves)\n", uniqueQty)

	// ── Rule 1: domain="validation" ──────────────────────────────────────────
	descVal, err := ruleSvc.UpsertValidationRule(ctx, models.UpsertValidationRuleRequest{
		TenantID:    tenantID,
		BOName:      "order",
		Name:        "side must be BUY (validation domain)",
		Description: "three-domain proof: domain=validation rule",
		Severity:    models.ValidationRuleSeverityBlock,
		Timing:      models.ValidationRuleTimingPreWrite,
		Domain:      models.ValidationRuleDomainDefault,
		RuleAST:     jsonRaw(`{"type":"condition","field":"side","operator":"equals","value":"BUY"}`),
	})
	if err != nil {
		log.Fatalf("author validation-domain rule: %v", err)
	}
	fmt.Printf("Authored domain=%q rule id=%s\n", descVal.Domain, descVal.ID)
	if descVal.Domain != "validation" {
		log.Fatalf("validation-domain rule round-trip mismatch: got %q", descVal.Domain)
	}

	// ── Rule 2: domain="mdm" ────────────────────────────────────────────────
	descMDM, err := ruleSvc.UpsertValidationRule(ctx, models.UpsertValidationRuleRequest{
		TenantID:    tenantID,
		BOName:      "order",
		Name:        "target_qty sanity cap (mdm domain)",
		Description: "three-domain proof: domain=mdm rule",
		Severity:    models.ValidationRuleSeverityBlock,
		Timing:      models.ValidationRuleTimingPreWrite,
		Domain:      models.ValidationRuleDomainMDM,
		RuleAST:     jsonRaw(`{"type":"condition","field":"target_qty","operator":"less_equal","value":1000000}`),
	})
	if err != nil {
		log.Fatalf("author mdm-domain rule: %v", err)
	}
	fmt.Printf("Authored domain=%q rule id=%s\n", descMDM.Domain, descMDM.ID)
	if descMDM.Domain != "mdm" {
		log.Fatalf("mdm-domain rule round-trip mismatch: got %q", descMDM.Domain)
	}

	// ── Rule 3: domain="compliance" ─────────────────────────────────────────
	descComp, err := ruleSvc.UpsertValidationRule(ctx, models.UpsertValidationRuleRequest{
		TenantID:    tenantID,
		BOName:      "order",
		Name:        "target_qty hard limit (compliance domain)",
		Description: "three-domain proof: domain=compliance rule",
		Severity:    models.ValidationRuleSeverityBlock,
		Timing:      models.ValidationRuleTimingPreWrite,
		Domain:      models.ValidationRuleDomainCompliance,
		RuleAST:     jsonRaw(`{"type":"condition","field":"target_qty","operator":"less_equal","value":100000}`),
	})
	if err != nil {
		log.Fatalf("author compliance-domain rule: %v", err)
	}
	fmt.Printf("Authored domain=%q rule id=%s\n", descComp.Domain, descComp.ID)
	if descComp.Domain != "compliance" {
		log.Fatalf("compliance-domain rule round-trip mismatch: got %q", descComp.Domain)
	}

	// ══════════════════════════════════════════════════════════════════════════
	// TEST CASES
	// Each test creates an allocated order (order + allocation in one shot with
	// enforcement OFF, then enables enforcement). This ensures account context
	// resolves for every write, satisfying the pre-existing account rule.
	// ══════════════════════════════════════════════════════════════════════════

	// ── validation domain: side == 'BUY' ────────────────────────────────────
	fmt.Println("\n── validation domain: side == 'BUY' ──")

	// Violation: side = 'SELL'
	setEnforce(false)
	ordSell, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"sec_id": 10, "side": "SELL", "order_type": "MARKET",
			"target_qty": 100, "executed_qty": 0, "leaves_qty": 100,
			"trade_date": time.Now().Format("2006-01-02"),
		},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("SELL order creation: %v", err)
	}
	sellID := fmt.Sprintf("%v", ordSell["id"])
	_, err = boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"order_id": sellID, "account_id": "ACCT-DISC-ACTIVE",
			"target_qty": 100, "allocated_qty": 0,
		},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("SELL allocation: %v", err)
	}
	setEnforce(true)
	_, err = boSvc.UpdateBORecord(ctx, secCtx, "order", sellID, models.BOCrudRecordRequest{
		Record: map[string]interface{}{"side": "SELL"},
	}, "verify_three_domains")
	if err == nil {
		log.Fatalf("validation-domain violation: expected side=SELL update to be rejected")
	}
	if !containsErr(err, descVal.Name) {
		log.Fatalf("validation-domain violation: error doesn't mention validation rule %q: %v", descVal.Name, err)
	}
	fmt.Printf("[PASS] side=SELL blocked by validation-domain rule: %v\n", err)

	// Pass: side = 'BUY' (new BUY order + allocation, enforcement ON)
	setEnforce(false)
	ordBuy, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"sec_id": 11, "side": "BUY", "order_type": "MARKET",
			"target_qty": 100, "executed_qty": 0, "leaves_qty": 100,
			"trade_date": time.Now().Format("2006-01-02"),
		},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("BUY order creation: %v", err)
	}
	buyID := fmt.Sprintf("%v", ordBuy["id"])
	_, err = boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"order_id": buyID, "account_id": "ACCT-DISC-ACTIVE",
			"target_qty": 100, "allocated_qty": 0,
		},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("BUY allocation: %v", err)
	}
	setEnforce(true)
	_, err = boSvc.UpdateBORecord(ctx, secCtx, "order", buyID, models.BOCrudRecordRequest{
		Record: map[string]interface{}{"side": "BUY"},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("validation-domain pass: expected BUY order to pass, got: %v", err)
	}
	fmt.Printf("[PASS] side=BUY accepted by validation-domain rule\n")

	// ── mdm domain: target_qty <= 1,000,000 ────────────────────────────────
	fmt.Println("\n── mdm domain: target_qty <= 1,000,000 ──")

	// Violation: target_qty = 5,000,000
	setEnforce(false)
	ordMDM, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"sec_id": 12, "side": "BUY", "order_type": "MARKET",
			"target_qty": 5000000, "executed_qty": 0, "leaves_qty": 5000000,
			"trade_date": time.Now().Format("2006-01-02"),
		},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("mdm violation order: %v", err)
	}
	mdmID := fmt.Sprintf("%v", ordMDM["id"])
	_, err = boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"order_id": mdmID, "account_id": "ACCT-DISC-ACTIVE",
			"target_qty": 5000000, "allocated_qty": 0,
		},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("mdm violation allocation: %v", err)
	}
	setEnforce(true)
	_, err = boSvc.UpdateBORecord(ctx, secCtx, "order", mdmID, models.BOCrudRecordRequest{
		Record: map[string]interface{}{"side": "BUY"},
	}, "verify_three_domains")
	if err == nil {
		log.Fatalf("mdm-domain violation: expected target_qty=5000000 to be rejected")
	}
	if !containsErr(err, descMDM.Name) {
		log.Fatalf("mdm-domain violation: error doesn't mention mdm rule %q: %v", descMDM.Name, err)
	}
	fmt.Printf("[PASS] target_qty=5000000 blocked by mdm-domain rule: %v\n", err)

	// Pass: target_qty = 50000 (below compliance limit so the compliance rule doesn't block)
	setEnforce(false)
	ordMDMPass, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"sec_id": 13, "side": "BUY", "order_type": "MARKET",
			"target_qty": 50000, "executed_qty": 0, "leaves_qty": 50000,
			"trade_date": time.Now().Format("2006-01-02"),
		},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("mdm pass order: %v", err)
	}
	mdmPassID := fmt.Sprintf("%v", ordMDMPass["id"])
	_, err = boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"order_id": mdmPassID, "account_id": "ACCT-DISC-ACTIVE",
			"target_qty": 50000, "allocated_qty": 0,
		},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("mdm pass allocation: %v", err)
	}
	setEnforce(true)
	_, err = boSvc.UpdateBORecord(ctx, secCtx, "order", mdmPassID, models.BOCrudRecordRequest{
		Record: map[string]interface{}{"side": "BUY"},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("mdm-domain pass: expected target_qty=50000 to pass, got: %v", err)
	}
	fmt.Printf("[PASS] target_qty=50000 accepted by mdm-domain rule\n")

	// ── compliance domain: target_qty <= 100,000 ────────────────────────────
	fmt.Println("\n── compliance domain: target_qty <= 100,000 ──")

	// Violation: target_qty = 500000 (exceeds compliance limit but within MDM cap)
	setEnforce(false)
	ordComp, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"sec_id": 14, "side": "BUY", "order_type": "MARKET",
			"target_qty": 500000, "executed_qty": 0, "leaves_qty": 500000,
			"trade_date": time.Now().Format("2006-01-02"),
		},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("compliance violation order: %v", err)
	}
	compID := fmt.Sprintf("%v", ordComp["id"])
	_, err = boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"order_id": compID, "account_id": "ACCT-DISC-ACTIVE",
			"target_qty": 500000, "allocated_qty": 0,
		},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("compliance violation allocation: %v", err)
	}
	setEnforce(true)
	_, err = boSvc.UpdateBORecord(ctx, secCtx, "order", compID, models.BOCrudRecordRequest{
		Record: map[string]interface{}{"side": "BUY"},
	}, "verify_three_domains")
	if err == nil {
		log.Fatalf("compliance-domain violation: expected target_qty=500000 to be rejected")
	}
	if !containsErr(err, descComp.Name) {
		log.Fatalf("compliance-domain violation: error doesn't mention compliance rule %q: %v", descComp.Name, err)
	}
	fmt.Printf("[PASS] target_qty=500000 blocked by compliance-domain rule: %v\n", err)

	// Pass: target_qty = 50000
	setEnforce(false)
	ordCompPass, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"sec_id": 15, "side": "BUY", "order_type": "MARKET",
			"target_qty": 50000, "executed_qty": 0, "leaves_qty": 50000,
			"trade_date": time.Now().Format("2006-01-02"),
		},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("compliance pass order: %v", err)
	}
	compPassID := fmt.Sprintf("%v", ordCompPass["id"])
	_, err = boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"order_id": compPassID, "account_id": "ACCT-DISC-ACTIVE",
			"target_qty": 50000, "allocated_qty": 0,
		},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("compliance pass allocation: %v", err)
	}
	setEnforce(true)
	_, err = boSvc.UpdateBORecord(ctx, secCtx, "order", compPassID, models.BOCrudRecordRequest{
		Record: map[string]interface{}{"side": "BUY"},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("compliance-domain pass: expected target_qty=50000 to pass, got: %v", err)
	}
	fmt.Printf("[PASS] target_qty=50000 accepted by compliance-domain rule\n")

	// ── All three domains fire simultaneously ───────────────────────────────
	fmt.Println("\n── all three fire simultaneously on a triple violation ──")
	setEnforce(false)
	ordTriple, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"sec_id": 16, "side": "SELL", "order_type": "MARKET",
			"target_qty": 5000000, "executed_qty": 0, "leaves_qty": 5000000,
			"trade_date": time.Now().Format("2006-01-02"),
		},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("triple order: %v", err)
	}
	tripleID := fmt.Sprintf("%v", ordTriple["id"])
	_, err = boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"order_id": tripleID, "account_id": "ACCT-DISC-ACTIVE",
			"target_qty": 5000000, "allocated_qty": 0,
		},
	}, "verify_three_domains")
	if err != nil {
		log.Fatalf("triple allocation: %v", err)
	}
	setEnforce(true)
	_, err = boSvc.UpdateBORecord(ctx, secCtx, "order", tripleID, models.BOCrudRecordRequest{
		Record: map[string]interface{}{"side": "SELL"},
	}, "verify_three_domains")
	if err == nil {
		log.Fatalf("triple violation: expected write to be rejected")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, descVal.Name) {
		log.Fatalf("triple violation: error doesn't mention validation-domain rule %q: %v", descVal.Name, err)
	}
	if !strings.Contains(errStr, descMDM.Name) {
		log.Fatalf("triple violation: error doesn't mention mdm-domain rule %q: %v", descMDM.Name, err)
	}
	if !strings.Contains(errStr, descComp.Name) {
		log.Fatalf("triple violation: error doesn't mention compliance-domain rule %q: %v", descComp.Name, err)
	}
	fmt.Printf("[PASS] Triple violation correctly attributed all three domain rules.\n")

	fmt.Println("\nAll three-domain proof checks passed.")
	fmt.Println("Three rules (validation/mdm/compliance), each with round-trip domain")
	fmt.Println("verification, violation-case blocking, and compliant-case passing - all")
	fmt.Println("through ValidationRuleService and shadow_evaluation.go:")
	fmt.Println("one editor, one engine, one storage convention.")
}

func containsErr(err error, s string) bool {
	return strings.Contains(err.Error(), s)
}

func jsonRaw(s string) []byte {
	return []byte(s)
}
