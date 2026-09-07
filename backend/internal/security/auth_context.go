package security

import (
	"context"
	"strings"
)

type authInfoKey struct{}

func WithAuthInfo(ctx context.Context, auth AuthInfo) context.Context {
	return context.WithValue(ctx, authInfoKey{}, auth)
}

func AuthInfoFromContext(ctx context.Context) (AuthInfo, bool) {
	value := ctx.Value(authInfoKey{})
	auth, ok := value.(AuthInfo)
	return auth, ok
}

func TenantIDFromContext(ctx context.Context) (string, bool) {
	auth, ok := AuthInfoFromContext(ctx)
	if !ok || len(auth.TenantIDs) == 0 {
		return "", false
	}
	return auth.TenantIDs[0], true
}

// ResolveTenantID is the canonical tenant-resolution rule for this codebase,
// consolidating what this repo's security-sweep (2026-09-07) found as six or
// more independent implementations with varying discipline — including one,
// SecurityContextFromRequest, used at 67 call sites, that accepted a
// client-supplied tenant unconditionally. The rule:
//
//   - auth.TenantIDs (from verified JWT claims) is authoritative.
//   - requested (a client-supplied tenant — X-Tenant-ID header, tenant_id
//     query param, or request-body field) is honored ONLY if it exactly
//     matches one of auth.TenantIDs, or the caller is a verified global
//     admin/ops (auth.IsGlobalAdmin).
//   - Any other case is a mismatch: returns ("", false). There is no
//     default tenant, ever — a caller with no identifiable tenant must be
//     rejected by its handler, not silently attributed to a guessed value.
//     A hardcoded fallback UUID was the single worst idiom this sweep
//     found (bo_crud_handler.go): unauthenticated writes landing under a
//     phantom tenant is a data-integrity failure, not just an access-
//     control one, and it fails silently instead of loudly.
//
// requested == "" is treated as "no override requested": the caller's own
// first tenant (auth.TenantIDs[0]) is returned if present, matching
// TenantIDFromContext's existing behavior for the common case where no
// cross-tenant header is sent at all.
func ResolveTenantID(auth AuthInfo, requested string) (string, bool) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		if len(auth.TenantIDs) == 0 {
			return "", false
		}
		return auth.TenantIDs[0], true
	}
	if auth.IsGlobalAdmin {
		return requested, true
	}
	for _, tid := range auth.TenantIDs {
		if tid == requested {
			return requested, true
		}
	}
	return "", false
}
