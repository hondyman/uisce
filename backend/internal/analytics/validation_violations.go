package analytics

// Persistence and retrieval for validation_rule_violations - the queryable
// record of a rule evaluation that failed, so "did this rule ever fire"
// survives past the server log line it's also written to. See
// docs/unified-rule-engine-handoff.md, "violations visible".

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ViolationRecord is one failed rule evaluation, ready to persist.
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
			(id, tenant_id, rule_id, rule_name, bo_key, severity, record_id, message, context, write_blocked, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, now())
	`, v.TenantID, v.RuleID, v.RuleName, v.BOKey, v.Severity, v.RecordID, v.Message, ctxJSON, v.WriteBlocked)
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
			SELECT id, tenant_id, rule_id, rule_name, bo_key, severity, record_id, message, context, write_blocked, created_at::text
			FROM validation_rule_violations
			WHERE tenant_id = $1 AND bo_key = $2
			ORDER BY created_at DESC LIMIT $3
		`, tenantID, boKey, limit)
		return rows, err
	}
	err := db.SelectContext(ctx, &rows, `
		SELECT id, tenant_id, rule_id, rule_name, bo_key, severity, record_id, message, context, write_blocked, created_at::text
		FROM validation_rule_violations
		WHERE tenant_id = $1
		ORDER BY created_at DESC LIMIT $2
	`, tenantID, limit)
	return rows, err
}
