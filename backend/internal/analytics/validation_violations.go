package analytics

// Persistence and retrieval for validation_rule_violations - the queryable
// record of a rule evaluation that failed, so "did this rule ever fire"
// survives past the server log line it's also written to. See
// docs/unified-rule-engine-handoff.md, "violations visible".

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// ViolationRecord is one failed rule evaluation, ready to persist.
// RuleError distinguishes "the rule ran and found a real violation" from
// "the rule couldn't run at all" (unresolvable field reference,
// malformed rule_ast) - see shadow_evaluation.go's ruleViolation for why
// this can never be a silent skip.
type ViolationRecord struct {
	TenantID     string
	RuleID       string
	RuleName     string
	BOKey        string
	Severity     string
	RecordID     string
	Message      string
	Context      map[string]interface{}
	WriteBlocked bool
	RuleError    bool
}

// PersistViolation writes v to validation_rule_violations. Deliberately
// takes db (not a transaction) - callers persist violations after
// deciding whether the write itself commits or rolls back, so a
// violation record survives either way, on its own connection.
func PersistViolation(ctx context.Context, db *sqlx.DB, v ViolationRecord) error {
	ctxJSON, err := json.Marshal(v.Context)
	if err != nil {
		return fmt.Errorf("marshal violation context: %w", err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO validation_rule_violations
			(id, tenant_id, rule_id, rule_name, bo_key, severity, record_id, message, context, write_blocked, rule_error, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now())
	`, v.TenantID, v.RuleID, v.RuleName, v.BOKey, v.Severity, v.RecordID, v.Message, ctxJSON, v.WriteBlocked, v.RuleError)
	return err
}

// ViolationSummary is what the violations API/UI reads back - the
// context payload stays raw JSON so callers can decide how much of it to
// render rather than this layer guessing.
type ViolationSummary struct {
	ID           uuid.UUID       `db:"id" json:"id"`
	TenantID     string          `db:"tenant_id" json:"tenant_id"`
	RuleID       uuid.UUID       `db:"rule_id" json:"rule_id"`
	RuleName     string          `db:"rule_name" json:"rule_name"`
	BOKey        string          `db:"bo_key" json:"bo_key"`
	Severity     string          `db:"severity" json:"severity"`
	RecordID     string          `db:"record_id" json:"record_id"`
	Message      string          `db:"message" json:"message"`
	Context      json.RawMessage `db:"context" json:"context"`
	WriteBlocked bool            `db:"write_blocked" json:"write_blocked"`
	RuleError    bool            `db:"rule_error" json:"rule_error"`
	CreatedAt    string          `db:"created_at" json:"created_at"`
}

// ListViolations returns the most recent violations, optionally filtered
// to one BO, newest first.
func ListViolations(ctx context.Context, db *sqlx.DB, tenantID, boKey string, limit int) ([]ViolationSummary, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var rows []ViolationSummary
	if boKey != "" {
		err := db.SelectContext(ctx, &rows, `
			SELECT id, tenant_id, rule_id, rule_name, bo_key, severity, record_id, message, context, write_blocked, rule_error, created_at::text
			FROM validation_rule_violations
			WHERE tenant_id = $1 AND bo_key = $2
			ORDER BY created_at DESC LIMIT $3
		`, tenantID, boKey, limit)
		return rows, err
	}
	err := db.SelectContext(ctx, &rows, `
		SELECT id, tenant_id, rule_id, rule_name, bo_key, severity, record_id, message, context, write_blocked, rule_error, created_at::text
		FROM validation_rule_violations
		WHERE tenant_id = $1
		ORDER BY created_at DESC LIMIT $2
	`, tenantID, limit)
	return rows, err
}

// RuleHealthSummary is the per-rule aggregate the system validations page
// (and the BO tab) reads to answer "did this rule ever fire" and "is this
// rule possibly broken, not just strict" without a person having to grep
// server logs - see docs/unified-rule-engine-handoff.md item 55: the
// cheap half of systematizing stale-rule detection (the real half - a
// schema-version binding on rules so one can't silently outlive the
// schema it targets - is a separate, unbuilt design question).
type RuleHealthSummary struct {
	RuleID         uuid.UUID `db:"rule_id" json:"ruleId"`
	RuleName       string    `db:"rule_name" json:"ruleName"`
	BOKey          string    `db:"bo_key" json:"boKey"`
	Severity       string    `db:"severity" json:"severity"`
	ViolationCount int       `db:"violation_count" json:"violationCount"`
	RuleErrorCount int       `db:"rule_error_count" json:"ruleErrorCount"`
	BlockedCount   int       `db:"blocked_count" json:"blockedCount"`
	LoggedCount    int       `json:"loggedCount"`
	LastFiredAt    *string   `db:"last_fired_at" json:"lastFiredAt"`
	// Suspect/SuspectReason are the health signal itself - a heuristic,
	// not a certainty. See GetRuleHealthSummary's doc comment for exactly
	// what "suspect" does and doesn't mean.
	Suspect       bool   `json:"suspect"`
	SuspectReason string `json:"suspectReason,omitempty"`
}

// GetRuleHealthSummary aggregates validation_rule_violations per rule and
// flags two suspect patterns, both cheap to compute from data this table
// already has:
//   - Any rule_error at all. Unambiguous by construction - a rule that
//     cannot even evaluate (an unresolvable field, a malformed AST) is
//     broken regardless of any rate, no denominator needed.
//   - A violation rate that looks like "this rule fails almost every
//     write" - approximated as violation_count against the BO's current
//     physical row count (via business_objects.driver_table_name), NOT
//     an exact "evaluations that failed / evaluations that ran" ratio,
//     since this table only records failures, never passes, so the true
//     denominator (total evaluations) isn't directly available. This is
//     a real approximation, stated as one: a BO with heavy deletes or a
//     rule newly added after most existing rows were written would both
//     under- or over-estimate the true rate. Good enough to surface "look
//     at this one first," not a certified defect count. Suppressed below
//     a small row-count floor so a nearly-empty BO doesn't manufacture a
//     100% rate from one or two rows.
//
// This is the cheap half of item 55's stale-rule-detection systematization;
// the real half (binding a rule to the schema version it was authored
// against) is a separate, unbuilt design question.
func GetRuleHealthSummary(ctx context.Context, db *sqlx.DB, tenantID string) ([]RuleHealthSummary, error) {
	var rows []RuleHealthSummary
	err := db.SelectContext(ctx, &rows, `
		SELECT
			rule_id, rule_name, bo_key,
			MAX(severity) AS severity,
			COUNT(*) AS violation_count,
			COUNT(*) FILTER (WHERE rule_error) AS rule_error_count,
			COUNT(*) FILTER (WHERE write_blocked) AS blocked_count,
			MAX(created_at)::text AS last_fired_at
		FROM validation_rule_violations
		WHERE tenant_id = $1
		GROUP BY rule_id, rule_name, bo_key
		ORDER BY MAX(created_at) DESC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("aggregate rule health: %w", err)
	}

	// Resolve each distinct BO's current physical row count once, not
	// once per rule - most tenants have far fewer BOs than rules.
	boCounts := make(map[string]int)
	for i := range rows {
		rows[i].LoggedCount = rows[i].ViolationCount - rows[i].BlockedCount
		if rows[i].RuleErrorCount > 0 {
			rows[i].Suspect = true
			rows[i].SuspectReason = fmt.Sprintf("%d of %d recorded evaluations could not run at all (unresolvable field or malformed rule)", rows[i].RuleErrorCount, rows[i].ViolationCount)
			continue
		}
		count, ok := boCounts[rows[i].BOKey]
		if !ok {
			count, err = boRowCount(ctx, db, tenantID, rows[i].BOKey)
			if err != nil {
				// Not fatal to the whole summary - just skip the
				// rate-based check for this BO's rules.
				count = -1
			}
			boCounts[rows[i].BOKey] = count
		}
		if count >= 5 && rows[i].ViolationCount >= count {
			rows[i].Suspect = true
			rows[i].SuspectReason = fmt.Sprintf("%d violations against ~%d rows currently in %s - approximately every row fails this rule", rows[i].ViolationCount, count, rows[i].BOKey)
		}
	}
	return rows, nil
}

// boRowCount returns the BO's current physical row count via
// business_objects.driver_table_name, mirroring
// internal/metadata/businessobject_service.go's resolveQualifiedTable
// (duplicated narrowly here rather than exported across packages for one
// call site - same qualified_path convention, "/schema/table").
func boRowCount(ctx context.Context, db *sqlx.DB, tenantID, boKey string) (int, error) {
	var driverTable string
	if err := db.GetContext(ctx, &driverTable, `
		SELECT driver_table_name FROM business_objects
		WHERE (bo_key = $1 OR bo_name = $1) AND tenant_id = $2::uuid LIMIT 1
	`, boKey, tenantID); err != nil {
		return 0, err
	}
	schema, table := "public", driverTable
	if strings.HasPrefix(driverTable, "/") {
		parts := strings.Split(strings.Trim(driverTable, "/"), "/")
		if len(parts) >= 2 {
			schema, table = parts[0], parts[1]
		} else if len(parts) == 1 && parts[0] != "" {
			table = parts[0]
		}
	} else if idx := strings.LastIndex(driverTable, "."); idx >= 0 {
		schema, table = driverTable[:idx], driverTable[idx+1:]
	}
	var count int
	quoted := pq.QuoteIdentifier(schema) + "." + pq.QuoteIdentifier(table)
	if err := db.GetContext(ctx, &count, fmt.Sprintf("SELECT COUNT(*) FROM %s", quoted)); err != nil {
		return 0, err
	}
	return count, nil
}
