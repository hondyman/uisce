package security

import "strings"

// ActiveTenant returns the tenant this request operates on, or ("", false) when
// none has been established. It never guesses: a caller authorized for several
// tenants who has not explicitly selected one (ActiveTenantID empty) has no active
// tenant, and the request must fail. A caller with exactly one tenant has an
// unambiguous one.
func (a AuthInfo) ActiveTenant() (string, bool) {
	if t := strings.TrimSpace(a.ActiveTenantID); t != "" {
		return t, true
	}
	if len(a.TenantIDs) == 1 {
		if t := strings.TrimSpace(a.TenantIDs[0]); t != "" {
			return t, true
		}
	}
	return "", false
}
