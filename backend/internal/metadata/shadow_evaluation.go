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
	"time"

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

// Every OMS BO now has a dedicated loader (below) rather than the earlier
// one-size-fits-all relatedRowContext shape: Placement/Execution need a
// two-hop parent (Execution -> Placement -> Order) and duplicate-row
// counts that a single parent-table + single-sibling-sum shape can't
// express, and once Execution needed its own loader there was no BO left
// for the generic shape to serve. See
// docs/unified-rule-engine-handoff.md's OMS validation spec for the full
// rule set each of these feeds.

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
	case "placement":
		s.loadPlacementContext(ctx, exec, data)
	case "execution":
		s.loadExecutionContext(ctx, exec, data)
	case "order_allocation":
		s.loadOrderAllocationContext(ctx, exec, data)
	case "execution_allocation":
		s.loadExecutionAllocationContext(ctx, exec, data)
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

// loadPlacementContext fills data for the Placement BO's rule set:
//   - order_target_qty: the parent order's target_qty (over-placement
//     check compares the sibling routed-qty sum against this).
//   - sibling_routed_sum: SUM(routed_qty) across every placement under
//     the same order, including the row this write just produced (same
//     "sum includes the just-written row, because evaluation runs inside
//     its own transaction" convention as Execution's sibling_qty_sum).
//   - broker_status: orm.broker.status for this placement's broker_id -
//     absent (not an error) if the broker isn't in the reference table,
//     same as Order's account lookup.
func (s *BusinessObjectService) loadPlacementContext(ctx context.Context, exec dbExecutor, data map[string]interface{}) {
	orderID, ok := data["order_id"]
	if !ok || orderID == nil {
		return
	}

	var targetQty float64
	if err := exec.GetContext(ctx, &targetQty, `SELECT target_qty FROM orm."order" WHERE id = $1`, orderID); err != nil {
		logging.GetLogger().Sugar().Warnf("placement rule context: failed to load parent order target_qty: %v", err)
	} else {
		data["order_target_qty"] = targetQty
	}

	var sum float64
	if err := exec.GetContext(ctx, &sum, `SELECT COALESCE(SUM(routed_qty), 0) FROM orm.placement WHERE order_id = $1`, orderID); err != nil {
		logging.GetLogger().Sugar().Warnf("placement rule context: failed to sum sibling routed_qty: %v", err)
	} else {
		data["sibling_routed_sum"] = sum
	}

	if brokerID, ok := data["broker_id"]; ok && brokerID != nil {
		var status string
		if err := exec.GetContext(ctx, &status, `SELECT status FROM orm.broker WHERE broker_id = $1`, brokerID); err == nil {
			data["broker_status"] = status
		}
	}
}

