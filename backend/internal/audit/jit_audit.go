// Package audit - Just-In-Time (JIT) access audit logging.
//
// This file contains audit types and helpers for JIT addon grant lifecycle
// events. It was extracted from internal/services/jit_audit*.go.
//
// Cardinal Rule 7 (tenant security): every event includes grantID and userID.
package audit

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// JIT GRANT AUDIT EVENT TYPES
// ============================================================================

// JITGrantAuditEvent represents a JIT grant lifecycle event.
type JITGrantAuditEvent struct {
	ID         string `db:"id"`
	GrantID    string `db:"grant_id"`
	UserID     string `db:"user_id"`
	EventType  string `db:"event_type"`
	Reason     string `db:"reason"`
	OccurredAt string `db:"occurred_at"`
}

// ============================================================================
// JIT GRANT AUDIT OPERATIONS
// ============================================================================

// AuditJITGrantEvent logs a JIT grant lifecycle event (grant, expire, revoke, renew).
func AuditJITGrantEvent(ctx context.Context, db *sql.DB, grantID uuid.UUID, userID, eventType, reason string) error {
	if db == nil {
		return nil
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO jit_addon_grant_audit (id, grant_id, user_id, event_type, reason, occurred_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New(), grantID, userID, eventType, reason, time.Now())
	return err
}

// ListJITGrantAuditEvents returns audit events, optionally filtered by user or bundle.
//
// Cardinal Rule 1.3 (UUID safety): bundle filter is a parameterized query.
func ListJITGrantAuditEvents(ctx context.Context, db *sql.DB, userID, bundleID string) ([]JITGrantAuditEvent, error) {
	if db == nil {
		return nil, nil
	}
	query := `SELECT id, grant_id, user_id, event_type, reason, occurred_at FROM jit_addon_grant_audit WHERE 1=1`
	var args []interface{}
	if userID != "" {
		query += " AND user_id = ?"
		args = append(args, userID)
	}
	if bundleID != "" {
		query += " AND grant_id IN (SELECT id FROM jit_addon_grant WHERE bundle_id = ?)"
		args = append(args, bundleID)
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []JITGrantAuditEvent
	for rows.Next() {
		var e JITGrantAuditEvent
		if err := rows.Scan(&e.ID, &e.GrantID, &e.UserID, &e.EventType, &e.Reason, &e.OccurredAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}