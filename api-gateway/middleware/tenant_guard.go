package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// TenantClaims represents the tenant-related JWT claims
type TenantClaims struct {
	Sub         string   `json:"sub"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	Roles       []string `json:"roles"`
	Scopes      []string `json:"scopes"`
	TenantScope string   `json:"tenant_scope"` // "single" | "multi" | "all"
	TenantID    string   `json:"tenant_id,omitempty"`
	TenantIDs   []string `json:"tenant_ids,omitempty"`
	OrgID       string   `json:"org_id,omitempty"`
	JTI         string   `json:"jti"`
	jwt.RegisteredClaims
}

type contextKey string

const (
	TenantContextKey contextKey = "tenant_context"
	ClaimsContextKey contextKey = "jwt_claims"
)

// TenantContext holds validated tenant information for the request
type TenantContext struct {
	UserID      string
	Email       string
	TenantScope string
	TenantID    string   // Validated and approved tenant ID
	TenantIDs   []string // Available tenant IDs for multi-tenant users
	OrgID       string
	Roles       []string
	Scopes      []string
}

// TenantGuard middleware enforces tenant context validation
// This is the core multi-tenant enforcement layer
func TenantGuard(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract claims from context (set by JWT middleware)
			claims, ok := r.Context().Value(ClaimsContextKey).(*TenantClaims)
			if !ok || claims == nil {
				http.Error(w, `{"error":"Unauthorized: missing or invalid JWT claims"}`, http.StatusUnauthorized)
				return
			}

			// Get requested tenant from header (client sets this)
			requestedTenant := r.Header.Get("X-Tenant-ID")

			// Validate tenant access based on tenant_scope
			validatedTenant, err := validateTenantAccess(claims, requestedTenant, r.URL.Path)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"Forbidden: %s"}`, err.Error()), http.StatusForbidden)
				return
			}

			// Build tenant context
			tenantCtx := &TenantContext{
				UserID:      claims.Sub,
				Email:       claims.Email,
				TenantScope: claims.TenantScope,
				TenantID:    validatedTenant,
				TenantIDs:   claims.TenantIDs,
				OrgID:       claims.OrgID,
				Roles:       claims.Roles,
				Scopes:      claims.Scopes,
			}

			// Inject validated headers for downstream services
			r.Header.Set("X-User-Id", claims.Sub)
			r.Header.Set("X-User-Email", claims.Email)
			r.Header.Set("X-Roles", strings.Join(claims.Roles, ","))
			r.Header.Set("X-Scopes", strings.Join(claims.Scopes, ","))
			r.Header.Set("X-Tenant-Scope", claims.TenantScope)
			r.Header.Set("X-Tenant-ID", validatedTenant) // Always set validated tenant
			if claims.OrgID != "" {
				r.Header.Set("X-Org-Id", claims.OrgID)
			}

			// Add tenant context to request context
			ctx := context.WithValue(r.Context(), TenantContextKey, tenantCtx)

			// Log tenant access for audit
			logTenantAccess(claims.Sub, claims.Email, validatedTenant, r.URL.Path, r.Method)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// validateTenantAccess enforces the tenant access rules
func validateTenantAccess(claims *TenantClaims, requestedTenant string, path string) (string, error) {
	switch claims.TenantScope {
	case "single":
		// Single-tenant users are hard-bound to their tenant
		if requestedTenant != "" && requestedTenant != claims.TenantID {
			return "", fmt.Errorf("cannot access other tenants (bound to: %s)", claims.TenantID)
		}
		// Auto-inject their tenant
		return claims.TenantID, nil

	case "multi":
		// Multi-tenant ops must explicitly set X-Tenant-ID
		if requestedTenant == "" {
			return "", fmt.Errorf("must specify X-Tenant-ID for multi-tenant access")
		}
		// Validate requested tenant is in their allowed list
		if !contains(claims.TenantIDs, requestedTenant) {
			return "", fmt.Errorf("not authorized for tenant: %s", requestedTenant)
		}
		return requestedTenant, nil

	case "all":
		// Global ops can access any tenant, but must be explicit for tenant-scoped APIs
		if isTenantScopedEndpoint(path) {
			if requestedTenant == "" {
				return "", fmt.Errorf("must specify X-Tenant-ID for tenant-scoped endpoints")
			}
		}
		// Allow empty tenant for platform-level endpoints
		return requestedTenant, nil

	default:
		return "", fmt.Errorf("invalid tenant_scope: %s", claims.TenantScope)
	}
}

// isTenantScopedEndpoint determines if an endpoint requires tenant context
func isTenantScopedEndpoint(path string) bool {
	// Platform-level endpoints that don't require tenant context
	platformPaths := []string{
		"/api/platform/",
		"/api/auth/",
		"/health",
		"/metrics",
	}

	for _, prefix := range platformPaths {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}

	// Everything else is tenant-scoped
	return true
}

// contains checks if a string exists in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// logTenantAccess logs tenant access for audit trail
func logTenantAccess(userID, email, tenantID, path, method string) {
	// In production, send to audit log system
	fmt.Printf("[TENANT_ACCESS] user=%s email=%s tenant=%s method=%s path=%s\n",
		userID, email, tenantID, method, path)
}

// GetTenantContext retrieves the tenant context from the request
func GetTenantContext(r *http.Request) (*TenantContext, bool) {
	ctx, ok := r.Context().Value(TenantContextKey).(*TenantContext)
	return ctx, ok
}

// ParseJWT parses and validates a JWT token, returning TenantClaims
func ParseJWT(tokenString string, jwtSecret string) (*TenantClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TenantClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*TenantClaims); ok && token.Valid {
		// Validate required fields
		if claims.Sub == "" || claims.TenantScope == "" {
			return nil, fmt.Errorf("missing required claims: sub or tenant_scope")
		}
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// JWTMiddleware extracts and validates JWT, adds claims to context
func JWTMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"Unauthorized: missing Authorization header"}`, http.StatusUnauthorized)
				return
			}

			// Parse "Bearer <token>"
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, `{"error":"Unauthorized: invalid Authorization header format"}`, http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Parse and validate JWT
			claims, err := ParseJWT(tokenString, jwtSecret)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"Unauthorized: %s"}`, err.Error()), http.StatusUnauthorized)
				return
			}

			// Add claims to context for downstream middleware
			ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireScope checks if the user has a specific scope
func RequireScope(requiredScope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(ClaimsContextKey).(*TenantClaims)
			if !ok || claims == nil {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}

			if !contains(claims.Scopes, requiredScope) {
				http.Error(w, fmt.Sprintf(`{"error":"Forbidden: requires scope '%s'"}`, requiredScope), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole checks if the user has a specific role
func RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(ClaimsContextKey).(*TenantClaims)
			if !ok || claims == nil {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}

			if !contains(claims.Roles, requiredRole) {
				http.Error(w, fmt.Sprintf(`{"error":"Forbidden: requires role '%s'"}`, requiredRole), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Example usage:
// router.Use(JWTMiddleware(jwtSecret))
// router.Use(TenantGuard(jwtSecret))
// router.Handle("/api/data", RequireScope("read:data")(handler))
