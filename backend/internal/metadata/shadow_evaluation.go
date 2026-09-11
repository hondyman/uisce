package metadata

// Validation-rule evaluation on the BO write path.
//
// Enforcement is a flag, not a mode: VALIDATION_RULES_ENFORCE=true turns
// it on; unset (the default) is shadow mode - every rule still runs and
// every violation is still logged and persisted, but nothing is
// rejected. This is legitimate as real enforcement (not just logging)
// specifically because these are platform-owned writes to a
// platform-owned local store (see docs/unified-rule-engine-handoff.md,
// "local orm schema") - the "can't block a write you don't make"
// constraint from the CDC-sourced strata doesn't apply here.
//
// With enforcement on, a BLOCK-severity violation rolls back the write's
// transaction before it commits; WARN-severity violations (and BLOCK
// violations while enforcement is off) are logged and persisted but
// never block. Every evaluated rule's outcome that fails is persisted to
// validation_rule_violations (see violations.go) regardless of whether
// the write itself was rejected, so "did this rule ever fire" survives
// past the log line.
//
// Related-row context is still the simple version deliberately: BOs with
// richer context needs (Order: allocation-sum, account lookup) get a
// dedicated loader function rather than forcing the generic
// parent+sibling shape to stretch to fit; BOs with the simple shape
// (Execution: parent placement + sibling exec-qty sum) use the generic
// relatedRowContext table. Neither is a general relationship-graph walk.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/models"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
	"github.com/jmoiron/sqlx"
)

// dbExecutor is satisfied by both *sqlx.DB and *sqlx.Tx - evaluation runs
// inside the same transaction as the write it's evaluating (so aggregate
// queries see the not-yet-committed row), while violation persistence
// runs against s.db directly (so a violation record survives even when
// the transaction it was found in gets rolled back).
type dbExecutor interface {
	QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error)
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

// enforcementEnabled reports whether BLOCK-severity violations should
// actually reject a write. Checked live (not cached) - this is a
// dev/proof toggle, not a hot path.
func enforcementEnabled() bool {
	return os.Getenv("VALIDATION_RULES_ENFORCE") == "true"
}

// ruleViolation is one failed rule evaluation, ready to log and persist.
// RuleError distinguishes "the rule ran and found a real violation" from
// "the rule couldn't run at all" (an unresolvable field reference, a
// malformed rule_ast) - the second case must never be a silent skip. The
// recurring failure mode across this whole engagement has been exactly
// that: a rule that looks wired up but silently never fires (stale
// bo_name, vacuous AND/OR, a schema the rule no longer matches). A rule
// error is persisted as a violation like any other, just tagged so it's
// not mistaken for "the data passed."
type ruleViolation struct {
	RuleID    string
	RuleName  string
	Severity  string
	Message   string
	Context   map[string]interface{}
	RuleError bool
}

// relatedRowContext describes, per BO key, how to load the simple-shape
// related-row aggregate context a rule needs: one parent row plus one
// sibling-row sum. See loadOrderContext for the BO that needs more than
// this shape provides.
type relatedRowContext struct {
	parentIDField   string
	parentTable     string
	parentIDColumn  string
	parentFields    []string
	siblingTable    string
	siblingSumField string
	siblingSumAs    string
}

var shadowRuleContexts = map[string]relatedRowContext{
	"execution": {
		parentIDField:   "placement_id",
		parentTable:     "orm.placement",
		parentIDColumn:  "id",
		parentFields:    []string{"routed_qty", "executed_qty"},
		siblingTable:    "orm.execution",
		siblingSumField: "exec_qty",
		siblingSumAs:    "sibling_qty_sum",
	},
}

