package audit

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hondyman/semlayer/backend/internal/auth"
)

// Role definitions and permissions
const (
	RoleGlobalAdmin = "global_admin"
	RoleGlobalOps   = "global_ops"
	RoleTenantAdmin = "tenant_admin"
	RoleTenantOps   = "tenant_ops"
)

// RolePermissions defines what each role can do
var RolePermissions = map[string]RolePermission{
	RoleGlobalAdmin: {
		CanViewAllTenants:      true,
		CanApproveChangeSets:   true,
		CanViewGovernance:      true,
		CanViewSemanticCatalog: true,
		CanViewCompliance:      true,
		CanAIReasonCrossTenant: true,
		CanViewIncidents:       true,
		DashboardType:          "global_admin",
	},
	RoleGlobalOps: {
		CanViewAllTenants:      false, // Limited to assigned tenants
		CanApproveChangeSets:   true,  // Medium risk only
		CanViewGovernance:      true,
		CanViewSemanticCatalog: true,
		CanViewCompliance:      true,
		CanAIReasonCrossTenant: true, // Within assigned tenants
		CanViewIncidents:       true,
		DashboardType:          "global_ops",
	},
	RoleTenantAdmin: {
		CanViewAllTenants:      false,
		CanApproveChangeSets:   true, // Tenant only
		CanViewGovernance:      true,
		CanViewSemanticCatalog: true,
		CanViewCompliance:      true,
		CanAIReasonCrossTenant: false,
		CanViewIncidents:       true,
		DashboardType:          "tenant_admin",
	},
	RoleTenantOps: {
		CanViewAllTenants:      false,
		CanApproveChangeSets:   false,
		CanViewGovernance:      false,
		CanViewSemanticCatalog: false,
		CanViewCompliance:      false,
		CanAIReasonCrossTenant: false,
		CanViewIncidents:       true,
		DashboardType:          "tenant_ops",
	},
}

// RolePermission defines permissions for a role
type RolePermission struct {
	CanViewAllTenants      bool
	CanApproveChangeSets   bool
	CanViewGovernance      bool
	CanViewSemanticCatalog bool
	CanViewCompliance      bool
	CanAIReasonCrossTenant bool
	CanViewIncidents       bool
	DashboardType          string
}