// loadExecutionContext fills data for the Execution BO's rule set. Parent
// fields come from a two-hop join (Execution -> Placement -> Order), since
// price-vs-limit and causality checks need the order's side/limit_price
// and the placement's created_at, not just the placement's own routed/
// executed quantities the earlier generic shape loaded.
//   - routed_qty/executed_qty: the parent placement's own fields (same
//     names/behavior as before this was split out of the generic shape).
//   - placement_created_at: the parent placement's created_at, for the
//     causality check (exec_time >= placement created_at).
//   - order_side/order_limit_price: the grandparent order's side and
//     limit_price, for the price-vs-limit check.
//   - sibling_qty_sum: SUM(exec_qty) across every execution under the
//     same placement, including this write - the overfill guard's
//     existing, already-proven context key (cmd/verify_shadow_context).
//   - duplicate_broker_exec_count: how many OTHER executions share this
//     row's broker_id + broker_exec_id (excluding this row by id) - the
//     duplicate-broker-exec WARN's context.
func (s *BusinessObjectService) loadExecutionContext(ctx context.Context, exec dbExecutor, data map[string]interface{}) {
	placementID, ok := data["placement_id"]
	if !ok || placementID == nil {
		return
	}

	row := make(map[string]interface{})
	rows, err := exec.QueryxContext(ctx, `
		SELECT p.routed_qty, p.executed_qty, p.created_at AS placement_created_at,
		       o.side AS order_side, o.limit_price AS order_limit_price
		FROM orm.placement p
		JOIN orm."order" o ON o.id = p.order_id
		WHERE p.id = $1`, placementID)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("execution rule context: failed to load parent placement/order: %v", err)
		return
	}
	if rows.Next() {
		_ = rows.MapScan(row)
		for k, v := range row {
			data[k] = normalizeScanned(v)
		}
	}
	rows.Close()

	var sum float64
	if err := exec.GetContext(ctx, &sum, `SELECT COALESCE(SUM(exec_qty), 0) FROM orm.execution WHERE placement_id = $1`, placementID); err != nil {
		logging.GetLogger().Sugar().Warnf("execution rule context: failed to sum sibling exec_qty: %v", err)
	} else {
		data["sibling_qty_sum"] = sum
	}

	// causality_ok: a precomputed boolean, not a raw timestamp comparison
	// left to the rule itself - ConditionEvaluator's greater_equal/
	// less_equal and AdvancedEvaluator's BinaryExpr both delegate to
	// numeric coercion (hierarchy_resolver.go's toNumber), which errors on
	// an RFC3339 timestamp string. Rather than teach either shared
	// comparator about timestamps (out of scope, and exactly the kind of
	// change the standing "never mutate the shared ConditionEvaluator"
	// rule exists to prevent), the comparison is done once here in Go and
	// exposed as a plain boolean the rule can check with "equals" - the
	// same "precompute it in the context provider" approach the sum/count
	// keys already use for anything the engine's own operators can't
	// express directly.
	if execTime, ok := data["exec_time"]; ok {
		if pt, pok := parseTimestamp(data["placement_created_at"]); pok {
			if et, eok := parseTimestamp(execTime); eok {
				data["causality_ok"] = !et.Before(pt)
			}
		}
	}

	brokerID, hasBroker := data["broker_id"]
	brokerExecID, hasBrokerExecID := data["broker_exec_id"]
	if hasBroker && hasBrokerExecID && brokerID != nil && brokerExecID != nil {
		var count int
		id := data["id"]
		if err := exec.GetContext(ctx, &count, `
			SELECT COUNT(*) FROM orm.execution
			WHERE broker_id = $1 AND broker_exec_id = $2 AND id != $3`,
			brokerID, brokerExecID, id); err != nil {
			logging.GetLogger().Sugar().Warnf("execution rule context: failed to count duplicate broker execs: %v", err)
		} else {
			data["duplicate_broker_exec_count"] = float64(count)
		}
	}
}

// loadOrderAllocationContext fills data for the OrderAllocation BO's rule
// set:
//   - account_status/account_is_discretionary: this allocation's own
//     account_id looked up directly in orm.account - simpler than
//     Order's version (item 8/13 of the handoff), which has to find an
//     account indirectly via the order's first allocation, because an
//     OrderAllocation row already carries its own account_id.
//   - alloc_fill_sum: SUM(alloc_exec_qty) across every execution_allocation
//     row distributed against this order_allocation, for the
//     allocation-reconcile rule (AllocatedQuantity == alloc_fill_sum).
func (s *BusinessObjectService) loadOrderAllocationContext(ctx context.Context, exec dbExecutor, data map[string]interface{}) {
	if accountID, ok := data["account_id"]; ok && accountID != nil {
		row := make(map[string]interface{})
		rows, err := exec.QueryxContext(ctx, `SELECT status, is_discretionary FROM orm.account WHERE account_id = $1`, accountID)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("order_allocation rule context: failed to load account: %v", err)
		} else {
			if rows.Next() {
				_ = rows.MapScan(row)
				if v, ok := row["status"]; ok {
					data["account_status"] = normalizeScanned(v)
				}
				if v, ok := row["is_discretionary"]; ok {
					data["account_is_discretionary"] = v
				}
			}
			rows.Close()
		}
	}

	if allocID, ok := data["id"]; ok && allocID != nil {
		var sum float64
		if err := exec.GetContext(ctx, &sum,
			`SELECT COALESCE(SUM(alloc_exec_qty), 0) FROM orm.execution_allocation WHERE order_allocation_id = $1`, allocID); err != nil {
			logging.GetLogger().Sugar().Warnf("order_allocation rule context: failed to sum alloc_fill: %v", err)
		} else {
			data["alloc_fill_sum"] = sum
		}
	}
}