// writeAndEnforce runs doWrite inside a transaction, evaluates every
// active validation rule for boKey against the row it produced (seeing
// the not-yet-committed write, via the same transaction), and either
// commits (no BLOCK violation, or enforcement is off) or rolls back (a
// BLOCK violation with enforcement on). Every violation found is
// persisted to validation_rule_violations afterward regardless of
// outcome, on s.db rather than the transaction, so the record survives a
// rollback. Returns an error naming the blocking rule(s) if the write
// was rejected.
func (s *BusinessObjectService) writeAndEnforce(ctx context.Context, tenantID string, bo *models.BusinessObjectDefinition, doWrite func(tx *sqlx.Tx) (map[string]interface{}, error)) (map[string]interface{}, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	result, err := doWrite(tx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	violations, blocked := s.evaluateAndEnforceRules(ctx, tx, tenantID, bo, result)

	if blocked {
		_ = tx.Rollback()
	} else if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	boKey := bo.Key
	recordID := fmt.Sprintf("%v", result["id"])
	for _, v := range violations {
		writeBlocked := blocked && v.Severity == "BLOCK"
		rec := analytics.ViolationRecord{
			TenantID: tenantID, RuleID: v.RuleID, RuleName: v.RuleName, BOKey: boKey,
			Severity: v.Severity, RecordID: recordID, Message: v.Message,
			Context: v.Context, WriteBlocked: writeBlocked, RuleError: v.RuleError,
		}
		if perr := analytics.PersistViolation(ctx, s.db, rec); perr != nil {
			logging.GetLogger().Sugar().Warnf("failed to persist violation for rule %s: %v", v.RuleID, perr)
		}
		if writeBlocked {
			logging.GetLogger().Sugar().Warnf("[WRITE REJECTED] rule=%q (%s) bo=%s severity=%s record_id=%s: %s",
				v.RuleName, v.RuleID, boKey, v.Severity, recordID, v.Message)
		} else {
			tag := "VIOLATION"
			if !enforcementEnabled() {
				tag = "VIOLATION - SHADOW"
			}
			logging.GetLogger().Sugar().Warnf("[%s] rule=%q (%s) bo=%s severity=%s record_id=%s: %s",
				tag, v.RuleName, v.RuleID, boKey, v.Severity, recordID, v.Message)
		}
	}

	if blocked {
		var names []string
		for _, v := range violations {
			if v.Severity == "BLOCK" {
				names = append(names, v.RuleName)
			}
		}
		return nil, fmt.Errorf("write rejected by validation rule(s): %s", strings.Join(names, "; "))
	}

	return result, nil
}

// evaluateAndEnforceRules runs every active validation rule for boKey
// against the just-written record (augmented with related-row context),
// using exec (normally the write's own transaction) for every read. It
// never itself commits or rolls back anything - the caller decides what
// to do with the returned violations. Panics are recovered so a bug here
// can never propagate into the write path.
func (s *BusinessObjectService) evaluateAndEnforceRules(ctx context.Context, exec dbExecutor, tenantID string, bo *models.BusinessObjectDefinition, record map[string]interface{}) (violations []ruleViolation, blocked bool) {
	boKey := bo.Key
	defer func() {
		if r := recover(); r != nil {
			logging.GetLogger().Sugar().Errorf("rule evaluation panicked for BO %s: %v", boKey, r)
			violations = nil
			blocked = false
		}
	}()

	svc := analytics.NewValidationRuleService(s.db)
	// domain="" - all domains enforce on the write path (validation,
	// plus mdm/compliance rules that are per-record write-time
	// constraints, per the rulefabric consolidation: those become
	// domain values on this same rule set rather than a second write
	// hook). Batch-shaped mdm/compliance rules (wash-trade over
	// history, concentration over positions) are not BO-scoped the same
	// way and are evaluated by the sweep harness, not here.
	rules, err := svc.ListByBO(ctx, tenantID, boKey, "")
	if err != nil {
		logging.GetLogger().Sugar().Warnf("rule evaluation: failed to list rules for BO %s: %v", boKey, err)
		return nil, false
	}
	if len(rules) == 0 {
		return nil, false
	}

	data := make(map[string]interface{}, len(record)+8)
	for k, v := range record {
		data[k] = coerceNumeric(v)
	}

	// Alias every semantic term to its currently-bound physical column's
	// value, so a rule authored against "TargetQuantity" (portable across
	// bindings) and one authored directly against "target_qty" (tied to
	// this binding) both evaluate correctly against the same write.
	if fieldMap, err := analytics.ResolveSemanticFieldMap(ctx, s.db, bo.ID, bo.DriverTableName); err != nil {
		logging.GetLogger().Sugar().Warnf("rule evaluation: failed to resolve semantic field map for BO %s: %v", boKey, err)
	} else {
		for semantic, physical := range fieldMap {
			if v, ok := data[physical]; ok {
				data[semantic] = v
			}
		}
	}

	switch boKey {
	case "order":
		s.loadOrderContext(ctx, exec, data)
	default:
		if rc, ok := shadowRuleContexts[boKey]; ok {
			s.loadRelatedRowContext(ctx, exec, rc, record, data)
		}
	}

	enforce := enforcementEnabled()
	ae := vm.NewAdvancedEvaluator()
	for _, rule := range rules {
		var node vm.RuleNode
		if err := json.Unmarshal(rule.RuleAST, &node); err != nil {
			v := ruleViolation{
				RuleID: rule.ID.String(), RuleName: rule.Name, Severity: rule.Severity,
				Message:   fmt.Sprintf("rule error: rule_ast did not parse: %v", err),
				Context:   data,
				RuleError: true,
			}
			violations = append(violations, v)
			if enforce && rule.Severity == "BLOCK" {
				blocked = true
			}
			continue
		}
		// Condition nodes (unlike Expression/FuncCall's FieldRef) treat a
		// missing field as false, nil, not an error -
		// ConditionEvaluator.evaluateSimpleCondition's documented
		// behavior, shared far too broadly to change safely from here.
		// So an unresolvable field reference is checked explicitly,
		// before evaluation, rather than relying on ae.Evaluate to
		// surface it as an error - otherwise a Condition-type rule
		// referencing a nonexistent or unbound term degrades silently
		// into "always false" instead of failing loud the way an
		// Expression-type rule already does.
		if missing := unresolvedFieldRefs(node, data); len(missing) > 0 {
			v := ruleViolation{
				RuleID: rule.ID.String(), RuleName: rule.Name, Severity: rule.Severity,
				Message:   fmt.Sprintf("rule error: field(s) %v not present in evaluation context (no MAPS_TO binding, and not a raw column on this record)", missing),
				Context:   data,
				RuleError: true,
			}
			violations = append(violations, v)
			if enforce && rule.Severity == "BLOCK" {
				blocked = true
			}
			continue
		}
		pass, err := ae.Evaluate(node, data)
		if err != nil {
			// A rule that can't evaluate - most commonly an unresolvable
			// field reference (a semantic term with no MAPS_TO binding on
			// this BO's current binding, or a genuinely typo'd field name)
			// - is never a silent skip. It's persisted as a violation like
			// any other, tagged RuleError so it isn't mistaken for "the
			// data passed", and treated at least as seriously as a real
			// BLOCK violation for enforcement purposes: not knowing
			// whether a rule is satisfied is not the same as it being
			// satisfied.
			v := ruleViolation{
				RuleID: rule.ID.String(), RuleName: rule.Name, Severity: rule.Severity,
				Message:   fmt.Sprintf("rule error: %v", err),
				Context:   data,
				RuleError: true,
			}
			violations = append(violations, v)
			if enforce && rule.Severity == "BLOCK" {
				blocked = true
			}
			continue
		}
		if pass {
			continue
		}
		v := ruleViolation{
			RuleID:   rule.ID.String(),
			RuleName: rule.Name,
			Severity: rule.Severity,
			Message:  fmt.Sprintf("%s failed for %s record %v", rule.Name, boKey, record["id"]),
			Context:  data,
		}
		violations = append(violations, v)
		if enforce && rule.Severity == "BLOCK" {
			blocked = true
		}
	}
	return violations, blocked
}

// loadRelatedRowContext fills data with the parent row's fields and the
// sibling-row aggregate described by rc, reading through exec so it sees
// the write it's evaluating even before that write's transaction commits.
func (s *BusinessObjectService) loadRelatedRowContext(ctx context.Context, exec dbExecutor, rc relatedRowContext, record map[string]interface{}, data map[string]interface{}) {
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
		rows, err := exec.QueryxContext(ctx, query, parentID)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("rule context: failed to load parent %s: %v", rc.parentTable, err)
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
		if err := exec.GetContext(ctx, &sum, query, parentID); err != nil {
			logging.GetLogger().Sugar().Warnf("rule context: failed to load sibling sum: %v", err)
			return
		}
		data[rc.siblingSumAs] = sum
	}
}

