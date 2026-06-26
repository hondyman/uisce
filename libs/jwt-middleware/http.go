package jwtmiddleware

import (
	"context"
	"log"
	"net/http"
	"strings"
)

// ContextKey is used for storing JWT claims in context
type ContextKey string

const (
	// ClaimsContextKey stores JWT claims in request context
	ClaimsContextKey ContextKey = "jwt_claims"
	// UserIDContextKey stores user ID in request context
	UserIDContextKey ContextKey = "user_id"
	// TenantIDContextKey stores tenant ID in request context
	TenantIDContextKey ContextKey = "tenant_id"
)

// GetClaimsFromContext retrieves JWT claims from request context
func GetClaimsFromContext(r *http.Request) *JWTClaims {
	claims, ok := r.Context().Value(ClaimsContextKey).(*JWTClaims)
	if !ok {
		return nil
	}
	return claims
}

// GetUserIDFromContext retrieves user ID from request context
func GetUserIDFromContext(r *http.Request) string {
	userID, ok := r.Context().Value(UserIDContextKey).(string)
	if !ok {
		return ""
	}
	return userID
}

// GetTenantIDFromContext retrieves tenant ID from request context
func GetTenantIDFromContext(r *http.Request) string {
	tenantID, ok := r.Context().Value(TenantIDContextKey).(string)
	if !ok {
		return ""
	}
	return tenantID
}

// JWTMiddleware is an HTTP middleware that validates JWT tokens
// It can be used with standard net/http or chi/gin routers
type JWTMiddleware struct {
	// SkipPaths is a list of paths that don't require authentication
	SkipPaths map[string]bool
	// Logger is optional, used for logging authentication events
	Logger *log.Logger
}

// NewJWTMiddleware creates a new JWT middleware with optional skip paths
func NewJWTMiddleware(skipPaths ...string) *JWTMiddleware {
	m := &JWTMiddleware{
		SkipPaths: make(map[string]bool),
	}

	for _, path := range skipPaths {
		m.SkipPaths[path] = true
	}

	return m
}

// SetLogger sets the logger for the middleware
func (m *JWTMiddleware) SetLogger(logger *log.Logger) {
	m.Logger = logger
}

// Handler wraps an HTTP handler to validate JWT tokens
func (m *JWTMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for configured paths
		if m.SkipPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		// Validate JWT token
		claims, err := ValidateTokenFromRequest(r)
		if err != nil {
			if m.Logger != nil {
				m.Logger.Printf("JWT validation failed: %v", err)
			}
			http.Error(w, `{"error": "unauthorized", "message": "`+err.Error()+`"}`, http.StatusUnauthorized)
			return
		}

		// Store claims in request context
		ctx := r.Context()
		ctx = context.WithValue(ctx, ClaimsContextKey, claims)
		ctx = context.WithValue(ctx, UserIDContextKey, claims.UserID)
		ctx = context.WithValue(ctx, TenantIDContextKey, claims.TenantID)

		// Add X-Tenant-ID header if present in claims (for downstream services)
		if claims.TenantID != "" {
			r.Header.Set("X-Tenant-ID", claims.TenantID)
		}

		if m.Logger != nil {
			m.Logger.Printf("JWT validated for tenant=%s user=%s", claims.TenantID, claims.UserID)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireTenant creates a handler that validates tenant access
func RequireTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaimsFromContext(r)
		if claims == nil {
			http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
			return
		}

		// Get tenant ID from header or query parameter
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			tenantID = r.URL.Query().Get("tenant_id")
		}

		if tenantID == "" {
			http.Error(w, `{"error": "missing tenant_id"}`, http.StatusBadRequest)
			return
		}

		// Validate tenant access
		if err := ValidateTenantAccess(claims, tenantID); err != nil {
			http.Error(w, `{"error": "forbidden", "message": "`+err.Error()+`"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireRole creates a handler that requires a specific role
func RequireRole(role string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaimsFromContext(r)
		if claims == nil {
			http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
			return
		}

		if !HasRole(claims, role) {
			http.Error(w, `{"error": "forbidden", "message": "insufficient permissions"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireRoles creates a handler that requires any of the specified roles
func RequireRoles(roles []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaimsFromContext(r)
		if claims == nil {
			http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
			return
		}

		hasRole := false
		for _, role := range roles {
			if HasRole(claims, role) {
				hasRole = true
				break
			}
		}

		if !hasRole {
			http.Error(w, `{"error": "forbidden", "message": "insufficient permissions"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ChiMiddleware creates a Chi-compatible middleware
// Usage: router.Use(jwtmiddleware.ChiMiddleware())
func ChiMiddleware() func(next http.Handler) http.Handler {
	m := NewJWTMiddleware(
		"/health",
		"/api/auth/login",
		"/api/auth/refresh",
		"/docs",
		"/docs/*",
	)

	return m.Handler
}

// OptionalJWTMiddleware is middleware that validates JWT if present, but doesn't require it
type OptionalJWTMiddleware struct {
	Logger *log.Logger
}

// Handler wraps an HTTP handler to optionally validate JWT tokens
func (m *OptionalJWTMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to validate JWT token if present
		if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			claims, err := ValidateTokenFromRequest(r)
			if err == nil {
				// Store claims in request context
				ctx := r.Context()
				ctx = context.WithValue(ctx, ClaimsContextKey, claims)
				ctx = context.WithValue(ctx, UserIDContextKey, claims.UserID)
				ctx = context.WithValue(ctx, TenantIDContextKey, claims.TenantID)

				if m.Logger != nil {
					m.Logger.Printf("Optional JWT validated for tenant=%s user=%s", claims.TenantID, claims.UserID)
				}

				r = r.WithContext(ctx)
			}
		}

		next.ServeHTTP(w, r)
	})
}

// NewOptionalJWTMiddleware creates optional JWT middleware
func NewOptionalJWTMiddleware() *OptionalJWTMiddleware {
	return &OptionalJWTMiddleware{}
}
