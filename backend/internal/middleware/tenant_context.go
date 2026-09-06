package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/identity"
)

const tenantContextKey = "tenant_context"

func WithTenantContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantIDStr := r.Header.Get("X-Tenant-Id")

		if actorTenantID, ok := identity.TenantIDFromContext(r.Context()); ok && actorTenantID != "" {
			tenantIDStr = actorTenantID
		}

		if tenantIDStr != "" {
			if _, err := uuid.Parse(tenantIDStr); err == nil {
				ctx := db.WithTenantContextToCtx(r.Context(), tenantIDStr)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func GetTenantContextFromRequest(ctx context.Context) *db.TenantCtx {
	v := ctx.Value(tenantContextKey)
	if v == nil {
		return nil
	}
	return v.(*db.TenantCtx)
}

func GetTenantIDFromRequest(ctx context.Context) (string, error) {
	return db.GetTenantIDFromCtx(ctx)
}

func SetSessionTenantContext(ctx context.Context, tx *sql.Tx, tenantID string) error {
	if _, err := uuid.Parse(tenantID); err != nil {
		return fmt.Errorf("invalid tenant ID format for RLS session setting: %w", err)
	}
	_, err := tx.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", tenantID)
	if err != nil {
		return fmt.Errorf("failed to set session RLS parameter uisce.current_tenant: %w", err)
	}
	return nil
}

func RequireVerifiedTenant(ctx context.Context) error {
	return db.RequireVerifiedTenantFromCtx(ctx)
}

func ValidateTenantAccess(ctx context.Context, targetTenantID string) error {
	tc := GetTenantContextFromRequest(ctx)
	if tc == nil {
		return nil
	}
	if tc.TenantID != targetTenantID {
		return fmt.Errorf("security boundary violation: tenant '%s' cannot access resource belonging to '%s'", tc.TenantID, targetTenantID)
	}
	return nil
}
