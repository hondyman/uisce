package region_test

import (
	"net/http"

	"github.com/hondyman/uisce/backend/internal/security"
)

// asTenant gives a request the verified active tenant AuthContextMiddleware would
// have established; the region gate takes its tenant from there only.
func asTenant(req *http.Request, tenantID string) *http.Request {
	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID: "user-1", TenantIDs: []string{tenantID}, ActiveTenantID: tenantID,
	})
	return req.WithContext(ctx)
}
