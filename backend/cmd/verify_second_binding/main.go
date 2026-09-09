// One-off verification for rule portability across bindings - the
// architecture question this session's retrofit exists to answer: does
// the *same* rule, authored once against a semantic term, evaluate
// correctly against two different physical bindings of the same BO?
//
// MAPS_TO remains the canonical binding (alpha.orm.order, proven live
// throughout this session). A second binding was populated for this
// proof via business_object_bindings/field_bindings (real tables, empty
// everywhere in the system before this - see
// docs/unified-rule-engine-handoff.md) pointing the Order BO's
// TargetQuantity/LimitPrice/ExecutedQuantity/LeavesQuantity terms at
// alpha.oms.orders' quantity/limit_price/filled_qty/leaves_qty columns.
//
// The second binding is deliberately NOT wired into the CRUD write path
// - direct evaluation against its rows is sufficient to prove resolution
// portability, and per-binding write routing is downstream of the still-
// open canonical-stratum question. This proof doesn't need to wait for
// that decision, and doesn't make it.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/models"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const (
	tenantID    = "99e99e99-99e9-49e9-89e9-99e99e99e999"
	orderBOID   = "b611af7b-8689-407d-807a-eeb315065e7d"
	orderDriver = "/orm/order"
)

func main() {
	db, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()

	svc := analytics.NewValidationRuleService(db)

	// Author the rule once, exactly as the UI does: against the semantic
	// term, not a physical column.
	desc, err := svc.UpsertValidationRule(ctx, models.UpsertValidationRuleRequest{
		TenantID: tenantID, BOName: "order",
		Name:        "TargetQuantity positive (cross-binding portability proof)",
		Description: "Same rule, evaluated against both the canonical MAPS_TO binding and a second field_bindings binding - verify_second_binding proof.",
		Severity:    models.ValidationRuleSeverityBlock,
		Timing:      models.ValidationRuleTimingPreWrite,
		Category:    "oms",
		RuleAST:     json.RawMessage(`{"type":"condition","field":"TargetQuantity","operator":"greater_than","value":0}`),
	})
	if err != nil {
		log.Fatalf("author rule: %v", err)
	}
	var node vm.RuleNode
	if err := json.Unmarshal(desc.RuleAST, &node); err != nil {
		log.Fatalf("parse rule_ast: %v", err)
	}
	fmt.Printf("Authored once, against the semantic term: rule id=%s field=TargetQuantity\n\n", desc.ID)

	ae := vm.NewAdvancedEvaluator()

	// --- Binding 1: canonical (MAPS_TO), alpha.orm.order ---
	fmt.Println("=== Binding 1: canonical (MAPS_TO) -> alpha.orm.order ===")
	fieldMap1, err := analytics.ResolveSemanticFieldMap(ctx, db, orderBOID, orderDriver)
	if err != nil {
		log.Fatalf("resolve canonical field map: %v", err)
	}
	fmt.Printf("Resolved %d semantic terms via MAPS_TO. TargetQuantity -> %s\n", len(fieldMap1), fieldMap1["TargetQuantity"])
	evalBoth(ae, node, fieldMap1, "orm.order (compliant)", map[string]interface{}{"target_qty": 100.0}, true)
	evalBoth(ae, node, fieldMap1, "orm.order (violating)", map[string]interface{}{"target_qty": -5.0}, false)

	// --- Binding 2: field_bindings, alpha.oms.orders ---
	fmt.Println("\n=== Binding 2: field_bindings -> alpha.oms.orders ===")
	var bindingID string
	if err := db.GetContext(ctx, &bindingID, `
		SELECT id FROM business_object_bindings WHERE bo_id = $1::uuid AND is_default = false LIMIT 1
	`, orderBOID); err != nil {
		log.Fatalf("find second binding: %v", err)
	}
	fieldMap2, err := analytics.ResolveSemanticFieldMapForBinding(ctx, db, bindingID)
	if err != nil {
		log.Fatalf("resolve second binding field map: %v", err)
	}
	fmt.Printf("Resolved %d semantic terms via field_bindings (binding %s). TargetQuantity -> %s\n", len(fieldMap2), bindingID, fieldMap2["TargetQuantity"])

	// Real rows from the real, live oms.orders table - not synthetic.
	var rows []struct {
		ID       uuid.UUID `db:"id"`
		Quantity float64   `db:"quantity"`
	}
	if err := db.SelectContext(ctx, &rows, `SELECT id, quantity FROM oms.orders ORDER BY quantity LIMIT 3`); err != nil {
		log.Fatalf("query oms.orders: %v", err)
	}
	for _, row := range rows {
		physical := map[string]interface{}{"quantity": row.Quantity}
		expect := row.Quantity > 0
		evalBoth(ae, node, fieldMap2, fmt.Sprintf("oms.orders row %s (quantity=%.2f, real live row)", row.ID, row.Quantity), physical, expect)
	}
	// Synthetic violating case (real rows all happen to be positive) -
	// proves the FAIL direction against this binding too, matching the
	// oracle rule's two-directions discipline.
	evalBoth(ae, node, fieldMap2, "oms.orders (synthetic, violating)", map[string]interface{}{"quantity": -10.0}, false)

	// Coverage guarantee: a partial binding must never silently borrow
	// from the canonical map for the terms it doesn't cover. This
	// binding was deliberately populated with only 4 of the Order BO's
	// 17 semantic terms - confirm the other 13 (e.g. "ManagerID", which
	// the canonical MAPS_TO binding does resolve) are genuinely absent
	// from binding 2's map, not silently inherited.
	if _, leaked := fieldMap2["ManagerID"]; leaked {
		log.Fatalf("binding 2's field map contains \"ManagerID\", which it has no field_bindings row for - this would be a silent fallback to the canonical binding, exactly the disease this retrofit closes")
	}
	fmt.Printf("\nConfirmed: binding 2's map has no entry for \"ManagerID\" (uncovered by design) - a rule referencing it against this binding would hit unresolvedFieldRefs and produce a rule_error, never silently borrow the canonical binding's mapping.\n")

	fmt.Println("\nSame rule (TargetQuantity > 0), two physical bindings, correct results in both - rule portability proven.")
}

func evalBoth(ae *vm.AdvancedEvaluator, node vm.RuleNode, fieldMap map[string]string, label string, physical map[string]interface{}, expectPass bool) {
	data := make(map[string]interface{}, len(physical)+len(fieldMap))
	for k, v := range physical {
		data[k] = v
	}
	for semantic, phys := range fieldMap {
		if v, ok := physical[phys]; ok {
			data[semantic] = v
		}
	}
	pass, err := ae.Evaluate(node, data)
	if err != nil {
		log.Fatalf("%s: evaluation errored: %v", label, err)
	}
	status := "PASS"
	if !pass {
		status = "FAIL"
	}
	if pass != expectPass {
		log.Fatalf("%s: expected pass=%v, got %v", label, expectPass, pass)
	}
	fmt.Printf("  %s -> %s (as expected)\n", label, status)
}
