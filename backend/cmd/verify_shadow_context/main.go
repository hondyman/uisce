// One-off verification for the related-row context provider on the BO
// write path (docs/unified-rule-engine-handoff.md, open item B).
//
// Authors a synthetic Tier-1-shaped rule - the overfill guard
// (Sigma execution qty for a placement <= the placement's routed
// quantity) - against the Execution BO, persists it through the real
// ValidationRuleService, then drives two real writes through the real
// BusinessObjectService.CreateBORecord path:
//
//  1. A compliant execution (well under the routed quantity) - expect no
//     shadow-violation log line.
//  2. An over-filling execution (pushes the running total over the
//     routed quantity) - expect a shadow-violation log line AND the
//     write to succeed anyway (shadow mode never blocks).
//
// This proves the whole chain: CreateBORecord -> evaluateShadowRules ->
// loadRelatedRowContext (parent placement fields + sibling qty sum) ->
// AdvancedEvaluator, log-only, write unblocked either way.
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

func main() {
	dsn := os.Getenv("DATABASE_URL")
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close()

	ctx := security.WithAuthInfo(context.Background(), security.AuthInfo{
		UserID: "verify_shadow_context", Roles: []string{"global_admin"},
	})
	secCtx := &security.Context{TenantID: tenantID, UserID: "verify_shadow_context"}

	boSvc := metadata.NewBusinessObjectService(db, nil, nil, nil)
	ruleSvc := analytics.NewValidationRuleService(db)

	// 1. Author the overfill-guard rule against the Execution BO.
	ruleAST := json.RawMessage(`{
		"type": "expression",
		"root": {"op": "<=", "left": {"path": "sibling_qty_sum"}, "right": {"path": "quantity"}}
	}`)
	desc, err := ruleSvc.UpsertValidationRule(ctx, models.UpsertValidationRuleRequest{
		TenantID:    tenantID,
		BOName:      "execution",
		Name:        "Overfill Guard (shadow-mode verification)",
		Description: "Sigma execution.qty for a placement <= placement.quantity (routed qty) - Tier-1-shaped synthetic rule proving the related-row context provider",
		Severity:    models.ValidationRuleSeverityBlock,
		Timing:      models.ValidationRuleTimingPreWrite,
		Category:    "capacity",
		RuleAST:     ruleAST,
	})
	if err != nil {
		log.Fatalf("UpsertValidationRule failed: %v", err)
	}
	fmt.Printf("Persisted overfill-guard rule: id=%s bo=%s\n", desc.ID, desc.BOName)

	// 2. Create a fresh placement (order_slice) with routed quantity 100.
	placementRec, err := boSvc.CreateBORecord(ctx, secCtx, "placement", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"parent_order_id": uuid.New().String(),
			"quantity":        100,
		},
	}, "verify_shadow_context")
	if err != nil {
		log.Fatalf("CreateBORecord(placement) failed: %v", err)
	}
	sliceID := fmt.Sprintf("%v", placementRec["id"])
	fmt.Printf("Created placement (order_slice): id=%s quantity=100\n", sliceID)

	// 3. Compliant execution: 60 <= 100. Expect no [SHADOW VIOLATION] line.
	fmt.Println("\n--- Write 1: compliant execution (qty=60, running total=60 <= 100) ---")
	if _, err := boSvc.CreateBORecord(ctx, secCtx, "execution", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"order_id":           uuid.New().String(),
			"slice_id":           sliceID,
			"venue_execution_id": "VERIFY-SHADOW-" + uuid.New().String(),
			"qty":                60,
			"price":              100.0,
			"gross_amount":       6000.0,
			"net_amount":         6000.0,
			"executed_at":        time.Now().Format(time.RFC3339),
		},
	}, "verify_shadow_context"); err != nil {
		log.Fatalf("CreateBORecord(execution #1) failed: %v", err)
	}
	fmt.Println("Write 1 succeeded. Check the log output above/below for [SHADOW VIOLATION] - there should be NONE for this write.")

	// 4. Overfilling execution: 60 + 60 = 120 > 100. Expect a
	// [SHADOW VIOLATION] log line, AND the write must still succeed -
	// that's the entire point of shadow mode.
	fmt.Println("\n--- Write 2: overfilling execution (qty=60, running total=120 > 100) ---")
	result, err := boSvc.CreateBORecord(ctx, secCtx, "execution", models.BOCrudRecordRequest{
		Record: map[string]interface{}{
			"order_id":           uuid.New().String(),
			"slice_id":           sliceID,
			"venue_execution_id": "VERIFY-SHADOW-" + uuid.New().String(),
			"qty":                60,
			"price":              100.0,
			"gross_amount":       6000.0,
			"net_amount":         6000.0,
			"executed_at":        time.Now().Format(time.RFC3339),
		},
	}, "verify_shadow_context")
	if err != nil {
		log.Fatalf("Write 2 FAILED - shadow mode must never block a write: %v", err)
	}
	fmt.Printf("Write 2 succeeded despite the overfill (record id=%v) - shadow mode did not block it, as required.\n", result["id"])
	fmt.Println("Check the log output above for a [SHADOW VIOLATION] line naming the overfill-guard rule - that confirms detection.")
}
