package metadata

// Shadow-mode validation-rule evaluation on the BO write path.
//
// Design: log-only from the first line - there is no enforcement to
// preserve, so log-only isn't a phase here, it's the initial state.
// Progressive per-rule enforcement is a later, separate flip once
// shadow-mode violations have been triaged. Everything in this file is
// additive and defensive: a failure here (rule lookup, context loading,
// evaluation) is logged and swallowed, never surfaced to the caller and
// never allowed to affect a write that has already committed.
//
// Related-row context is the simple version deliberately: when a BO has
// any active validation rules, load its immediate related-row aggregates
// unconditionally (sibling rows sharing the same parent, and the parent
// row itself) rather than trying to infer which aggregate a given rule's
// AST references. Scoping the load to only the fields a rule actually
// needs is a real future optimization, not this one.

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/logging"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// relatedRowContext describes, per BO key, how to load the related-row
// aggregate context a Tier-1-shaped rule needs. Deliberately a small,
// explicit table rather than a generic relationship-graph walk - the
// generic version is real future work (see the handoff doc), this is the
// minimum that proves the wiring end-to-end for the BO write path that
// exists today.
type relatedRowContext struct {
	// parentIDField is the column on the written record that points at
	// the parent row (e.g. "slice_id" on an execution row).
	parentIDField string
	// parentTable/parentIDColumn locate the parent row.
	parentTable    string
	parentIDColumn string
	// parentFields are copied from the parent row into the evaluation
	// context verbatim (e.g. the placement's routed quantity).
	parentFields []string
	// siblingTable/siblingSumField locate the numeric column to sum
	// across every row (including the one just written) that shares the
	// same parentIDField value; the result is attached to the context as
	// siblingSumAs. siblingTable is normally the written BO's own
	// physical table.
	siblingTable    string
	siblingSumField string
	siblingSumAs    string
}

// shadowRuleContexts is keyed by bo_key. Only "execution" is wired today
// (the Tier-1 overfill guard: Sigma execution qty per placement <=
// placement.quantity) - the other 4 OMS BOs don't have a working
// driver_table_name yet (see the handoff doc's open item) and shouldn't
// be added here until they do, or this table would silently no-op for
// them forever instead of surfacing that gap.
var shadowRuleContexts = map[string]relatedRowContext{
	"execution": {
		parentIDField:   "slice_id",
		parentTable:     "oms.order_slice",
		parentIDColumn:  "id",
		parentFields:    []string{"quantity", "filled_qty"},
		siblingTable:    "oms.execution",
		siblingSumField: "qty",
		siblingSumAs:    "sibling_qty_sum",
	},
}

// evaluateShadowRules runs every active validation rule for boKey against
// the just-written record, augmented with related-row aggregate context,
// and logs (never blocks on) any violation. Called from CreateBORecord
// and UpdateBORecord after the write has already succeeded.
func (s *BusinessObjectService) evaluateShadowRules(ctx context.Context, tenantID, boKey string, record map[string]interface{}) {
	defer func() {
		if r := recover(); r != nil {
			logging.GetLogger().Sugar().Errorf("shadow rule evaluation panicked for BO %s: %v", boKey, r)
		}
	}()

	svc := analytics.NewValidationRuleService(s.db)
	rules, err := svc.ListByBO(ctx, tenantID, boKey)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("shadow rule evaluation: failed to list rules for BO %s: %v", boKey, err)
		return
	}
	if len(rules) == 0 {
		return
	}

	data := make(map[string]interface{}, len(record)+4)
	for k, v := range record {
		data[k] = v
	}

	if rc, ok := shadowRuleContexts[boKey]; ok {
		s.loadRelatedRowContext(ctx, rc, record, data)
	}

	ae := vm.NewAdvancedEvaluator()
	for _, rule := range rules {
		var node vm.RuleNode
		if err := json.Unmarshal(rule.RuleAST, &node); err != nil {
			logging.GetLogger().Sugar().Warnf("shadow rule %s (%s): rule_ast did not parse: %v", rule.ID, rule.Name, err)
			continue
		}
		pass, err := ae.Evaluate(node, data)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("shadow rule %s (%s) errored during evaluation for BO %s: %v", rule.ID, rule.Name, boKey, err)
			continue
		}
		if !pass {
			// SHADOW MODE: logged only, never blocks - this record has
			// already been written successfully.
			logging.GetLogger().Sugar().Warnf(
				"[SHADOW VIOLATION] rule=%q (%s) bo=%s severity=%s record_id=%v context=%v",
				rule.Name, rule.ID, boKey, rule.Severity, record["id"], data,
			)
		}
	}
}

// loadRelatedRowContext fills data with the parent row's fields and the
// sibling-row aggregate described by rc, reading directly off s.db - the
// same connection CreateBORecord/UpdateBORecord already write through.
func (s *BusinessObjectService) loadRelatedRowContext(ctx context.Context, rc relatedRowContext, record map[string]interface{}, data map[string]interface{}) {
	parentID, ok := record[rc.parentIDField]
	if !ok || parentID == nil {
		return
	}

	if len(rc.parentFields) > 0 {
		cols := ""
		for i, f := range rc.parentFields {
			if i > 0 {
				cols += ", "
			}
			cols += f
		}
		row := make(map[string]interface{})
		query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1", cols, rc.parentTable, rc.parentIDColumn)
		rows, err := s.db.QueryxContext(ctx, query, parentID)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("shadow context: failed to load parent %s: %v", rc.parentTable, err)
			return
		}
		if rows.Next() {
			_ = rows.MapScan(row)
			for k, v := range row {
				data[k] = coerceNumeric(v)
			}
		}
		rows.Close()
	}

	if rc.siblingSumField != "" && rc.siblingTable != "" {
		var sum float64
		query := fmt.Sprintf(
			"SELECT COALESCE(SUM(%s), 0) FROM %s WHERE %s = $1",
			rc.siblingSumField, rc.siblingTable, rc.parentIDField,
		)
		if err := s.db.GetContext(ctx, &sum, query, parentID); err != nil {
			logging.GetLogger().Sugar().Warnf("shadow context: failed to load sibling sum: %v", err)
			return
		}
		data[rc.siblingSumAs] = sum
	}
}

// coerceNumeric converts the []byte lib/pq returns for Postgres numeric
// columns (numeric isn't one of its natively-typed scan targets) into a
// float64, so it lands in the evaluation context as something
// AdvancedEvaluator's arithmetic operators recognize as numeric rather
// than as an opaque byte slice. Non-numeric-looking values pass through
// unchanged.
func coerceNumeric(v interface{}) interface{} {
	b, ok := v.([]byte)
	if !ok {
		return v
	}
	if f, err := strconv.ParseFloat(string(b), 64); err == nil {
		return f
	}
	return string(b)
}
