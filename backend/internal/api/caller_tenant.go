package api

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
)

// callerTenant returns the tenant a request operates on, writing the error
// response itself when there is none.
//
// The tenant comes from verified authentication (AuthContextMiddleware). A tenant
// named by the client in a query parameter or request body is only a *request*:
// it is honored when it equals the verified tenant, or when the caller is a
// verified global admin selecting a tenant explicitly. Anything else is refused.
// There is no default tenant: a request without an established tenant is
// rejected, never attributed to a guessed one (including the gold copy).
func callerTenant(w http.ResponseWriter, r *http.Request, requested ...string) (string, bool) {
	var clientTenant string
	for _, v := range requested {
		if v = strings.TrimSpace(v); v != "" {
			clientTenant = v
			break
		}
	}
	auth, hasAuth := security.AuthInfoFromContext(r.Context())
	verified, hasVerified := TenantIDFromRequest(r)

	if clientTenant == "" {
		if !hasVerified {
			http.Error(w, "unauthorized: no tenant established for this request", http.StatusUnauthorized)
			return "", false
		}
		return verified, true
	}
	if _, err := uuid.Parse(clientTenant); err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return "", false
	}
	if hasVerified && clientTenant == verified {
		return verified, true
	}
	if hasAuth && auth.IsGlobalAdmin {
		return clientTenant, true
	}
	if !hasAuth && !hasVerified {
		http.Error(w, "unauthorized: no tenant established for this request", http.StatusUnauthorized)
		return "", false
	}
	// A caller authorized for several tenants who has not otherwise selected one
	// may name one of them; the token authorizes it. Any other tenant is refused.
	if !hasVerified && hasAuth {
		for _, tid := range auth.TenantIDs {
			if tid == clientTenant {
				return clientTenant, true
			}
		}
	}
	http.Error(w, "forbidden: tenant_id does not match the authenticated tenant", http.StatusForbidden)
	return "", false
}
