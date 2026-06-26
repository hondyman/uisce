package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// Context key constants for JWT claims
const (
	ContextKeyUserID   = "user_id"
	ContextKeyTenantID = "tenant_id"
	ContextKeyTenants  = "tenant_ids"
	ContextKeyRoles    = "roles"
	ContextKeyEmail    = "email"
	ContextKeyJTI      = "jti"
)

// JWTClaims represents the standard JWT claims used across the platform
type JWTClaims struct {
	UserID       string                 `json:"user_id"`
	Email        string                 `json:"email"`
	Name         string                 `json:"name"`
	Role         string                 `json:"role"`
	Roles        []string               `json:"roles"`
	Organization string                 `json:"organization"`
	TenantID     string                 `json:"tenant_id"`
	TenantIDs    []string               `json:"tenant_ids"`
	Permissions  []string               `json:"permissions"`
	IsCoreAdmin  bool                   `json:"is_core_admin"`
	JTI          string                 `json:"jti"` // JWT ID for revocation
	HasuraClaims map[string]interface{} `json:"https://hasura.io/jwt/claims"`
	jwt.RegisteredClaims
}

// JWTMiddleware validates JWT tokens and extracts claims into context
// Supports Bearer token format: "Authorization: Bearer <token>"
func JWTMiddleware(jwtSecret string, logger *logrus.Entry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// In development, allow requests with X-User-ID header for testing
				if strings.ToLower(os.Getenv("DEV_ALLOW_UNAUTH_XUSER")) == "true" && r.Header.Get("X-User-ID") != "" {
					ctx := context.WithValue(r.Context(), ContextKeyUserID, r.Header.Get("X-User-ID"))
					ctx = context.WithValue(ctx, ContextKeyTenantID, jwtmiddleware.GetClaimsFromContext(r).TenantID)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}

				logger.Warn("Missing Authorization header")
				writeErrorResponse(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Extract Bearer token
			tokenString := extractBearerToken(authHeader)
			if tokenString == "" {
				logger.Warn("Invalid authorization format")
				writeErrorResponse(w, "Invalid authorization format (expected Bearer token)", http.StatusUnauthorized)
				return
			}

			// Parse and validate JWT token
			claims := &JWTClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				// Validate signing method is HMAC
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})

			if err != nil {
				logger.WithError(err).Warn("JWT parsing failed")
				writeErrorResponse(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			if !token.Valid {
				logger.Warn("Invalid token")
				writeErrorResponse(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Validate required claims
			if claims.UserID == "" {
				logger.Warn("Token missing user_id claim")
				writeErrorResponse(w, "Token missing required claims", http.StatusUnauthorized)
				return
			}

			// Validate tenant access - require at least one tenant
			if claims.TenantID == "" && len(claims.TenantIDs) == 0 {
				logger.Warn("Token missing tenant_id or tenant_ids claim")
				writeErrorResponse(w, "Token missing tenant claims", http.StatusUnauthorized)
				return
			}

			// Add claims to context for downstream handlers
			ctx := r.Context()
			ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextKeyEmail, claims.Email)
			ctx = context.WithValue(ctx, ContextKeyTenantID, claims.TenantID)
			ctx = context.WithValue(ctx, ContextKeyTenants, claims.TenantIDs)
			ctx = context.WithValue(ctx, ContextKeyRoles, claims.Roles)
			ctx = context.WithValue(ctx, ContextKeyJTI, claims.JTI)

			logger.WithFields(logrus.Fields{
				"user_id":   claims.UserID,
				"tenant_id": claims.TenantID,
				"role":      claims.Role,
				"path":      r.RequestURI,
			}).Debug("JWT validated successfully")

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TenantGuardMiddleware ensures user has access to the requested tenant
// Extracts tenant from X-Tenant-ID header and validates against JWT claims
func TenantGuardMiddleware(logger *logrus.Entry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Get requested tenant from header
			requestedTenant := jwtmiddleware.GetClaimsFromContext(r).TenantID
			if requestedTenant == "" {
				// Fall back to tenant from JWT claims
				if tenant, ok := ctx.Value(ContextKeyTenantID).(string); ok && tenant != "" {
					requestedTenant = tenant
				} else {
					logger.Warn("Missing tenant identification")
					writeErrorResponse(w, "X-Tenant-ID header required or token missing tenant", http.StatusForbidden)
					return
				}
			}

			// Get authenticated tenant from JWT
			jwtTenant := ""
			if tenant, ok := ctx.Value(ContextKeyTenantID).(string); ok {
				jwtTenant = tenant
			}

			jwtTenants := []string{}
			if tenants, ok := ctx.Value(ContextKeyTenants).([]string); ok {
				jwtTenants = tenants
			}

			// Validate user has access to requested tenant
			hasAccess := jwtTenant == requestedTenant
			if !hasAccess && len(jwtTenants) > 0 {
				for _, t := range jwtTenants {
					if t == requestedTenant {
						hasAccess = true
						break
					}
				}
			}

			if !hasAccess {
				logger.WithFields(logrus.Fields{
					"requested_tenant": requestedTenant,
					"jwt_tenant":       jwtTenant,
					"jwt_tenants":      jwtTenants,
				}).Warn("Tenant access denied")
				writeErrorResponse(w, "Access denied for requested tenant", http.StatusForbidden)
				return
			}

			// Add requested tenant to context for downstream handlers
			ctx = context.WithValue(ctx, ContextKeyTenantID, requestedTenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ExtractUserIDFromContext retrieves user_id from request context (lenient version)
func ExtractUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(ContextKeyUserID).(string); ok {
		return userID
	}
	return ""
}

// ExtractUserIDFromContextStrict retrieves user_id from request context with error handling
func ExtractUserIDFromContextStrict(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(ContextKeyUserID).(string)
	if !ok || userID == "" {
		return "", errors.New("user_id not found in context")
	}
	return userID, nil
}

// ExtractTenantIDFromContext retrieves tenant_id from request context (lenient version)
func ExtractTenantIDFromContext(ctx context.Context) string {
	if tenantID, ok := ctx.Value(ContextKeyTenantID).(string); ok {
		return tenantID
	}
	return ""
}

// ExtractTenantIDFromContextStrict retrieves tenant_id from request context with error handling
func ExtractTenantIDFromContextStrict(ctx context.Context) (string, error) {
	tenantID, ok := ctx.Value(ContextKeyTenantID).(string)
	if !ok || tenantID == "" {
		return "", errors.New("tenant_id not found in context")
	}
	return tenantID, nil
}

// ExtractEmailFromContext retrieves user email from request context
func ExtractEmailFromContext(ctx context.Context) string {
	if email, ok := ctx.Value(ContextKeyEmail).(string); ok {
		return email
	}
	return ""
}

// ExtractJTIFromContext retrieves JWT ID (jti) from request context for token revocation
func ExtractJTIFromContext(ctx context.Context) string {
	if jti, ok := ctx.Value(ContextKeyJTI).(string); ok {
		return jti
	}
	return ""
}

// ExtractTenantsFromContext retrieves all tenant_ids from request context
func ExtractTenantsFromContext(ctx context.Context) []string {
	if tenants, ok := ctx.Value(ContextKeyTenants).([]string); ok {
		return tenants
	}
	return []string{}
}

// ExtractRolesFromContext retrieves user roles from request context
func ExtractRolesFromContext(ctx context.Context) []string {
	if roles, ok := ctx.Value(ContextKeyRoles).([]string); ok {
		return roles
	}
	return []string{}
}

// HasRole checks if user has a specific role
func HasRole(ctx context.Context, requiredRole string) bool {
	roles := ExtractRolesFromContext(ctx)
	for _, role := range roles {
		if role == requiredRole {
			return true
		}
	}
	return false
}

// extractBearerToken extracts the JWT token from Authorization header
func extractBearerToken(authHeader string) string {
	const bearerPrefix = "Bearer "
	if strings.HasPrefix(authHeader, bearerPrefix) {
		return strings.TrimSpace(authHeader[len(bearerPrefix):])
	}
	return ""
}

// WithClaims adds claims to a context (useful for testing)
func WithClaims(ctx context.Context, claims map[string]interface{}) context.Context {
	if userID, ok := claims["user_id"].(string); ok {
		ctx = context.WithValue(ctx, ContextKeyUserID, userID)
	}
	if tenantID, ok := claims["tenant_id"].(string); ok {
		ctx = context.WithValue(ctx, ContextKeyTenantID, tenantID)
	}
	if roles, ok := claims["roles"].([]string); ok {
		ctx = context.WithValue(ctx, ContextKeyRoles, roles)
	}
	if email, ok := claims["email"].(string); ok {
		ctx = context.WithValue(ctx, ContextKeyEmail, email)
	}
	if tenants, ok := claims["tenant_ids"].([]string); ok {
		ctx = context.WithValue(ctx, ContextKeyTenants, tenants)
	}
	return ctx
}

// writeErrorResponse writes a JSON error response
func writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, `{"error": "%s"}`, message)
}
