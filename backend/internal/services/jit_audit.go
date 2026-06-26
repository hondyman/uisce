package services

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// AuditJITGrantEvent logs a JIT grant lifecycle event (grant, expire, revoke, renew).
func AuditJITGrantEvent(ctx context.Context, db *sql.DB, grantID uuid.UUID, userID, eventType, reason string) error {
	_, err := db.ExecContext(ctx, `INSERT INTO jit_addon_grant_audit (id, grant_id, user_id, event_type, reason, occurred_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New(), grantID, userID, eventType, reason, time.Now())
	return err
}
