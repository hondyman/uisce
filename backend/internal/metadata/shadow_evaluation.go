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
type ruleViolation struct {
	RuleID   string
	RuleName string
	Severity string
	Message  string
	Context  map[string]interface{}
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
func (s *BusinessObjectService) writeAndEnforce(ctx context.Context, tenantID, boKey string, doWrite func(tx *sqlx.Tx) (map[string]interface{}, error)) (map[string]interface{}, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	result, err := doWrite(tx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	violations, blocked := s.evaluateAndEnforceRules(ctx, tx, tenantID, boKey, result)

	if blocked {
		_ = tx.Rollback()
	} else if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	recordID := fmt.Sprintf("%v", result["id"])
	for _, v := range violations {
		writeBlocked := blocked && v.Severity == "BLOCK"
		rec := analytics.ViolationRecord{
			TenantID: tenantID, RuleID: v.RuleID, RuleName: v.RuleName, BOKey: boKey,
			Severity: v.Severity, RecordID: recordID, Message: v.Message,
			Context: v.Context, WriteBlocked: writeBlocked,
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
func (s *BusinessObjectService) evaluateAndEnforceRules(ctx context.Context, exec dbExecutor, tenantID, boKey string, record map[string]interface{}) (violations []ruleViolation, blocked bool) {
	defer func() {
		if r := recover(); r != nil {
			logging.GetLogger().Sugar().Errorf("rule evaluation panicked for BO %s: %v", boKey, r)
			violations = nil
			blocked = false
		}
	}()

	svc := analytics.NewValidationRuleService(s.db)
	rules, err := svc.ListByBO(ctx, tenantID, boKey)
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
			logging.GetLogger().Sugar().Warnf("rule %s (%s): rule_ast did not parse: %v", rule.ID, rule.Name, err)
			continue
		}
		pass, err := ae.Evaluate(node, data)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("rule %s (%s) errored during evaluation for BO %s: %v", rule.ID, rule.Name, boKey, err)
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
