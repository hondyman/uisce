package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ============================================================================
// RBAC Permission Enforcement Middleware
// Fortune 500 Enterprise Security Layer
// ============================================================================

type RBACEnforcer struct {
	db *sqlx.DB
}

func NewRBACEnforcer(db *sqlx.DB) *RBACEnforcer {
	return &RBACEnforcer{db: db}
}

// RequirePermission enforces that the user has a specific permission
// Usage: r.With(rbac.RequirePermission("process.read")).Get("/api/processes", handler)
func (re *RBACEnforcer) RequirePermission(permissionKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract user from context (set by auth middleware)
			userID := getUserIDFromContext(r.Context())
			if userID == "" {
				http.Error(w, "Unauthorized: No user context", http.StatusUnauthorized)
				return
			}

			// Extract tenant/datasource from query or headers
			tenantID := getTenantIDFromRequest(r)
			datasourceID := getDatasourceIDFromRequest(r)

			if tenantID == "" || datasourceID == "" {
				http.Error(w, "Bad Request: tenant_id and datasource_id required", http.StatusBadRequest)
				return
			}

			// Check permission using database function
			hasPerm, err := re.checkUserPermission(userID, tenantID, datasourceID, permissionKey)
			if err != nil {
				log.Printf("Error checking permission: %v", err)
				http.Error(w, "Internal Server Error: Permission check failed", http.StatusInternalServerError)
				return
			}

			if !hasPerm {
				log.Printf("User %s denied: missing permission %s", userID, permissionKey)
				http.Error(w, fmt.Sprintf("Forbidden: Permission '%s' required", permissionKey), http.StatusForbidden)
				return
			}

			// User has permission, proceed
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission enforces that the user has ANY of the specified permissions
// Useful for OR logic: user needs process.read OR process.execute
func (re *RBACEnforcer) RequireAnyPermission(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := getUserIDFromContext(r.Context())
			if userID == "" {
				http.Error(w, "Unauthorized: No user context", http.StatusUnauthorized)
				return
			}

			tenantID := getTenantIDFromRequest(r)
			datasourceID := getDatasourceIDFromRequest(r)

			if tenantID == "" || datasourceID == "" {
				http.Error(w, "Bad Request: tenant_id and datasource_id required", http.StatusBadRequest)
				return
			}

			// Check if user has ANY of the permissions
			for _, perm := range permissions {
				hasPerm, err := re.checkUserPermission(userID, tenantID, datasourceID, perm)
				if err != nil {
					log.Printf("Error checking permission %s: %v", perm, err)
					continue
				}
				if hasPerm {
					// User has at least one permission, proceed
					next.ServeHTTP(w, r)
					return
				}
			}

			// User doesn't have any of the required permissions
			log.Printf("User %s denied: missing all permissions %v", userID, permissions)
			http.Error(w, fmt.Sprintf("Forbidden: One of these permissions required: %v", permissions), http.StatusForbidden)
		})
	}
}

