package services

import (
	"context"
	"database/sql"
)

type JITGrantAuditEvent struct {
	ID         string
	GrantID    string
	UserID     string
	EventType  string
	Reason     string
	OccurredAt string
}

// ListJITGrantAuditEvents returns audit events, optionally filtered by user or bundle.
func ListJITGrantAuditEvents(ctx context.Context, db *sql.DB, userID, bundleID string) ([]JITGrantAuditEvent, error) {
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
		if err := rows.Scan(&e.ID, &e.GrantID, &e.UserID, &e.EventType, &e.Reason, &e.OccurredAt); err == nil {
			events = append(events, e)
		}
	}
	return events, nil
}
