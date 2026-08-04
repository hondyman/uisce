package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/identity"
	"github.com/hondyman/uisce/backend/internal/security"
)

type marketplaceScope string

const (
	ScopeMarketplaceInstall  marketplaceScope = "marketplace:install"
	ScopeMarketplacePublish marketplaceScope = "marketplace:publish"
	ScopeMarketplaceAdmin   marketplaceScope = "marketplace:admin"
	ScopeMarketplaceAudit   marketplaceScope = "marketplace:auditor"
)

// RequireMarketplaceScope returns a chi middleware that enforces exactly-one
// marketplace permission scope.  It is a hard 401/403 gate — there is no Gold Copy
// fallback, no implicit read, and no tenant-header trust.
//
// Required context values (populated by AuthContextMiddleware):
//   - security.AuthInfo  (from security.AuthInfoFromContext)
//   - identity.TenantID  (from identity.TenantIDFromContext)
//
// Returns:
//   401 Unauthorized  – no AuthInfo in context (no valid JWT/API-key)
//   403 Forbidden     – AuthInfo present but actor lacks the required scope
//   next handler      – authorized
func RequireMarketplaceScope(required marketplaceScope) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 1. AuthInfo MUST be present — no Gold Copy fallback.
			authInfo, ok := security.AuthInfoFromContext(ctx)
			if !ok || authInfo.UserID == "" {
				http.Error(w, `{"error":"unauthorized","message":"Valid authentication required"}`,
					http.StatusUnauthorized)
				return
			}

			// 2. Scope check.
			if !marketplaceScopeSatisfied(authInfo, required) {
				http.Error(w, `{"error":"forbidden","message":"Insufficient marketplace permission: `+
					string(required)+`"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// marketplaceScopeSatisfied returns true when authInfo grants the required scope.
// A global_admin or global_ops bypasses all marketplace scope checks.
// Otherwise the actor must hold the exact role matching the required scope
// MarketplaceScopeSatisfied checks whether the given context's AuthInfo
// grants the required marketplace scope.  Use this in handlers that need
// a tighter scope than the group-level middleware enforces.
func MarketplaceScopeSatisfied(ctx context.Context, required marketplaceScope) bool {
	info, ok := security.AuthInfoFromContext(ctx)
	if !ok {
		return false
	}
	return marketplaceScopeSatisfied(info, required)
}

// marketplaceScopeSatisfied is the unexported implementation.
func marketplaceScopeSatisfied(info security.AuthInfo, required marketplaceScope) bool {
	if info.IsGlobalAdmin {
		return true
	}
	return hasMarketplaceRole(info.Roles, required)
}

// hasMarketplaceRole returns true when the role list contains the role that
// corresponds to the given marketplace scope.
func hasMarketplaceRole(roles []string, scope marketplaceScope) bool {
	required := scopeToRole(scope)
	for _, r := range roles {
		if r == required || r == "global_admin" || r == "global_ops" {
			return true
		}
	}
	return false
}

// scopeToRole maps a marketplace scope string to the bp_roles.role_key that
// grants it.  These must match the Gold Copy roles seeded by the migration.
func scopeToRole(scope marketplaceScope) string {
	switch scope {
	case ScopeMarketplaceInstall:
		return "marketplace_publisher" // publishers can install; admins can too
	case ScopeMarketplacePublish:
		return "marketplace_publisher"
	case ScopeMarketplaceAdmin:
		return "marketplace_admin"
	case ScopeMarketplaceAudit:
		return "marketplace_auditor"
	default:
		return ""
	}
}

// MarketplaceAuthFromRequest extracts actor + tenant IDs from the request context
// that has already been processed by AuthContextMiddleware.  Handlers MUST use
// this instead of reading X-Tenant-ID or X-User-ID headers directly.
func MarketplaceAuthFromRequest(r *http.Request) (actorID, tenantID string, ok bool) {
	ctx := r.Context()
	authInfo, ok := security.AuthInfoFromContext(ctx)
	if !ok || authInfo.UserID == "" {
		return "", "", false
	}
	actorID = authInfo.UserID

	tid, ok := identity.TenantIDFromContext(ctx)
	if !ok || tid == "" {
		// Derive from AuthInfo.TenantIDs when identity context is absent.
		if len(authInfo.TenantIDs) > 0 {
			tid = authInfo.TenantIDs[0]
		}
	}
	return actorID, tid, actorID != ""
}

// GetListingID reads the {listingId} path parameter from the HTTP request.
// Returns empty string if the parameter is absent.
func GetListingID(r *http.Request) string {
	return chi.URLParam(r, "listingId")
}
