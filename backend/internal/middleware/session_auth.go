package middleware

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/auth"
	imodels "github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/requestcontext"
	"github.com/hondyman/semlayer/backend/internal/utils/ip"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// SessionAuthConfig holds configuration for the session auth middleware
type SessionAuthConfig struct {
	DB                  *sql.DB
	SessionCookie       string
	AllowBearerFallback bool
}

// SessionAuthMiddleware validates a user session from a secure httpOnly cookie and enriches the RequestContext.
func SessionAuthMiddleware(cfg SessionAuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If no DB configured, skip session handling and allow handlers to enforce auth
			if cfg.DB == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Extract token from cookie or optional bearer header
			token := ""
			if c, err := r.Cookie(cfg.SessionCookie); err == nil {
				token = c.Value
			}
			if token == "" && cfg.AllowBearerFallback { // migration path
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					token = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}
			if token == "" { // unauthenticated
				// allow public endpoints to decide; handlers can enforce
				next.ServeHTTP(w, r)
				return
			}

			// Validate session
			var userID string
			var tenantID sql.NullString
			var expires time.Time
			var active bool
			err := cfg.DB.QueryRow(`SELECT user_id, expires_at, is_active FROM private_markets_sessions WHERE session_token = $1`, token).Scan(&userID, &expires, &active)

			// DEBUG: Log session validation details
			now := time.Now()
			fmt.Printf("[SESSION_AUTH] token=%s... err=%v active=%v expires=%v now=%v isExpired=%v\n",
				token[:min(16, len(token))], err, active, expires, now, now.After(expires))

			if err != nil || !active || now.After(expires) {
				// Expired / invalid: clear cookie and continue
				http.SetCookie(w, &http.Cookie{Name: cfg.SessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
				next.ServeHTTP(w, r)
				return
			}

			// IP Whitelist Check
			if tenantID.Valid {
				var whitelist []string
				rows, err := cfg.DB.Query("SELECT e.ip_address FROM tenant_ip_whitelist_entries e JOIN tenant_ip_whitelist_assignments a ON a.whitelist_id = e.id WHERE a.tenant_id = $1", tenantID.String)
				if err != nil {
					http.Error(w, "Failed to query IP whitelist", http.StatusInternalServerError)
					return
				}
				defer rows.Close()

				for rows.Next() {
					var ipAddress string
					if err := rows.Scan(&ipAddress); err != nil {
						http.Error(w, "Failed to scan IP whitelist", http.StatusInternalServerError)
						return
					}
					whitelist = append(whitelist, ipAddress)
				}

				if len(whitelist) > 0 {
					remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
					if err != nil {
						remoteIP = r.RemoteAddr // Assume it's just the IP if SplitHostPort fails
					}

					if !ip.IsIpAllowed(whitelist, remoteIP) {
						http.Error(w, "IP address not allowed", http.StatusForbidden)
						return
					}
				}
			}

			// Populate / update RequestContext if present
			// Fetch user record and attach to the request context so handlers can access it.
			var user imodels.User
			var permsRaw []byte
			if err := cfg.DB.QueryRow(`SELECT id, email, name, role, organization, permissions, is_core_admin, is_active, tenant_id FROM public.users WHERE id = $1`, userID).Scan(&user.ID, &user.Email, &user.Name, &user.Role, &user.Organization, &permsRaw, &user.IsCoreAdmin, &user.IsActive, &tenantID); err == nil {
				// permissions is JSONB; if present, unmarshal into []string
				if len(permsRaw) > 0 {
					// best-effort unmarshal, ignore errors
					_ = json.Unmarshal(permsRaw, &user.Permissions)
				}

				// Populate TenantID from the session default, but allow header override
				if tenantID.Valid {
					user.TenantID = tenantID.String
				}

				// Allow client to specify tenant context (e.g. for super admins or switching contexts)
				if reqTenantID := jwtmiddleware.GetClaimsFromContext(r).TenantID; reqTenantID != "" {
					// TODO: Add strict permission check here (e.g. only allow if user is super admin or if users can belong to multiple tenants)
					// For now, trusting the header for multi-tenant management flows
					user.TenantID = reqTenantID
				}

				// ========================================
				// SECURITY: Set PostgreSQL session variable for Row-Level Security (RLS)
				// This ensures that all database queries are automatically filtered by tenant_id
				// ========================================
				if user.TenantID != "" {
					_, err := cfg.DB.ExecContext(r.Context(), "SET LOCAL app.current_tenant_id = $1", user.TenantID)
					if err != nil {
						// Log error but don't fail the request
						fmt.Printf("[SESSION_AUTH] Failed to set app.current_tenant_id: %v\n", err)
					}
				}

				// Check if user is a global admin (Uisce organization)
				if user.Organization == "uisce" && user.Role == "admin" {
					// Set flag indicating this is a global admin session
					_, _ = cfg.DB.ExecContext(r.Context(), "SET LOCAL app.is_global_admin = 'true'")
				}

				// Attach to request context using typed auth helper
				ctx := auth.SetUserInContext(r.Context(), user)
				r = r.WithContext(ctx)
			} else {
				// DEBUG: Log the error when user query fails
				fmt.Printf("[SESSION_AUTH] User query failed for userID=%s: %v\n", userID, err)
			}

			// Continue request chain (authenticated or not)
			next.ServeHTTP(w, r)
		})
	}
}

// GetRequestContextFromStd attempts to get request context previously attached by gin/chi wrapper; returns nil if unavailable.
func GetRequestContextFromStd(r *http.Request) *requestcontext.RequestContext {
	return requestcontext.GetRequestContext(r.Context())
}
