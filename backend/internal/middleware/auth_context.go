package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/identity"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/services"
)

// AuthContextMiddleware returns a chi-compatible middleware that validates
// an Authorization Bearer token using SecurityManager and injects actor/tenant
// into the request context. If validation fails the request continues but no
// actor is set (handlers should enforce auth as needed).
//
// Impersonation token detection: the middleware first attempts to parse the
// Bearer token as a platform-internal impersonation context token (HMAC-SHA256).
// If it matches, the request context is populated with the TARGET tenant_id as
// the concrete tenant identifier — meaning all downstream RLS, ABAC, and
// BuildContext logic runs identically to a normal tenant-scoped request.
func AuthContextMiddleware(secMgr *services.SecurityManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// X-Tenant-ID and X-User-ID are identity assertions that only this
			// middleware may establish, from a verified token or API key. Whatever the
			// client sent is discarded up front: the client's tenant is kept only as a
			// *requested selection*, honored below solely if the caller is authorized
			// for it. A request with no token, an invalid token or an unauthorized
			// selection therefore reaches handlers with no tenant and no user, and
			// fails. There are no fallbacks.
			clientTenant := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
			r.Header.Del("X-Tenant-ID")
			r.Header.Del("X-User-ID")

			if secMgr != nil {
				authHeader := r.Header.Get("Authorization")
				if authHeader != "" {
					// ── Check for impersonation context token first ──────────
					// Impersonation tokens are HMAC-signed platform-internal tokens
					// that carry a concrete target tenant_id and impersonation_active=true.
					// If this succeeds, the downstream security context behaves exactly
					// like a normal tenant user — no special-casing required.
					rawToken := strings.TrimPrefix(authHeader, "Bearer ")
					if impPayload, err := security.ValidateImpersonationToken(rawToken); err == nil {
						uid := impPayload.Sub
						tenantID := impPayload.TenantID

						// Authoritative identity: the REAL admin, not the target tenant user.
						r.Header.Set("X-User-ID", uid)
						r.Header.Set("X-Tenant-ID", tenantID)

						// Signal to downstream handlers and the frontend that impersonation is active.
						w.Header().Set("X-Impersonation-Active", "true")
						w.Header().Set("X-Real-Admin-ID", uid)
						w.Header().Set("X-Impersonation-Mode", string(impPayload.Mode))

						// Preserve the real admin roles so downstream authorization and audit
						// know which role initiated the session. Fall back to the token's
						// admin_role for tokens that do not carry RealRoles. Legacy tokens
						// issued before multi-role support have neither field; treat them as
						// global_admin because the platform previously allowed only global
						// admins to impersonate.
						realRoles := impPayload.RealRoles
						adminRole := impPayload.AdminRole
						if len(realRoles) == 0 {
							if adminRole != "" {
								realRoles = []string{adminRole}
							} else {
								realRoles = []string{security.RoleGlobalAdmin}
								adminRole = security.RoleGlobalAdmin
							}
						}
						isGlobalAdmin := hasRole(realRoles, security.RoleGlobalAdmin) ||
							hasRole(realRoles, security.RoleGlobalOps) ||
							hasRole(realRoles, "core_admin") ||
							hasRole(realRoles, "is_core_admin")

						ctx := identity.WithActorTenant(r.Context(), uid, tenantID)
						ctx = security.WithAuthInfo(ctx, security.AuthInfo{
							UserID:                 uid,
							Roles:                  realRoles,
							TenantIDs:              []string{tenantID},
							ActiveTenantID:         tenantID,
							RequestedTenantID:      clientTenant,
							IsGlobalAdmin:          isGlobalAdmin,
							ImpersonationActive:    true,
							RealAdminUserID:        uid,
							ImpersonationSessionID: impPayload.SessionID,
							ImpersonationMode:      string(impPayload.Mode),
							ImpersonationAdminRole: adminRole,
						})
						// Attach the impersonation scope so BuildContext can enforce it.
						// Default to tenant-wide; honour the token's scope_kind/scope_id when set.
						ctx = security.WithImpersonationScope(ctx, security.ImpersonationScopeContext{
							Kind: impPayload.ScopeKind,
							ID:   impPayload.ScopeID,
						})
						r = r.WithContext(ctx)
						next.ServeHTTP(w, r)
						return
					}

					// ── Standard JWT validation ──────────────────────────────
					if jclaims, err := secMgr.ValidateToken(rawToken); err == nil {
						uid := jclaims.UserID
						if uid != "" {
							// Inject UserID into headers for legacy handlers that rely on it
							r.Header.Set("X-User-ID", uid)

							// isGlobalAdmin is true for global_admin, global_ops, or the legacy is_core_admin flag.
							// Computed before tenant resolution below: a verified global admin's JWT
							// legitimately carries no tenant claim (they aren't scoped to one tenant), so
							// the client-supplied X-Tenant-ID header is the only way they can select a
							// tenant to operate on. SecurityContextFromRequest / ResolveTenantID already
							// treat that header as untrusted for everyone else and independently verify
							// auth.IsGlobalAdmin from the role claims before honoring it - so trusting the
							// header here for a verified admin does not reopen the tenant-header-spoofing
							// hole the 2026-09-07 hotfix closed for non-admin callers.
							isGlobalAdmin := hasRole(normalizeStringList(jclaims.Roles), "global_admin") ||
								hasRole(normalizeStringList(jclaims.Roles), "global_ops")
							// Backward-compat: also accept legacy "core_admin" / "is_core_admin" claim if present.
							if !isGlobalAdmin && len(normalizeStringList(jclaims.Roles)) > 0 {
								for _, role := range normalizeStringList(jclaims.Roles) {
									if role == "core_admin" || role == "is_core_admin" {
										isGlobalAdmin = true
										break
									}
								}
							}

							// Establish the active tenant from verified sources only. Every other
							// case leaves it empty and the request fails downstream; there is no
							// default tenant and no guessing.
							claimTenant := strings.TrimSpace(jclaims.TenantID)
							tenantIDs := normalizeTenantIDs(jclaims.TenantIDs, claimTenant)
							tenantID := ""
							switch {
							case claimTenant != "":
								tenantID = claimTenant // explicit tenant claim
							case len(tenantIDs) == 1:
								tenantID = tenantIDs[0] // unambiguous
							case len(tenantIDs) > 1:
								// Several tenants: the caller must select one, and the selection
								// counts only if the token authorizes it.
								if clientTenant != "" && containsString(tenantIDs, clientTenant) {
									tenantID = clientTenant
								} else {
									logging.GetLogger().Sugar().Warnf("[AuthContextMiddleware] multi-tenant caller user=%s sent no valid tenant selection (requested=%q); no active tenant", uid, clientTenant)
								}
							case isGlobalAdmin:
								// A verified global admin/ops caller has no tenant claim and selects
								// the tenant to operate on explicitly. Must be a valid UUID.
								if parsed, err := uuid.Parse(clientTenant); err == nil {
									tenantID = parsed.String()
									tenantIDs = []string{tenantID}
								}
							}
							if tenantID != "" {
								r.Header.Set("X-Tenant-ID", tenantID)
							}

							ctx := identity.WithActorTenant(r.Context(), uid, tenantID)
							ctx = security.WithAuthInfo(ctx, security.AuthInfo{
								UserID:         uid,
								Roles:          normalizeStringList(jclaims.Roles),
								TenantIDs:      tenantIDs,
								ActiveTenantID: tenantID,
								IsGlobalAdmin:  isGlobalAdmin,
								RawClaims:      jclaims,
							})
							r = r.WithContext(ctx)
						}
					}
				} else if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
					if ak, ok := secMgr.GetAPIKey(apiKey); ok && ak != nil {
						uid := ak.UserID
						if uid != "" {
							r.Header.Set("X-User-ID", uid)

							tenantID := ak.TenantID
							if tenantID != "" {
								r.Header.Set("X-Tenant-ID", tenantID)
							}

							ctx := identity.WithActorTenant(r.Context(), uid, tenantID)
							ctx = security.WithAuthInfo(ctx, security.AuthInfo{
								UserID:            uid,
								Roles:             normalizeStringList(ak.Roles),
								TenantIDs:         normalizeTenantIDs(ak.TenantIDs, tenantID),
								ActiveTenantID:    tenantID,
								RequestedTenantID: clientTenant,
								RawClaims:         nil,
							})
							r = r.WithContext(ctx)
						}
					}
				}

			}
			next.ServeHTTP(w, r)
		})
	}
}

func normalizeTenantIDs(values []string, fallback string) []string {
	result := []string{}
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 && strings.TrimSpace(fallback) != "" {
		result = append(result, strings.TrimSpace(fallback))
	}
	return result
}

func normalizeStringList(values []string) []string {
	result := []string{}
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func containsString(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}
