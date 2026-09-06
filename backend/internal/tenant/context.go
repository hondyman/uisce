package tenant

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// ExtractTenantFromContext extracts tenant ID from context
func ExtractTenantFromContext(ctx context.Context) (uuid.UUID, error) {
	tenantStr, ok := ctx.Value("tenant_id").(string)
	if !ok {
		tenantStr, ok = ctx.Value("app.current_tenant_id").(string)
		if !ok {
			return uuid.Nil, fmt.Errorf("missing tenant context")
		}
	}

	return uuid.Parse(tenantStr)
}

// SetRLSContext sets the RLS context GUC for the given transaction.
//
// SECURITY: this must be called with an open *sql.Tx, never a bare *sql.DB.
// Uses `set_config('uisce.current_tenant', $1, true)` — the `true` argument
// makes it transaction-scoped (equivalent to SET LOCAL), so it reverts
// automatically at COMMIT/ROLLBACK and is safe for connection pools.
// Note: PostgreSQL's SET/SET LOCAL statements do not support bind parameters
// ($1, $2) — the previous `SET LOCAL uisce.current_tenant = $1` syntax was
// silently failing with a syntax error on every call; all callers were
// running without any RLS context.
func SetRLSContext(ctx context.Context, tx interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}, tenantID string) error {
	_, err := tx.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", tenantID)
	return err
}

// SetupAuthContext sets up authentication context with tenant ID
func SetupAuthContext(ctx context.Context, tenantID string) context.Context {
	ctx = context.WithValue(ctx, "tenant_id", tenantID)
	ctx = context.WithValue(ctx, "app.current_tenant_id", tenantID)
	return ctx
}