// TenantScopeMiddlewareChi enforces multi-tenant isolation (Chi version)
func TenantScopeMiddlewareChi(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user's allowed tenants from auth context
		allowedTenants := auth.AllowedTenantsFromContext(r.Context())

		if len(allowedTenants) == 0 {
			http.Error(w, "no accessible tenants", http.StatusForbidden)
			return
		}

		// Add to request context for use in handlers
		ctx := context.WithValue(r.Context(), "allowedTenants", allowedTenants)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RoleBasedAccessMiddleware enforces role-specific permissions
func RoleBasedAccessMiddleware(requiredRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRoles := auth.RolesFromContext(r.Context())

			// Check if user has at least one of the required roles
			hasRole := false
			for _, required := range requiredRoles {
				for _, userRole := range userRoles {
					if userRole == required {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}

			if !hasRole {
				http.Error(w,
					fmt.Sprintf("this action requires one of: %v", requiredRoles),
					http.StatusForbidden,
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetAllowedTenants extracts tenant scope from request context
func GetAllowedTenants(r *http.Request) TenantScope {
	tenants := r.Context().Value("allowedTenants")
	if ts, ok := tenants.(TenantScope); ok {
		return ts
	}
	// Fallback to auth context
	return auth.AllowedTenantsFromContext(r.Context())
}

// EnforceTenantScope ensures requested tenants are in user's scope
func EnforceTenantScope(r *http.Request, requestedTenants []string) (TenantScope, error) {
	allowed := GetAllowedTenants(r)

	if len(requestedTenants) == 0 {
		return allowed, nil
	}

	// Intersect requested tenants with allowed tenants
	result := make(TenantScope, 0)
	for _, requested := range requestedTenants {
		if allowed.Contains(requested) {
			result = append(result, requested)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("none of the requested tenants are accessible")
	}

	return result, nil
}

// ValidateRolePermission checks if a user's role allows a specific action
func ValidateRolePermission(r *http.Request, permission string) bool {
	userRoles := auth.RolesFromContext(r.Context())

	for _, roleName := range userRoles {
		perm, exists := RolePermissions[roleName]
		if !exists {
			continue
		}

		switch permission {
		case "view_all_tenants":
			if perm.CanViewAllTenants {
				return true
			}
		case "approve_changesets":
			if perm.CanApproveChangeSets {
				return true
			}
		case "view_governance":
			if perm.CanViewGovernance {
				return true
			}
		case "view_semantic_catalog":
			if perm.CanViewSemanticCatalog {
				return true
			}
		case "view_compliance":
			if perm.CanViewCompliance {
				return true
			}
		case "ai_reason_cross_tenant":
			if perm.CanAIReasonCrossTenant {
				return true
			}
		case "view_incidents":
			if perm.CanViewIncidents {
				return true
			}
		}
	}

	return false
}

// GetRoleType returns the primary role type for a user
func GetRoleType(r *http.Request) string {
	userRoles := auth.RolesFromContext(r.Context())

	// Priority order: global_admin > global_ops > tenant_admin > tenant_ops
	roleOrder := []string{RoleGlobalAdmin, RoleGlobalOps, RoleTenantAdmin, RoleTenantOps}

	for _, role := range roleOrder {
		for _, userRole := range userRoles {
			if userRole == role {
				return role
			}
		}
	}

	return "" // Unknown role
}

// GetDashboardType returns the appropriate dashboard type for user's role
func GetDashboardType(r *http.Request) string {
	roleType := GetRoleType(r)
	if perm, exists := RolePermissions[roleType]; exists {
		return perm.DashboardType
	}
	return "unknown"
}

// EnforceSingleTenant ensures user has single tenant access (for tenant roles)
func EnforceSingleTenant(r *http.Request) (string, error) {
	allowed := GetAllowedTenants(r)

	if len(allowed) != 1 {
		return "", fmt.Errorf("this action requires single-tenant access")
	}

	return allowed[0], nil
}

// BuildAIPromptWithTenantScope creates an AI prompt that enforces tenant scoping
func BuildAIPromptWithTenantScope(basePrompt string, tenantScope TenantScope, userRole string) string {
	perm := RolePermissions[userRole]

	scope := ""
	if perm.CanAIReasonCrossTenant {
		scope = fmt.Sprintf("You can reason across the following tenants: %v", tenantScope)
	} else {
		if len(tenantScope) == 1 {
			scope = fmt.Sprintf("You are scoped to tenant: %s", tenantScope[0])
		} else {
			scope = fmt.Sprintf("You are scoped to tenants: %v", tenantScope)
		}
		scope += "\nYou CANNOT reference data from other tenants."
	}

	return fmt.Sprintf("%s\n\n%s", scope, basePrompt)
}

// AuditExplorerACL defines granular access control
type AuditExplorerACL struct {
	Role              string
	TenantScope       TenantScope
	CanViewTimeline   bool
	CanViewEntities   bool
	CanViewIncidents  bool
	CanViewCompliance bool
	CanExplainWithAI  bool
	TimelineFilters   []string // which artifact types visible
}

// ComputeACL computes the ACL for a user based on their role and scope
func ComputeACL(r *http.Request) *AuditExplorerACL {
	roleType := GetRoleType(r)
	tenantScope := GetAllowedTenants(r)
	_ = RolePermissions[roleType] // Verified role exists

	acl := &AuditExplorerACL{
		Role:        roleType,
		TenantScope: tenantScope,
	}

	switch roleType {
	case RoleGlobalAdmin:
		acl.CanViewTimeline = true
		acl.CanViewEntities = true
		acl.CanViewIncidents = true
		acl.CanViewCompliance = true
		acl.CanExplainWithAI = true
		acl.TimelineFilters = []string{"*"} // All types

	case RoleGlobalOps:
		acl.CanViewTimeline = true
		acl.CanViewEntities = true
		acl.CanViewIncidents = true
		acl.CanViewCompliance = true
		acl.CanExplainWithAI = true
		acl.TimelineFilters = []string{"job_run", "dag_run", "incident", "compliance_violation"}

	case RoleTenantAdmin:
		acl.CanViewTimeline = true
		acl.CanViewEntities = true
		acl.CanViewIncidents = true
		acl.CanViewCompliance = true
		acl.CanExplainWithAI = true
		acl.TimelineFilters = []string{"*"} // All types

	case RoleTenantOps:
		acl.CanViewTimeline = true
		acl.CanViewEntities = false
		acl.CanViewIncidents = true
		acl.CanViewCompliance = false
		acl.CanExplainWithAI = true
		acl.TimelineFilters = []string{"job_run", "dag_run", "incident", "workflow_event"}
	}

	return acl
}
