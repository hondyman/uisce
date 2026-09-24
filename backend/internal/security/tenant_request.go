package security

import (
	"fmt"
	"net/http"
	"strings"
)

// ResolveTenantForRequest resolves the tenant for an HTTP request by delegating to
// ResolveTenantID (the single canonical rule). It is the only tenant-resolution
// path that WithTenantContext may use; no other implementation is permitted.
//
// Behavior:
//   - If no X-Tenant-ID header is present: returns auth.TenantIDs[0] from JWT.
//   - If X-Tenant-ID header is present:
//   - Caller is global_admin or global_ops → header value honored (admin override).
//   - Header matches one of auth.TenantIDs → header value honored.
//   - Otherwise → returns error (fail-closed: spoof attempt rejected).
//   - No valid AuthInfo in context → returns error (401 equivalent).
//
// This function deliberately does NOT fall back to identity.TenantIDFromContext
// or any other source. Only ResolveTenantID is authoritative.
func ResolveTenantForRequest(r *http.Request) (string, error) {
	auth, ok := AuthInfoFromContext(r.Context())
	if !ok {
		return "", fmt.Errorf("tenant resolution requires authentication context")
	}

	requested := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	tenant, ok := ResolveTenantID(auth, requested)
	if !ok {
		if requested != "" {
			return "", fmt.Errorf("tenant %q is not authorized for this request", requested)
		}
		return "", fmt.Errorf("no tenant available: JWT token must include tenant_id or tenant_ids claim")
	}

	return tenant, nil
}
