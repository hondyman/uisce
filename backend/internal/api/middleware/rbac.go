package middleware

import (
	"net/http"
)

type Role string

const (
	RoleAdvisor    Role = "advisor"
	RoleCompliance Role = "compliance"
	RoleAdmin      Role = "admin"
)

// Mock user context - in reality this would come from JWT/Session
type UserContext struct {
	UserID string
	Role   Role
}

// RequireRole creates a middleware that enforces role access
func RequireRole(allowedRoles ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Extract User (Mocked for now)
			// In prod: user := r.Context().Value("user").(*UserContext)
			user := &UserContext{UserID: "mock-user", Role: RoleAdvisor} // Default to Advisor

			// Override for testing via header
			if r.Header.Get("X-Mock-Role") != "" {
				user.Role = Role(r.Header.Get("X-Mock-Role"))
			}

			// 2. Check Permissions
			allowed := false
			for _, role := range allowedRoles {
				if user.Role == role || user.Role == RoleAdmin {
					allowed = true
					break
				}
			}

			if !allowed {
				http.Error(w, "Forbidden: Insufficient Permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Helper to check if user has permission for specific resource action
// (Not used in middleware directly but available for handlers)
func CheckPermission(user *UserContext, resource string, action string) bool {
	// Simple map-based logic as per blueprint
	if user.Role == RoleAdmin {
		return true
	}
	if user.Role == RoleCompliance && (resource == "audit_logs" || resource == "reports") {
		return true
	}
	if user.Role == RoleAdvisor && (resource == "ai_session" || resource == "client_data") {
		return true
	}
	return false
}
