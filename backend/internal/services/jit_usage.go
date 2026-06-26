package services

import (
	"context"
	"database/sql"
	"time"
)

// MarkJITGrantUsed updates the last_used_at timestamp for a grant.
func MarkJITGrantUsed(ctx context.Context, db *sql.DB, grantID string) error {
	_, err := db.ExecContext(ctx, `UPDATE jit_addon_grant SET last_used_at = $1 WHERE id = $2`, time.Now(), grantID)
	return err
}

// RevokeUnusedJITGrants revokes grants that have not been used within a policy window.
func RevokeUnusedJITGrants(ctx context.Context, db *sql.DB, unusedWindow time.Duration) error {
	_, err := db.ExecContext(ctx, `UPDATE jit_addon_grant SET status = 'revoked' WHERE status = 'active' AND (last_used_at IS NULL OR last_used_at < $1)`, time.Now().Add(-unusedWindow))
	return err
}
