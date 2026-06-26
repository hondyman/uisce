package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/scheduler_intelligence"
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
