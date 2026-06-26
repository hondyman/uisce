package middleware

import (
	"net/http"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/identity"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// AuthContextMiddleware returns a chi-compatible middleware that validates
// an Authorization Bearer token using SecurityManager and injects actor/tenant
// into the request context. If validation fails the request continues but no
// actor is set (handlers should enforce auth as needed).
func AuthContextMiddleware(secMgr *services.SecurityManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secMgr != nil {
				authHeader := r.Header.Get("Authorization")
				if authHeader != "" {
					if jclaims, err := secMgr.ValidateToken(authHeader); err == nil {
						uid := jclaims.UserID
						if uid != "" {
							// Inject UserID into headers for legacy handlers that rely on it
							r.Header.Set("X-User-ID", uid)

							// Authoritative Tenant ID from token
							tenantID := strings.TrimSpace(jclaims.TenantID)
							tenantIDs := normalizeTenantIDs(jclaims.TenantIDs, tenantID)
							if tenantID != "" {
								// Override header with authoritative value from token if present.
								// If missing from token, we do NOT fallback to header to prevent injection.
								r.Header.Set("X-Tenant-ID", tenantID)
							} else if len(tenantIDs) == 1 {
								r.Header.Set("X-Tenant-ID", tenantIDs[0])
							}

							ctx := identity.WithActorTenant(r.Context(), uid, tenantID)
							ctx = security.WithAuthInfo(ctx, security.AuthInfo{
								UserID:    uid,
								Roles:     normalizeStringList(jclaims.Roles),
								TenantIDs: tenantIDs,
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
