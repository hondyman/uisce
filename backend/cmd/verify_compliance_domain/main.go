// One-off proof for the rulefabric consolidation's "fold, don't bridge"
// decision (see docs/unified-rule-engine-handoff.md, "Rulefabric
// consolidation"): a domain="compliance" rule, authored through the same
// ValidationRuleService every validation rule already uses, blocks a real
// write through the same write path (shadow_evaluation.go's
// evaluateAndEnforceRules), with no separate rulefabric write-hook
// involved at all.
//
// Targets the order_allocation BO specifically, not order, to keep this
// proof isolated from a known, separate, already-documented issue: a
// pre-existing tenant-wide "Placement routed quantity..." rule on the
// order BO currently rule-errors on this branch (its field loader,
// placement_routed_sum, was only ever added on feat/calc-engine-measures)
// - that's unrelated to this proof and would contaminate attribution if
// the write under test were on the order BO.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
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
		UserID: "verify_compliance_domain", Roles: []string{"global_admin"},
	})
	secCtx := &security.Context{TenantID: tenantID, UserID: "verify_compliance_domain"}
	boSvc := metadata.NewBusinessObjectService(db, nil, nil, nil)
	ruleSvc := analytics.NewValidationRuleService(db)

	// 1. Parent order, enforcement OFF - the pre-existing placement-rule
	// rule_error (unrelated to this proof) shadow-logs but doesn't block.
	setEnforce(false)
	order, err := boSvc.CreateBORecord(ctx, secCtx, "order", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"sec_id": 1, "side": "BUY", "order_type": "MARKET",
			"target_qty": 5000, "executed_qty": 0, "leaves_qty": 5000,
			"trade_date": time.Now().Format("2006-01-02"),
		},
	}, "verify_compliance_domain")
	if err != nil {
		log.Fatalf("parent order creation failed unexpectedly: %v", err)
	}
	orderID := fmt.Sprintf("%v", order["id"])
	fmt.Printf("Created parent order %s (target_qty=5000)\n", orderID)

	// 2. Author a domain="compliance" BLOCK rule on order_allocation -
	// a concentration-style threshold (target_qty must not exceed 1000).
	desc, err := ruleSvc.UpsertValidationRule(ctx, models.UpsertValidationRuleRequest{
		TenantID:    tenantID,
		BOName:      "order_allocation",
		Name:        "Allocation concentration threshold (compliance proof)",
		Description: "verify_compliance_domain proof rule - domain=compliance",
		Severity:    models.ValidationRuleSeverityBlock,
		Timing:      models.ValidationRuleTimingPreWrite,
		Category:    "concentration",
		Domain:      models.ValidationRuleDomainCompliance,
		RuleAST:     json.RawMessage(`{"type":"condition","field":"target_qty","operator":"less_equal","value":1000}`),
	})
	if err != nil {
		log.Fatalf("author compliance-domain rule: %v", err)
	}
	fmt.Printf("Authored domain=%q rule %q: id=%s severity=%s\n", desc.Domain, desc.Name, desc.ID, desc.Severity)
	if desc.Domain != models.ValidationRuleDomainCompliance {
		log.Fatalf("round-trip mismatch: authored domain=%q, read back domain=%q", models.ValidationRuleDomainCompliance, desc.Domain)
	}

	// 3. Violating write, enforcement ON -> expect BLOCK, row absent.
	setEnforce(true)
	before := countAllocations(db)
	violatingID := uuid.New().String()
	_, err = boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"id": violatingID, "order_id": orderID, "account_id": "ACCT-DISC-ACTIVE",
			"target_qty": 5000, "allocated_qty": 0,
		},
	}, "verify_compliance_domain")
	after := countAllocations(db)
	if err == nil {
		log.Fatalf("expected the compliance-domain rule to reject target_qty=5000 (> 1000), but it succeeded")
	}
	if after != before {
		log.Fatalf("write was rejected (err=%v) but order_allocation row count still changed: %d -> %d", err, before, after)
	}
	fmt.Printf("[PASS] Violating write correctly rejected via the real write path (no rulefabric/TriggerEngine involved): %v\n", err)

	violations, err := analytics.ListViolations(ctx, db, tenantID, "order_allocation", 500)
	if err != nil {
		log.Fatalf("list violations: %v", err)
	}
	found := false
	for _, v := range violations {
		if v.RecordID == violatingID && v.WriteBlocked && v.RuleName == desc.Name {
			found = true
		}
	}
	if !found {
		log.Fatalf("no write_blocked=true violation found attributing the rejection to rule %q", desc.Name)
	}
	fmt.Printf("[PASS] Violation persisted and attributed to the compliance-domain rule by name.\n")

	// 4. Compliant value (target_qty=100, <= 1000), enforcement ON.
	// order_allocation carries a separate, pre-existing, already-documented
	// rule ("Allocated quantity must equal distributed execution-allocation
	// fills") that references a context field (alloc_fill_sum) this
	// branch's loader never populates - the same class of gap as
	// placement_routed_sum on the order BO - so it rule-errors and blocks
	// *every* order_allocation write on this branch unconditionally,
	// regardless of this proof. That's a real, separate, already-known
	// issue, not something this proof can or should route around by
	// faking context data. What this step can still prove cleanly: the
	// compliance-domain rule itself correctly discriminates - it must be
	// ABSENT from the blocking-rule list this time, even though the
	// write still fails for the unrelated reason.
	compliantID := uuid.New().String()
	_, err = boSvc.CreateBORecord(ctx, secCtx, "order_allocation", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"id": compliantID, "order_id": orderID, "account_id": "ACCT-DISC-ACTIVE",
			"target_qty": 100, "allocated_qty": 0,
		},
	}, "verify_compliance_domain")
	if err == nil {
		log.Fatalf("expected this write to still fail (the unrelated alloc_fill_sum rule-error blocks every order_allocation write on this branch), but it succeeded - that unrelated bug may have been fixed, which would change this proof's shape")
	}
	if contains(err.Error(), desc.Name) {
		log.Fatalf("compliance-domain rule appeared in the blocking list for a COMPLIANT value (target_qty=100 <= 1000) - it does not discriminate correctly: %v", err)
	}
	fmt.Printf("[PASS] Compliant value (target_qty=100) correctly did NOT trigger the compliance-domain rule "+
		"(write still failed for the separate, unrelated, pre-existing alloc_fill_sum issue): %v\n", err)

	fmt.Println("\nAll checks passed: a domain=compliance rule, authored through ValidationRuleService, " +
		"blocks a real write and correctly discriminates compliant from non-compliant values through " +
		"shadow_evaluation.go's write path - no rulefabric write-hook involved.")
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

func countAllocations(db *sqlx.DB) int {
	var n int
	if err := db.Get(&n, `SELECT count(*) FROM orm.order_allocation`); err != nil {
		log.Fatalf("count order_allocation: %v", err)
	}
	return n
}
