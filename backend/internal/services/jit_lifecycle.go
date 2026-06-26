package services

import (
	"context"
	"database/sql"
	"time"
)

// ExpireJITAddonGrants finds and expires all JIT grants that have passed their expiry time.
func ExpireJITAddonGrants(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `UPDATE jit_addon_grant SET status = 'expired' WHERE expires_at <= $1 AND status = 'active'`, time.Now())
	return err
}

// RevokeJITAddonGrant sets a JIT grant to revoked status (manual or policy-driven).
func RevokeJITAddonGrant(ctx context.Context, db *sql.DB, grantID string) error {
	_, err := db.ExecContext(ctx, `UPDATE jit_addon_grant SET status = 'revoked' WHERE id = $1`, grantID)
	return err
}
