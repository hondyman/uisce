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
// It previously used `set_config(..., false)` (session-scoped) under the GUC
// name `app.current_tenant_id` — no RLS policy in this codebase reads that
// name (the real policies, in backend/migrations/20260727000020_enable_tenant_rls.sql
// and .../20260727000030_strict_tenant_rls.sql, all check
// `uisce.current_tenant`), so every caller believed it was scoping RLS and
// wasn't. `set_config(..., false)` is also unsafe with a connection pool: it
// persists on the pooled connection past the end of the transaction/request,
// potentially leaking one tenant's session variable into whatever request
// reuses that connection next. `SET LOCAL` is transaction-scoped and reverts
// automatically at COMMIT/ROLLBACK, which is what a pooled connection needs.
func SetRLSContext(ctx context.Context, tx interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}, tenantID string) error {
	_, err := tx.ExecContext(ctx, "SET LOCAL uisce.current_tenant = $1", tenantID)
	return err
}

// SetupAuthContext sets up authentication context with tenant ID
func SetupAuthContext(ctx context.Context, tenantID string) context.Context {
	ctx = context.WithValue(ctx, "tenant_id", tenantID)
	ctx = context.WithValue(ctx, "app.current_tenant_id", tenantID)
	return ctx
}