// RequireAllPermissions enforces that the user has ALL of the specified permissions
// Useful for AND logic: user needs both process.read AND step.execute
func (re *RBACEnforcer) RequireAllPermissions(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := getUserIDFromContext(r.Context())
			if userID == "" {
				http.Error(w, "Unauthorized: No user context", http.StatusUnauthorized)
				return
			}

			tenantID := getTenantIDFromRequest(r)
			datasourceID := getDatasourceIDFromRequest(r)

			if tenantID == "" || datasourceID == "" {
				http.Error(w, "Bad Request: tenant_id and datasource_id required", http.StatusBadRequest)
				return
			}

			// Check if user has ALL of the permissions
			missingPerms := []string{}
			for _, perm := range permissions {
				hasPerm, err := re.checkUserPermission(userID, tenantID, datasourceID, perm)
				if err != nil {
					log.Printf("Error checking permission %s: %v", perm, err)
					http.Error(w, "Internal Server Error: Permission check failed", http.StatusInternalServerError)
					return
				}
				if !hasPerm {
					missingPerms = append(missingPerms, perm)
				}
			}

			if len(missingPerms) > 0 {
				log.Printf("User %s denied: missing permissions %v", userID, missingPerms)
				http.Error(w, fmt.Sprintf("Forbidden: All of these permissions required: %v", permissions), http.StatusForbidden)
				return
			}

			// User has all permissions, proceed
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole enforces that the user has a specific role
// Usage: r.With(rbac.RequireRole("admin")).Post("/api/admin/settings", handler)
func (re *RBACEnforcer) RequireRole(roleKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := getUserIDFromContext(r.Context())
			if userID == "" {
				http.Error(w, "Unauthorized: No user context", http.StatusUnauthorized)
				return
			}

			tenantID := getTenantIDFromRequest(r)
			datasourceID := getDatasourceIDFromRequest(r)

			if tenantID == "" || datasourceID == "" {
				http.Error(w, "Bad Request: tenant_id and datasource_id required", http.StatusBadRequest)
				return
			}

			// Check if user has the role
			hasRole, err := re.checkUserRole(userID, tenantID, datasourceID, roleKey)
			if err != nil {
				log.Printf("Error checking role: %v", err)
				http.Error(w, "Internal Server Error: Role check failed", http.StatusInternalServerError)
				return
			}

			if !hasRole {
				log.Printf("User %s denied: missing role %s", userID, roleKey)
				http.Error(w, fmt.Sprintf("Forbidden: Role '%s' required", roleKey), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRoleLevel enforces minimum role level (viewer < editor < admin < super_admin)
// Usage: r.With(rbac.RequireRoleLevel("admin")).Delete("/api/processes/:id", handler)
func (re *RBACEnforcer) RequireRoleLevel(minLevel string) func(http.Handler) http.Handler {
	levelOrder := map[string]int{
		"viewer":      1,
		"editor":      2,
		"approver":    3,
		"admin":       4,
		"super_admin": 5,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := getUserIDFromContext(r.Context())
			if userID == "" {
				http.Error(w, "Unauthorized: No user context", http.StatusUnauthorized)
				return
			}

			tenantID := getTenantIDFromRequest(r)
			datasourceID := getDatasourceIDFromRequest(r)

			if tenantID == "" || datasourceID == "" {
				http.Error(w, "Bad Request: tenant_id and datasource_id required", http.StatusBadRequest)
				return
			}

			// Get user's highest role level
			userLevel, err := re.getUserHighestRoleLevel(userID, tenantID, datasourceID)
			if err != nil {
				log.Printf("Error getting user role level: %v", err)
				http.Error(w, "Internal Server Error: Role level check failed", http.StatusInternalServerError)
				return
			}

			minLevelNum := levelOrder[minLevel]
			userLevelNum := levelOrder[userLevel]

			if userLevelNum < minLevelNum {
				log.Printf("User %s denied: role level %s < required %s", userID, userLevel, minLevel)
				http.Error(w, fmt.Sprintf("Forbidden: Minimum role level '%s' required", minLevel), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CheckDelegation checks if a delegation exists for approval actions
// This is typically called within approval handlers, not as middleware
func (re *RBACEnforcer) CheckDelegation(delegatorID, delegateID, tenantID, datasourceID, resourceType, resourceID string) (string, error) {
	var delegationID string
	err := re.db.QueryRow(`
		SELECT bp_get_active_delegation($1, $2, $3, $4, $5, $6)
	`, delegatorID, delegateID, tenantID, datasourceID, resourceType, resourceID).Scan(&delegationID)

	if err != nil {
		return "", err
	}

	return delegationID, nil
}

// ============================================================================
// INTERNAL HELPER FUNCTIONS
// ============================================================================

func (re *RBACEnforcer) checkUserPermission(userID, tenantID, datasourceID, permissionKey string) (bool, error) {
	var hasPerm bool
	err := re.db.QueryRow(`
		SELECT bp_user_has_permission($1, $2, $3, $4)
	`, userID, tenantID, datasourceID, permissionKey).Scan(&hasPerm)

	if err != nil {
		return false, fmt.Errorf("permission check query failed: %w", err)
	}

	return hasPerm, nil
}

func (re *RBACEnforcer) checkUserRole(userID, tenantID, datasourceID, roleKey string) (bool, error) {
	var hasRole bool
	err := re.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM bp_user_roles ur
			JOIN bp_roles r ON ur.role_id = r.id
			WHERE ur.user_id = $1
			  AND ur.tenant_id = $2
			  AND ur.datasource_id = $3
			  AND r.role_key = $4
			  AND ur.is_active = true
			  AND (ur.expires_at IS NULL OR ur.expires_at > CURRENT_TIMESTAMP)
		)
	`, userID, tenantID, datasourceID, roleKey).Scan(&hasRole)

	if err != nil {
		return false, fmt.Errorf("role check query failed: %w", err)
	}

	return hasRole, nil
}

func (re *RBACEnforcer) getUserHighestRoleLevel(userID, tenantID, datasourceID string) (string, error) {
	var roleLevel string
	err := re.db.QueryRow(`
		SELECT r.role_level
		FROM bp_user_roles ur
		JOIN bp_roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1
		  AND ur.tenant_id = $2
		  AND ur.datasource_id = $3
		  AND ur.is_active = true
		  AND (ur.expires_at IS NULL OR ur.expires_at > CURRENT_TIMESTAMP)
		ORDER BY 
		  CASE r.role_level 
		    WHEN 'super_admin' THEN 5
		    WHEN 'admin' THEN 4
		    WHEN 'approver' THEN 3
		    WHEN 'editor' THEN 2
		    WHEN 'viewer' THEN 1
		    ELSE 0
		  END DESC
		LIMIT 1
	`, userID, tenantID, datasourceID).Scan(&roleLevel)

	if err != nil {
		return "", fmt.Errorf("role level query failed: %w", err)
	}

	return roleLevel, nil
}

// Extract user ID from request context (set by auth middleware)
func getUserIDFromContext(ctx context.Context) string {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		// Try alternative key names
		if user, ok := ctx.Value("user").(map[string]interface{}); ok {
			if id, ok := user["id"].(string); ok {
				return id
			}
		}
		return ""
	}
	return userID
}

// Extract tenant ID from query string or header
func getTenantIDFromRequest(r *http.Request) string {
	// Try query parameter first
	if tenantID := r.URL.Query().Get("tenant_id"); tenantID != "" {
		return tenantID
	}
	// Try header
	return jwtmiddleware.GetClaimsFromContext(r).TenantID
}

// Extract datasource ID from query string or header
func getDatasourceIDFromRequest(r *http.Request) string {
	// Try query parameter first
	if datasourceID := r.URL.Query().Get("datasource_id"); datasourceID != "" {
		return datasourceID
	}
	// Try header
	return r.Header.Get("X-Tenant-Datasource-ID")
}

// ============================================================================
// FIELD-LEVEL MASKING HELPER (for use in handlers)
// ============================================================================

type FieldMaskingRule struct {
	FieldName      string
	MaskingType    string
	MaskingPattern string
}

// GetFieldMaskingRules retrieves masking rules for a user and resource
func (re *RBACEnforcer) GetFieldMaskingRules(userID, tenantID, datasourceID, resourceType string) ([]FieldMaskingRule, error) {
	rows, err := re.db.Query(`
		SELECT DISTINCT fmr.field_name, fmr.masking_type, fmr.masking_pattern
		FROM bp_user_roles ur
		JOIN bp_roles r ON ur.role_id = r.id
		JOIN bp_field_masking_rules fmr ON fmr.tenant_id = ur.tenant_id 
		                               AND fmr.datasource_id = ur.datasource_id
		                               AND fmr.resource_type = $4
		WHERE ur.user_id = $1
		  AND ur.tenant_id = $2
		  AND ur.datasource_id = $3
		  AND ur.is_active = true
		  AND NOT (r.id = ANY(fmr.unmasked_roles))
		  AND (ur.expires_at IS NULL OR ur.expires_at > CURRENT_TIMESTAMP)
	`, userID, tenantID, datasourceID, resourceType)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []FieldMaskingRule
	for rows.Next() {
		var rule FieldMaskingRule
		if err := rows.Scan(&rule.FieldName, &rule.MaskingType, &rule.MaskingPattern); err != nil {
			continue
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

// ApplyFieldMasking applies masking rules to data (simple implementation)
func ApplyFieldMasking(data map[string]interface{}, rules []FieldMaskingRule) {
	for _, rule := range rules {
		if val, exists := data[rule.FieldName]; exists {
			if strVal, ok := val.(string); ok {
				data[rule.FieldName] = maskValue(strVal, rule.MaskingPattern)
			}
		}
	}
}

func maskValue(value, pattern string) string {
	if pattern == "" {
		return "***MASKED***"
	}

	// Simple pattern matching
	// pattern examples: "XXX-XX-####" (SSN), "XXXX-####" (bank account)
	// X = masked, # = visible from end

	visibleCount := 0
	for _, char := range pattern {
		if char == '#' {
			visibleCount++
		}
	}

	if visibleCount == 0 {
		return pattern
	}

	if len(value) < visibleCount {
		return value
	}

	visible := value[len(value)-visibleCount:]
	masked := ""
	for i := 0; i < len(pattern)-visibleCount; i++ {
		if pattern[i] == 'X' {
			masked += "X"
		} else {
			masked += string(pattern[i])
		}
	}

	return masked + visible
}
