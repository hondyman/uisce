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

// SetRLSContext sets the RLS context variable for a database connection or transaction
func SetRLSContext(ctx context.Context, db interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}, tenantID string) error {
	// Note: We ignore the result of ExecContext as it's a SET command
	_, err := db.ExecContext(ctx, "SELECT set_config('app.current_tenant_id', $1, false)", tenantID)
	return err
}

// SetupAuthContext sets up authentication context with tenant ID
func SetupAuthContext(ctx context.Context, tenantID string) context.Context {
	ctx = context.WithValue(ctx, "tenant_id", tenantID)
	ctx = context.WithValue(ctx, "app.current_tenant_id", tenantID)
	return ctx
}
