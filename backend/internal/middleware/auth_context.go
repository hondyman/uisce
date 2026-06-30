package middleware

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/identity"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/internal/services"
)

type contextKey string

const (
	UserEmailKey contextKey = "user_email"
)

// AuthContextMiddleware returns a chi-compatible middleware that validates
// an Authorization Bearer token using SecurityManager and injects actor/tenant
// into the request context. If validation fails the request continues but no
// actor is set (handlers should enforce auth as needed).
//
// When profileSvc is provided, the middleware also enriches the request context
// with the user's abstract security profile: internal Uisce operator roles are
// taken from the uisce_metadata.operator_role claim, while tenant-scoped
// enterprise IdP groups are mapped via security.identity_profile_mappings to a
// functional role and clearance level.
//
// Impersonation token detection: the middleware first attempts to parse the
// Bearer token as a platform-internal impersonation context token (HMAC-SHA256).
// If it matches, the request context is populated with the TARGET tenant_id as
// the concrete tenant identifier — meaning all downstream RLS, ABAC, and
// BuildContext logic runs identically to a normal tenant-scoped request.
func AuthContextMiddleware(secMgr *services.SecurityManager, profileSvc *security.ProfileService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secMgr == nil {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				// ── Check for impersonation context token first ──────────
				rawToken := strings.TrimPrefix(authHeader, "Bearer ")
				if impPayload, err := security.ValidateImpersonationToken(rawToken); err == nil {
					uid := impPayload.Sub
					tenantID := impPayload.TenantID

					r.Header.Set("X-User-ID", uid)
					r.Header.Set("X-Tenant-ID", tenantID)

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

					if isGlobalAdmin {
						r.Header.Set("X-Is-Core-Admin", "true")
					}
					if adminRole != "" {
						r.Header.Set("X-User-Role", adminRole)
					}

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
				}

				// ── Standard JWT validation ──────────────────────────────
				if jclaims, err := secMgr.ValidateToken(authHeader); err == nil {
					uid := jclaims.UserID
					if uid == "" {
						next.ServeHTTP(w, r)
						return
					}

					// Inject UserID into headers for legacy handlers that rely on it
					r.Header.Set("X-User-ID", uid)

					// Authoritative Tenant ID from token
					tenantID := strings.TrimSpace(jclaims.TenantID)
					tenantIDs := normalizeTenantIDs(jclaims.TenantIDs, tenantID)
					if tenantID != "" {
						r.Header.Set("X-Tenant-ID", tenantID)
					} else if len(tenantIDs) == 1 {
						tenantID = tenantIDs[0]
						r.Header.Set("X-Tenant-ID", tenantID)
					}

					roles := normalizeStringList(jclaims.Roles)
					functionalRole := strings.TrimSpace(jclaims.OperatorRole)
					clearanceLevel := "L1"
					isGlobalAdmin := false

					// Internal Uisce staff carry an explicit operator_role in the token.
					if functionalRole != "" {
						roles = appendIfMissing(roles, functionalRole)
						isGlobalAdmin = functionalRole == "global_admin" ||
							functionalRole == "global_ops" ||
							functionalRole == "core_admin" ||
							functionalRole == "is_core_admin"
						// Helpdesk / professional services are lease-scoped support roles,
						// not global platform operators. They must not receive the
						// X-Is-Core-Admin header that would bypass tenant filtering.

						// Enforce support lease validation for professional services and helpdesk operators
						if (functionalRole == "professional_services" || functionalRole == "helpdesk") && profileSvc != nil {
							path := r.URL.Path
							isPlatformAPI := strings.HasPrefix(path, "/api/tenants/") ||
								strings.HasPrefix(path, "/api/admin/") ||
								strings.HasPrefix(path, "/api/rbac/")

							if !isPlatformAPI {
								requestedTenant := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
								if requestedTenant == "" {
									http.Error(w, "Forbidden: Ambient Power Prohibited. Explicitly select a target Tenant ID.", http.StatusForbidden)
									return
								}

								tid, err := uuid.Parse(requestedTenant)
								if err != nil {
									http.Error(w, "Forbidden: Invalid target Tenant ID format.", http.StatusForbidden)
									return
								}

								_, err = profileSvc.VerifyStaffAssignment(r.Context(), jclaims.Email, tid)
								if err != nil {
									http.Error(w, "Forbidden: No active data lease assignment exists for this target tenant.", http.StatusForbidden)
									return
								}

								// Lease verified! Set tenant context.
								tenantID = requestedTenant
								r.Header.Set("X-Tenant-ID", tenantID)
								tenantIDs = []string{tenantID}
							}
						}
					} else if profileSvc != nil && jclaims.ClientID != "" && len(jclaims.IDPGroups) > 0 {
						// Branch B1: Bulletproof Client-to-Tenant group federation mapping
						resolvedTenant, resolvedRole, err := profileSvc.ResolveTenantAndRole(r.Context(), jclaims.ClientID, jclaims.IDPGroups)
						if err != nil {
							if errors.Is(err, sql.ErrNoRows) {
								http.Error(w, "Forbidden: Identity claims and client origin do not map to a recognized production tenant profile.", http.StatusForbidden)
								return
							}
							http.Error(w, "Internal Database System Error", http.StatusInternalServerError)
							return
						}

						// Strip any incoming client-provided X-Tenant-ID header overrides to mitigate manipulation vectors
						r.Header.Del("X-Tenant-ID")

						tenantID = resolvedTenant
						r.Header.Set("X-Tenant-ID", tenantID)
						tenantIDs = []string{tenantID}
						functionalRole = resolvedRole
						roles = appendIfMissing(roles, functionalRole)
					} else if profileSvc != nil && tenantID != "" && len(jclaims.IDPGroups) > 0 {
						// Branch B2: Legacy tenant-scoped group mapping (using token tenant_id)
						tid, err := uuid.Parse(tenantID)
						if err == nil {
							if fr, cl, err := profileSvc.EnrichSubjectAttributes(r.Context(), tid, uid, jclaims.IDPGroups); err == nil {
								functionalRole = fr
								clearanceLevel = cl
								roles = appendIfMissing(roles, functionalRole)
							}
						}
					}

					// Fallback: If tenantID is still empty, resolve tenant from local database mapping (e.g. for Alice)
					if tenantID == "" && profileSvc != nil {
						if dbTenant, err := profileSvc.GetTenantIDByUser(r.Context(), uid, jclaims.Email); err == nil && dbTenant != "" {
							tenantID = dbTenant
							r.Header.Set("X-Tenant-ID", tenantID)
							tenantIDs = []string{tenantID}
						}
					}

					// Fallback global-admin detection from legacy role claims.
					if !isGlobalAdmin {
						isGlobalAdmin = hasRole(roles, "global_admin") || hasRole(roles, "global_ops") ||
							hasRole(roles, "core_admin") || hasRole(roles, "is_core_admin")
					}

					// Expose enriched attributes as headers for legacy handlers.
					if functionalRole != "" {
						r.Header.Set("X-User-Role", functionalRole)
						r.Header.Set("X-User-Permissions", functionalRole)
					}
					if clearanceLevel != "" {
						r.Header.Set("X-Clearance-Level", clearanceLevel)
					}
					if isGlobalAdmin {
						r.Header.Set("X-Is-Core-Admin", "true")
					}
					if jclaims.Email != "" {
						r.Header.Set("X-User-Email", jclaims.Email)
					}

					ctx := identity.WithActorTenant(r.Context(), uid, tenantID)
					ctx = context.WithValue(ctx, UserEmailKey, jclaims.Email)
					ctx = security.WithAuthInfo(ctx, security.AuthInfo{
						UserID:         uid,
						Email:          jclaims.Email,
						Roles:          roles,
						TenantIDs:      tenantIDs,
						IsGlobalAdmin:  isGlobalAdmin,
						FunctionalRole: functionalRole,
						ClearanceLevel: clearanceLevel,
						IDPGroups:      jclaims.IDPGroups,
					})
					r = r.WithContext(ctx)
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
						})
						r = r.WithContext(ctx)
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func appendIfMissing(list []string, value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return list
	}
	for _, item := range list {
		if strings.EqualFold(strings.TrimSpace(item), trimmed) {
			return list
		}
	}
	return append(list, trimmed)
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
