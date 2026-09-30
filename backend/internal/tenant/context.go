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
// Uses `set_config(..., true)` to ensure settings are transaction-scoped (SET LOCAL).
//
// CANONICAL GUC: `app.current_tenant` is the standard GUC name for all new RLS policies.
// LEGACY COMPATIBILITY: `app.current_tenant_id` and `uisce.current_tenant` are set
// concurrently as compatibility shims for pre-existing migrations and schedule runners.
// Backlog item (governance session): Migrate all legacy policies to `app.current_tenant`
// and collapse to the single canonical setting.
func SetRLSContext(ctx context.Context, tx interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}, tenantID string) error {
	_, err := tx.ExecContext(ctx, `
		SELECT set_config('app.current_tenant', $1, true),
		       set_config('app.current_tenant_id', $1, true),
		       set_config('uisce.current_tenant', $1, true)
	`, tenantID)
	return err
}

// SetupAuthContext sets up authentication context with tenant ID
func SetupAuthContext(ctx context.Context, tenantID string) context.Context {
	ctx = context.WithValue(ctx, "tenant_id", tenantID)
	ctx = context.WithValue(ctx, "app.current_tenant_id", tenantID)
	return ctx
}
