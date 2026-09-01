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
// idpRegistry, when non-nil, additionally enforces that an externally-issued
// token's IDP is actually registered for the tenant(s) it claims (see
// services.ValidateIssuerTenant) — pass nil to skip this check (e.g. in a
// test harness with no tenant_identity_providers configured).
//
// Impersonation token detection: the middleware first attempts to parse the
// Bearer token as a platform-internal impersonation context token (HMAC-SHA256).
// If it matches, the request context is populated with the TARGET tenant_id as
// the concrete tenant identifier — meaning all downstream RLS, ABAC, and
// BuildContext logic runs identically to a normal tenant-scoped request.
func AuthContextMiddleware(secMgr *services.SecurityManager, idpRegistry security.IssuerRegistry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
						// Full tenant isolation for externally-issued tokens: confirm
						// the signing IDP is actually registered for the tenant(s)
						// the token claims (see services.ValidateIssuerTenant). A
						// valid signature alone only proves "a registered IDP signed
						// this" — not that this is the IDP this tenant trusts, which
						// matters once a tenant can have more than one registered
						// realm (M&A) or an issuer is deliberately cross-tenant
						// (super-admin / professional-services). No-op for internal
						// HS256 tokens (jclaims.Issuer is empty for those).
						//
						// issuerTrustedTenants, when non-nil, is the full set of
						// tenants this token's issuer is registered for — used below
						// to constrain the global-admin X-Tenant-ID-header path so a
						// cross-tenant-realm admin can only select a tenant their
						// issuer actually covers, not an arbitrary UUID.
						var issuerTrustedTenants []string
						if idpRegistry != nil {
							trusted, err := services.ValidateIssuerTenant(r.Context(), idpRegistry, jclaims)
							if err != nil {
								logging.GetLogger().Sugar().Warnf("[AuthContextMiddleware] issuer/tenant validation failed: %v", err)
								next.ServeHTTP(w, r)
								return
							}
							issuerTrustedTenants = trusted
						}
						uid := jclaims.UserID
						if uid != "" {
							// Inject UserID into headers for legacy handlers that rely on it
							r.Header.Set("X-User-ID", uid)

							// isGlobalAdmin is true for global_admin, global_ops, or the legacy is_core_admin flag.
							// Computed up front because it gates whether the client-supplied
							// X-Tenant-ID header may be trusted below.
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

							// Authoritative Tenant ID from token
							tenantID := strings.TrimSpace(jclaims.TenantID)
							tenantIDs := normalizeTenantIDs(jclaims.TenantIDs, tenantID)
							if tenantID != "" {
								// Override header with authoritative value from token if present.
								r.Header.Set("X-Tenant-ID", tenantID)
							} else if len(tenantIDs) == 1 {
								tenantID = tenantIDs[0]
								r.Header.Set("X-Tenant-ID", tenantID)
							} else if len(tenantIDs) == 0 && isGlobalAdmin {
								// The token carries no tenant claim. Global admins operate across all
								// tenants and pick the active one via the UI, so the client-supplied
								// X-Tenant-ID header is the only source of that selection. Trusting it
								// here is safe specifically because it is gated on the verified
								// global_admin/global_ops role from the signed token — a non-admin
								// token can never reach this branch and cannot spoof a tenant via the
								// header. Still parsed as a UUID to reject malformed/injected values.
								headerTenant := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
								parsed, parseErr := uuid.Parse(headerTenant)
								// When this token came from a registered external issuer,
								// issuerTrustedTenants is that issuer's full registered set —
								// a cross-tenant-realm admin may only select a tenant their
								// issuer actually covers, not an arbitrary UUID. Internal
								// (HS256) global-admin tokens have issuerTrustedTenants == nil
								// and keep the prior, unconstrained behavior.
								allowedByIssuer := issuerTrustedTenants == nil
								if !allowedByIssuer && parseErr == nil {
									for _, t := range issuerTrustedTenants {
										if t == parsed.String() {
											allowedByIssuer = true
											break
										}
									}
								}
								if parseErr == nil && allowedByIssuer {
									tenantID = parsed.String()
									r.Header.Set("X-Tenant-ID", tenantID)
									tenantIDs = []string{tenantID}
									logging.GetLogger().Sugar().Infof("[AuthContextMiddleware] Global admin token lacks tenant claim; trusting X-Tenant-ID header: tenant=%s user=%s", tenantID, uid)
								} else {
									if parseErr == nil && !allowedByIssuer {
										logging.GetLogger().Sugar().Warnf("[AuthContextMiddleware] Global admin selected tenant=%s not in issuer's registered set; rejecting: user=%s", headerTenant, uid)
									} else {
										logging.GetLogger().Sugar().Warnf("[AuthContextMiddleware] Global admin token lacks tenant claim and X-Tenant-ID header is missing or not a valid UUID: header=%q user=%s", headerTenant, uid)
									}
									r.Header.Del("X-Tenant-ID")
								}
							} else if len(tenantIDs) == 0 {
								// Non-admin token with no tenant claim: never trust a client-supplied
								// header, since there is no verified entitlement to check it against.
								r.Header.Del("X-Tenant-ID")
							}

							ctx := identity.WithActorTenant(r.Context(), uid, tenantID)
							ctx = security.WithAuthInfo(ctx, security.AuthInfo{
								UserID:        uid,
								Roles:         normalizeStringList(jclaims.Roles),
								TenantIDs:     tenantIDs,
								IsGlobalAdmin: isGlobalAdmin,
								RawClaims:     jclaims,
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
								UserID:    uid,
								Roles:     normalizeStringList(ak.Roles),
								TenantIDs: normalizeTenantIDs(ak.TenantIDs, tenantID),
								RawClaims: nil,
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
