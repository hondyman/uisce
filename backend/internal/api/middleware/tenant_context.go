package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/scheduler_intelligence"
)

type contextKey string

const (
	tenantContextKey contextKey = "tenant_context"
)

// WithTenantContext extracts actor type and tenant ID from request headers
func WithTenantContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actorType := r.Header.Get("X-Actor-Type")
		tenantIDStr := r.Header.Get("X-Tenant-Id")

		tc := &scheduler_intelligence.TenantContext{
			Actor: scheduler_intelligence.ActorTenantOps, // default
		}

		if actorType == string(scheduler_intelligence.ActorGlobalOps) {
			tc.Actor = scheduler_intelligence.ActorGlobalOps
		}

		if tenantIDStr != "" {
			if id, err := uuid.Parse(tenantIDStr); err == nil {
				tc.TenantID = &id
			}
		}

		ctx := context.WithValue(r.Context(), tenantContextKey, tc)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetTenantContext retrieves the tenant context from the request context
func GetTenantContext(ctx context.Context) *scheduler_intelligence.TenantContext {
	v := ctx.Value(tenantContextKey)
	if v == nil {
		return nil
	}
	return v.(*scheduler_intelligence.TenantContext)
}

// ValidateTenantAccess checks if the active request context is authorized to access targetTenantID
func ValidateTenantAccess(ctx context.Context, targetTenantID string) error {
	tc := GetTenantContext(ctx)
	if tc == nil {
		return nil // unconstrained if no middleware context set
	}
	if tc.Actor == scheduler_intelligence.ActorGlobalOps {
		return nil // global ops authorized for all tenants
	}
	if tc.TenantID == nil {
		return fmt.Errorf("security boundary violation: active session lacks tenant context")
	}
	if tc.TenantID.String() != targetTenantID {
		return fmt.Errorf("security boundary violation: tenant '%s' cannot access resource belonging to '%s'", tc.TenantID.String(), targetTenantID)
	}
	return nil
}

// SetSessionTenantContext injects 'SET LOCAL uisce.current_tenant = ...' into PostgreSQL session
func SetSessionTenantContext(ctx context.Context, db *sql.DB, tenantID string) error {
	if _, err := uuid.Parse(tenantID); err != nil {
		return fmt.Errorf("invalid tenant ID format for RLS session setting: %w", err)
	}
	_, err := db.ExecContext(ctx, "SET LOCAL uisce.current_tenant = $1", tenantID)
	if err != nil {
		return fmt.Errorf("failed to set session RLS parameter uisce.current_tenant: %w", err)
	}
	return nil
}