// loadOrderContext fills data for the Order BO's rule set:
//   - allocation_target_qty_sum: SUM(order_allocation.target_qty) for
//     this order - the allocation-completeness rule compares this
//     against the order's own target_qty.
//   - account_status/account_is_discretionary: read off orm.account via
//     the order's first allocation's account_id (ordered by created_at).
//     Simplification: an order with allocations split across multiple
//     accounts only gets the first account's compliance fields - correct
//     for the proof's single-account order chains, a real multi-account
//     order would need a per-allocation evaluation pass instead of one
//     order-level check.
func (s *BusinessObjectService) loadOrderContext(ctx context.Context, exec dbExecutor, data map[string]interface{}) {
	orderID, ok := data["id"]
	if !ok || orderID == nil {
		return
	}

	var sum float64
	if err := exec.GetContext(ctx, &sum,
		"SELECT COALESCE(SUM(target_qty), 0) FROM orm.order_allocation WHERE order_id = $1", orderID); err != nil {
		logging.GetLogger().Sugar().Warnf("order rule context: failed to sum allocations: %v", err)
	} else {
		data["allocation_target_qty_sum"] = sum
	}

	rows, err := exec.QueryxContext(ctx,
		`SELECT a.status, a.is_discretionary
		 FROM orm.order_allocation oa
		 JOIN orm.account a ON a.account_id = oa.account_id
		 WHERE oa.order_id = $1
		 ORDER BY oa.created_at ASC
		 LIMIT 1`, orderID)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("order rule context: failed to load account: %v", err)
		return
	}
	defer rows.Close()
	if rows.Next() {
		row := make(map[string]interface{})
		_ = rows.MapScan(row)
		if v, ok := row["status"]; ok {
			data["account_status"] = coerceNumeric(v)
		}
		if v, ok := row["is_discretionary"]; ok {
			data["account_is_discretionary"] = v
		}
	}
}