// loadExecutionAllocationContext fills data for the ExecutionAllocation
// BO's rule set - the one BO whose rules cross two different parents
// (its execution and its order_allocation) that must agree with each
// other:
//   - parent_exec_qty/parent_exec_price: the parent execution's own qty/
//     price, for the distribution-completeness and price-consistency
//     rules.
//   - parent_order_id: the parent execution's order_id (via
//     execution_id), for the same-order-linkage rule.
//   - alloc_order_id: the linked order_allocation's order_id (via
//     order_allocation_id) - same-order-linkage compares this against
//     parent_order_id.
//   - sibling_alloc_sum: SUM(alloc_exec_qty) across every
//     execution_allocation row distributed against the same execution,
//     including this write.
func (s *BusinessObjectService) loadExecutionAllocationContext(ctx context.Context, exec dbExecutor, data map[string]interface{}) {
	if executionID, ok := data["execution_id"]; ok && executionID != nil {
		row := make(map[string]interface{})
		rows, err := exec.QueryxContext(ctx, `SELECT exec_qty AS parent_exec_qty, exec_price AS parent_exec_price, order_id AS parent_order_id FROM orm.execution WHERE id = $1`, executionID)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("execution_allocation rule context: failed to load parent execution: %v", err)
		} else {
			if rows.Next() {
				_ = rows.MapScan(row)
				for k, v := range row {
					data[k] = normalizeScanned(v)
				}
			}
			rows.Close()
		}

		var sum float64
		if err := exec.GetContext(ctx, &sum, `SELECT COALESCE(SUM(alloc_exec_qty), 0) FROM orm.execution_allocation WHERE execution_id = $1`, executionID); err != nil {
			logging.GetLogger().Sugar().Warnf("execution_allocation rule context: failed to sum sibling allocations: %v", err)
		} else {
			data["sibling_alloc_sum"] = sum
		}
	}

	if orderAllocID, ok := data["order_allocation_id"]; ok && orderAllocID != nil {
		var orderID string
		if err := exec.GetContext(ctx, &orderID, `SELECT order_id FROM orm.order_allocation WHERE id = $1`, orderAllocID); err != nil {
			logging.GetLogger().Sugar().Warnf("execution_allocation rule context: failed to load order_allocation's order_id: %v", err)
		} else {
			data["alloc_order_id"] = orderID
		}
	}

	// same_order_ok: a precomputed boolean, not a direct
	// parent_order_id == alloc_order_id expression - AdvancedEvaluator's
	// BinaryExpr calls toFloat64 unconditionally, even for "==" (see
	// advanced_evaluator.go's evalBinaryExpr), so an Expression-type
	// equality on two UUID strings errors ("operands not numeric") no
	// matter what the values are. Same fix shape as causality_ok above:
	// do the (string) comparison once here in Go, expose it as a plain
	// boolean a Condition node can check - not a change to the shared
	// evaluator.
	if parentOrderID, ok := data["parent_order_id"]; ok {
		if allocOrderID, ok2 := data["alloc_order_id"]; ok2 {
			data["same_order_ok"] = fmt.Sprintf("%v", parentOrderID) == fmt.Sprintf("%v", allocOrderID)
		}
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

	// OrderAllocations: deliver the allocation rows as a collection so
	// SUM(OrderAllocations.target_qty) resolves via ResolveFieldPathArray.
	// Each row's values pass through normalizeRow so numeric columns (NUMERIC
	// via lib/pq → []byte → coerceNumeric → float64) land as float64 in
	// the evaluation context, matching what requireFloatSlice expects.
	// Always delivered: an empty array when there are no allocations.
	var normalized []map[string]interface{}
	allocRows, err := exec.QueryxContext(ctx,
		`SELECT target_qty, allocated_qty, status, created_at
		 FROM orm.order_allocation
		 WHERE order_id = $1
		 ORDER BY created_at ASC`, orderID)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("order rule context: failed to load OrderAllocations: %v", err)
	} else {
		for allocRows.Next() {
			row := make(map[string]interface{})
			_ = allocRows.MapScan(row)
			normalized = append(normalized, normalizeRow(row))
		}
		allocRows.Close()
	}
	data["OrderAllocations"] = normalized

	// placement_routed_sum: SUM(routed_qty) across every placement under
	// this order, including this write - the over-placement guard's
	// context (mirrors Execution's sibling_qty_sum convention).
	var routedSum float64
	if err := exec.GetContext(ctx, &routedSum,
		"SELECT COALESCE(SUM(routed_qty), 0) FROM orm.placement WHERE order_id = $1", orderID); err != nil {
		logging.GetLogger().Sugar().Warnf("order rule context: failed to sum placement routed_qty: %v", err)
	} else {
		data["placement_routed_sum"] = routedSum
	}

	// duplicate_order_count: how many OTHER orders share this order's
	// sec_id/side/target_qty/trade_date/manager_id (NULL manager_id
	// treated as equal to NULL via IS NOT DISTINCT FROM, so two orders
	// with no manager_id set can still be flagged as duplicates of each
	// other) - the duplicate-order soft-check's context. Reads straight
	// off data (already coerced/loaded from the record being written),
	// not a second DB round trip for the order's own fields.
	if secID, ok := data["sec_id"]; ok && secID != nil {
		var dupCount int
		if err := exec.GetContext(ctx, &dupCount, `
			SELECT COUNT(*) FROM orm."order"
			WHERE sec_id = $1 AND side = $2 AND target_qty = $3 AND trade_date = $4
			  AND manager_id IS NOT DISTINCT FROM $5 AND id != $6`,
			secID, data["side"], data["target_qty"], data["trade_date"], data["manager_id"], orderID); err != nil {
			logging.GetLogger().Sugar().Warnf("order rule context: failed to count duplicate orders: %v", err)
		} else {
			data["duplicate_order_count"] = float64(dupCount)
		}
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
			data["account_status"] = normalizeScanned(v)
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
	"sibling_qty_sum":             true,
	"routed_qty":                  true,
	"executed_qty":                true,
	"allocation_target_qty_sum":   true,
	"account_status":              true,
	"account_is_discretionary":    true,
	"placement_routed_sum":        true,
	"duplicate_order_count":       true,
	"order_target_qty":            true,
	"sibling_routed_sum":          true,
	"broker_status":               true,
	"placement_created_at":        true,
	"order_side":                  true,
	"order_limit_price":           true,
	"duplicate_broker_exec_count": true,
	"alloc_fill_sum":              true,
	"parent_exec_qty":             true,
	"parent_exec_price":           true,
	"parent_order_id":             true,
	"alloc_order_id":              true,
	"sibling_alloc_sum":           true,
	"causality_ok":                true,
	"same_order_ok":               true,
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

// normalizeScanned is coerceNumeric's counterpart for values that come
// from rows.MapScan (used throughout this file's context loaders) rather
// than a typed GetContext(&typedVar, ...) scan. MapScan's driver.Value
// conversion leaves text-like Postgres types (uuid, varchar) as raw
// []byte for some column type OIDs but not others - found empirically
// while debugging the same-order-linkage rule: orm.execution.order_id
// (uuid) came back as []byte, while a typed `var status string` scan of
// the very same kind of column elsewhere in this file (broker_status,
// via GetContext) came back as a clean Go string. coerceNumeric alone
// doesn't fix this - it only converts a []byte to float64 when it
// actually parses as a number, and returns non-numeric []byte
// unchanged - so a MapScan'd UUID or status string was landing in `data`
// still as []byte, which breaks both direct string comparisons
// (Condition's compareValues does reflect.DeepEqual, and
// DeepEqual([]byte("ACTIVE"), "ACTIVE") is false - different types,
// never mind equal content) and this file's own fmt.Sprintf("%v", ...)
// string-building (which renders []byte as its numeric byte values, not
// its text). Convert []byte to string FIRST, then let coerceNumeric
// still do its normal job of promoting a numeric string to float64 -
// this fixes the string case without changing coerceNumeric's existing,
// already-relied-upon behavior for record's own already-stringified
// fields (see coerceNumeric's own doc comment) or for GetContext's
// typed scans (which were never affected by this bug to begin with).
func normalizeScanned(v interface{}) interface{} {
	if b, ok := v.([]byte); ok {
		return coerceNumeric(string(b))
	}
	return coerceNumeric(v)
}

func normalizeRow(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = normalizeScanned(v)
	}
	return out
}

// parseTimestamp handles the two shapes a timestamptz value reaches this
// code as: a Go time.Time (the common case - lib/pq recognizes the
// column type from both a QueryxContext scan and an INSERT ... RETURNING
// scan) or a string (a defensive fallback, in case a caller ever supplies
// one as a record field directly rather than a time.Time).
func parseTimestamp(v interface{}) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		return t, true
	case string:
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.999999-07", "2006-01-02 15:04:05-07", "2006-01-02"} {
			if parsed, err := time.Parse(layout, t); err == nil {
				return parsed, true
			}
		}
	}
	return time.Time{}, false
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
