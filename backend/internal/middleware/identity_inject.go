package middleware

import (
	"net/http"

	"github.com/hondyman/semlayer/backend/internal/identity"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// InjectActorTenantFromHeaders is a simple middleware that reads X-User-ID/X-Actor-ID and X-Tenant-ID
// from request headers and stores them into the request context so downstream handlers can use identity.ActorIDFromContext
func InjectActorTenantFromHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor := r.Header.Get("X-User-ID")
		if actor == "" {
			actor = r.Header.Get("X-Actor-ID")
		}
		tenant := jwtmiddleware.GetClaimsFromContext(r).TenantID
		ctx := identity.WithActorTenant(r.Context(), actor, tenant)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
