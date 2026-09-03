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
			// 1. Capture client-requested tenant selection before stripping
			clientSelectedTenant := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))

			// 2. Unconditionally strip all spoofable identity and impersonation headers
			r.Header.Del("X-User-ID")
			r.Header.Del("X-Tenant-ID")
			r.Header.Del("X-Real-Admin-ID")
			for k := range r.Header {
				if strings.HasPrefix(strings.ToLower(k), "x-impersonation-") {
					r.Header.Del(k)
				}
			}

			// 3. OPTIONS preflight pass-through
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			if secMgr == nil {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			apiKey := r.Header.Get("X-API-Key")

			if authHeader != "" {
				if !strings.HasPrefix(authHeader, "Bearer ") {
					http.Error(w, `{"error": "unauthorized", "message": "invalid authorization header format"}`, http.StatusUnauthorized)
					return
				}

				rawToken := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
				if rawToken == "" {
					http.Error(w, `{"error": "unauthorized", "message": "empty bearer token"}`, http.StatusUnauthorized)
					return
				}

				dotCount := strings.Count(rawToken, ".")
				switch dotCount {
				case 1:
					// ── Impersonation context token dispatch (payload.signature) ─────────
					impPayload, err := security.ValidateImpersonationToken(rawToken)
					if err != nil {
						logging.GetLogger().Sugar().Warnf("[AuthContextMiddleware] Impersonation token rejected: %v", err)
						http.Error(w, `{"error": "unauthorized", "message": "invalid impersonation token"}`, http.StatusUnauthorized)
						return
					}

					uid := impPayload.Sub
					tenantID := impPayload.TenantID

					// Authoritative identity: the REAL admin, not the target tenant user.
					r.Header.Set("X-User-ID", uid)
					r.Header.Set("X-Tenant-ID", tenantID)

					// Signal to downstream handlers and the frontend that impersonation is active.
					w.Header().Set("X-Impersonation-Active", "true")
					w.Header().Set("X-Real-Admin-ID", uid)
					w.Header().Set("X-Impersonation-Mode", string(impPayload.Mode))

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
					ctx = security.WithImpersonationScope(ctx, security.ImpersonationScopeContext{
						Kind: impPayload.ScopeKind,
						ID:   impPayload.ScopeID,
					})
					r = r.WithContext(ctx)
					next.ServeHTTP(w, r)
					return

				case 2:
					// ── Standard JWT validation (header.payload.signature) ────────────────
					jclaims, err := secMgr.ValidateToken(rawToken)
					if err != nil {
						logging.GetLogger().Sugar().Warnf("[AuthContextMiddleware] JWT validation failed: %v", err)
						http.Error(w, `{"error": "unauthorized", "message": "invalid token"}`, http.StatusUnauthorized)
						return
					}

					var issuerTrustedTenants []string
					if idpRegistry != nil {
						trusted, err := services.ValidateIssuerTenant(r.Context(), idpRegistry, jclaims)
						if err != nil {
							logging.GetLogger().Sugar().Warnf("[AuthContextMiddleware] issuer/tenant validation failed: %v", err)
							http.Error(w, `{"error": "unauthorized", "message": "untrusted token issuer or tenant mismatch"}`, http.StatusUnauthorized)
							return
						}
						issuerTrustedTenants = trusted
					}

					logging.GetLogger().Sugar().Infof("[AuthContextMiddleware] Auth event: sub=%s, alg=%s, iss=%s", jclaims.UserID, jclaims.Alg, jclaims.Issuer)

					uid := jclaims.UserID
					if uid == "" {
						http.Error(w, `{"error": "unauthorized", "message": "missing subject in token"}`, http.StatusUnauthorized)
						return
					}

					r.Header.Set("X-User-ID", uid)

					isGlobalAdmin := hasRole(normalizeStringList(jclaims.Roles), "global_admin") ||
						hasRole(normalizeStringList(jclaims.Roles), "global_ops")
					if !isGlobalAdmin && len(normalizeStringList(jclaims.Roles)) > 0 {
						for _, role := range normalizeStringList(jclaims.Roles) {
							if role == "core_admin" || role == "is_core_admin" {
								isGlobalAdmin = true
								break
							}
						}
					}

					tenantID := strings.TrimSpace(jclaims.TenantID)
					tenantIDs := normalizeTenantIDs(jclaims.TenantIDs, tenantID)
					if tenantID != "" {
						r.Header.Set("X-Tenant-ID", tenantID)
					} else if len(tenantIDs) == 1 {
						tenantID = tenantIDs[0]
						r.Header.Set("X-Tenant-ID", tenantID)
					} else if len(tenantIDs) == 0 && isGlobalAdmin {
						parsed, parseErr := uuid.Parse(clientSelectedTenant)
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
							logging.GetLogger().Sugar().Infof("[AuthContextMiddleware] Global admin token lacks tenant claim; trusting sanitized client X-Tenant-ID: tenant=%s user=%s", tenantID, uid)
						} else {
							if parseErr == nil && !allowedByIssuer {
								logging.GetLogger().Sugar().Warnf("[AuthContextMiddleware] Global admin selected tenant=%s not in issuer's registered set; rejecting: user=%s", clientSelectedTenant, uid)
							} else {
								logging.GetLogger().Sugar().Warnf("[AuthContextMiddleware] Global admin token lacks tenant claim and client X-Tenant-ID is missing or not a valid UUID: header=%q user=%s", clientSelectedTenant, uid)
							}
							r.Header.Del("X-Tenant-ID")
						}
					} else if len(tenantIDs) == 0 {
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
					next.ServeHTTP(w, r)
					return

				default:
					http.Error(w, `{"error": "unauthorized", "message": "malformed bearer token structure"}`, http.StatusUnauthorized)
					return
				}
			} else if apiKey != "" {
				ak, ok := secMgr.GetAPIKey(apiKey)
				if !ok || ak == nil {
					logging.GetLogger().Sugar().Warnf("[AuthContextMiddleware] Invalid API key presented")
					http.Error(w, `{"error": "unauthorized", "message": "invalid api key"}`, http.StatusUnauthorized)
					return
				}

				uid := ak.UserID
				if uid == "" {
					http.Error(w, `{"error": "unauthorized", "message": "invalid api key user mapping"}`, http.StatusUnauthorized)
					return
				}

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
				next.ServeHTTP(w, r)
				return
			}

			// No credentials presented: headers are stripped, proceed anonymously
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
