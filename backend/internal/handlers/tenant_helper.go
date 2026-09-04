package handlers

import (
	"net/http"

	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

// getSecureTenantID resolves the tenant for the request. It NEVER trusts the
// client-supplied X-Tenant-ID header directly: the header is only honored
// when security.AuthInfoFromContext (populated by AuthContextMiddleware from
// a verified JWT) confirms the caller is a global admin/ops, or already
// names that exact tenant among the caller's JWT-issued tenants. Note:
// jwtmiddleware.GetClaimsFromContext is dead in this app (no middleware ever
// populates it), so that branch was previously equivalent to always trusting
// the raw header — the real vulnerability this function had.
func getSecureTenantID(r *http.Request) string {
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		return claims.TenantID
	}

	auth, ok := security.AuthInfoFromContext(r.Context())
	if !ok {
		return ""
	}

	headerTenant := r.Header.Get("X-Tenant-ID")
	if headerTenant == "" {
		if len(auth.TenantIDs) > 0 {
			return auth.TenantIDs[0]
		}
		return ""
	}

	if auth.IsGlobalAdmin {
		return headerTenant
	}
	for _, tid := range auth.TenantIDs {
		if tid == headerTenant {
			return headerTenant
		}
	}
	return ""
}