// knownTransientContextFields are related-row-context keys (loaded by
// loadOrderContext/loadRelatedRowContext) that can legitimately be absent
// for reasons that have nothing to do with the rule being broken - e.g.
// "account_status" isn't there yet because this order has no allocation
// yet, not because the term is unbound. unresolvedFieldRefs excludes
// these from the fail-loud check entirely; a rule referencing one of
// these that's currently absent evaluates via Condition's existing
// "missing field -> false" behavior, same as before this check existed.
// Deliberately not derived from shadowRuleContexts/loadOrderContext
// automatically - keeping it a short, explicit, reviewable list here
// beats a generic mechanism for the two BOs that need it today.
var knownTransientContextFields = map[string]bool{
	"sibling_qty_sum":           true,
	"routed_qty":                true,
	"executed_qty":              true,
	"allocation_target_qty_sum": true,
	"account_status":            true,
	"account_is_discretionary":  true,
}

// unresolvedFieldRefs walks node's tree and returns every top-level
// (no ".") field reference that is neither a key in data nor a known-
// transient context field - the pre-evaluation check that makes a truly
// unresolvable Condition-type reference (an unbound semantic term, a
// typo'd field name) fail loud instead of silently evaluating to false
// (see the call site's comment for why this can't be fixed inside
// ConditionEvaluator itself, which the whole engine shares).
func unresolvedFieldRefs(node vm.RuleNode, data map[string]interface{}) []string {
	refs := make(map[string]bool)
	collectRuleFieldRefs(node, refs)
	var missing []string
	for f := range refs {
		if strings.Contains(f, ".") {
			continue // nested paths are HierarchyResolver's concern, not this check's
		}
		if knownTransientContextFields[f] {
			continue
		}
		if _, ok := data[f]; !ok {
			missing = append(missing, f)
		}
	}
	return missing
}

func collectRuleFieldRefs(node vm.RuleNode, out map[string]bool) {
	switch node.Type {
	case vm.NodeTypeGroup:
		if node.Group != nil {
			for _, c := range node.Group.Conditions {
				collectRuleFieldRefs(c, out)
			}
		}
	case vm.NodeTypeCondition:
		if node.Condition != nil {
			f := node.Condition.Field
			if node.Condition.FieldPath != "" {
				f = node.Condition.FieldPath
			}
			if f != "" {
				out[f] = true
			}
		}
	case vm.NodeTypeExpression:
		if node.Expression != nil {
			collectExprFieldRefs(node.Expression.Root, out)
		}
	}
}

func collectExprFieldRefs(n vm.ExprNode, out map[string]bool) {
	switch t := n.(type) {
	case *vm.BinaryExpr:
		collectExprFieldRefs(t.Left, out)
		collectExprFieldRefs(t.Right, out)
	case *vm.FieldRef:
		out[t.Path] = true
	case *vm.FuncCall:
		for _, a := range t.Args {
			collectExprFieldRefs(a, out)
		}
	}
}

// coerceNumeric converts values that are numeric but not typed as such
// into float64, so they land in the evaluation context as something
// AdvancedEvaluator's arithmetic operators recognize. Two cases hit
// this: lib/pq returns Postgres `numeric` columns as []byte (numeric
// isn't one of its natively-typed scan targets), and CreateBORecord/
// UpdateBORecord stringify every []byte field before returning their
// result map (for JSON serialization), so by the time a written
// record's own numeric fields reach here they're strings, not floats.
// Both cases get one ParseFloat attempt; anything that doesn't parse
// (a status string, a UUID, a real non-numeric value) passes through
// completely unchanged, including its original type.
func coerceNumeric(v interface{}) interface{} {
	var s string
	switch t := v.(type) {
	case []byte:
		s = string(t)
	case string:
		s = t
	default:
		return v
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return v
}
