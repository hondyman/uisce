package api_test

import (
	"net/http"

	"github.com/hondyman/uisce/backend/internal/security"
)

// withTenantUserAuth gives a request the verified identity AuthContextMiddleware
// would have established: a user with one active tenant. Client-supplied
// X-Tenant-ID / X-User-ID headers are never an identity.
func withTenantUserAuth(req *http.Request, tenantID, userID string) *http.Request {
	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:         userID,
		TenantIDs:      []string{tenantID},
		ActiveTenantID: tenantID,
	})
	return req.WithContext(ctx)
}
