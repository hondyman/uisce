package aso

import (
	"context"

	"github.com/google/uuid"
)

// ============================================================================
// ASO Roles
// ============================================================================

// ASORole defines ASO-specific roles
type ASORole string

const (
	RoleGoldCopyAdmin ASORole = "goldcopy_admin"
	RoleGlobalOps     ASORole = "global_ops"
	RoleTenantAdmin   ASORole = "tenant_admin"
	RoleTenantOps     ASORole = "tenant_ops"
)

// AuthContext represents the authenticated user's context
type AuthContext struct {
	UserID     string
	Email      string
	Roles      []string
	TenantID   *uuid.UUID  // Primary tenant if tenant-scoped
	TenantIDs  []uuid.UUID // All accessible tenants
	IsGoldCopy bool
}

// ============================================================================
// Permission Helpers
// ============================================================================

// HasRole checks if the auth context has a specific role
func HasRole(ac *AuthContext, role ASORole) bool {
	if ac == nil {
		return false
	}
	for _, r := range ac.Roles {
		if r == string(role) {
			return true
		}
	}
	return false
}

// CanViewCore checks if user can view core (gold copy) optimizations
func CanViewCore(ac *AuthContext) bool {
	if ac == nil {
		return false
	}
	return HasRole(ac, RoleGoldCopyAdmin) || HasRole(ac, RoleGlobalOps) || ac.IsGoldCopy
}

// CanViewTenant checks if user can view a specific tenant's optimizations
func CanViewTenant(ac *AuthContext, tenantID uuid.UUID) bool {
	if ac == nil {
		return false
	}

	// GoldCopy Admin and GlobalOps can view all tenants
	if HasRole(ac, RoleGoldCopyAdmin) || HasRole(ac, RoleGlobalOps) {
		return true
	}

	// Check if tenant is in accessible list
	for _, tid := range ac.TenantIDs {
		if tid == tenantID {
			return true
		}
	}

	// Check primary tenant
	if ac.TenantID != nil && *ac.TenantID == tenantID {
		return true
	}

	return false
}

// CanViewOptimization checks if user can view an optimization
func CanViewOptimization(ac *AuthContext, opt *ASOOptimization) bool {
	if ac == nil || opt == nil {
		return false
	}

	if opt.Scope == ASOScopeCore {
		return CanViewCore(ac)
	}

	if opt.TenantID != nil {
		return CanViewTenant(ac, *opt.TenantID)
	}

	return false
}

// CanApproveOptimization checks if user can approve an optimization
func CanApproveOptimization(ac *AuthContext, opt *ASOOptimization) bool {
	if ac == nil || opt == nil {
		return false
	}

	// Core optimizations: only GoldCopy Admin
	if opt.Scope == ASOScopeCore {
		return HasRole(ac, RoleGoldCopyAdmin)
	}

	// Tenant optimizations: Tenant Admin, GlobalOps, or GoldCopy Admin
	if opt.TenantID != nil {
		if !CanViewTenant(ac, *opt.TenantID) {
			return false
		}
		return HasRole(ac, RoleTenantAdmin) || HasRole(ac, RoleGlobalOps) || HasRole(ac, RoleGoldCopyAdmin)
	}

	return false
}

// CanApplyOptimization checks if user can apply an optimization
func CanApplyOptimization(ac *AuthContext, opt *ASOOptimization, policy *ASOPolicy) bool {
	if ac == nil || opt == nil {
		return false
	}

	// Core optimizations: only GoldCopy Admin
	if opt.Scope == ASOScopeCore {
		return HasRole(ac, RoleGoldCopyAdmin)
	}

	// Tenant optimizations
	if opt.TenantID != nil {
		if !CanViewTenant(ac, *opt.TenantID) {
			return false
		}

		// If policy is auto_apply, anyone with access can trigger
		if policy != nil && policy.Mode == ASOModeAutoApply {
			return HasRole(ac, RoleTenantAdmin) || HasRole(ac, RoleTenantOps) ||
				HasRole(ac, RoleGlobalOps) || HasRole(ac, RoleGoldCopyAdmin)
		}

		// Manual apply: need at least TenantAdmin
		return HasRole(ac, RoleTenantAdmin) || HasRole(ac, RoleGlobalOps) || HasRole(ac, RoleGoldCopyAdmin)
	}

	return false
}

// CanRejectOptimization checks if user can reject an optimization
func CanRejectOptimization(ac *AuthContext, opt *ASOOptimization) bool {
	// Same as approve
	return CanApproveOptimization(ac, opt)
}

// CanConfigurePolicy checks if user can modify ASO policy
func CanConfigurePolicy(ac *AuthContext, policy *ASOPolicy) bool {
	if ac == nil || policy == nil {
		return false
	}

	// Core policy: only GoldCopy Admin
	if policy.TenantID == nil {
		return HasRole(ac, RoleGoldCopyAdmin)
	}

	// Tenant policy: Tenant Admin, GlobalOps, GoldCopy Admin
	if CanViewTenant(ac, *policy.TenantID) {
		return HasRole(ac, RoleTenantAdmin) || HasRole(ac, RoleGlobalOps) || HasRole(ac, RoleGoldCopyAdmin)
	}

	return false
}

// CanEnableAutoApply checks if user can enable auto-apply mode
func CanEnableAutoApply(ac *AuthContext, policy *ASOPolicy) bool {
	if ac == nil || policy == nil {
		return false
	}

	// Auto-apply requires elevated permissions
	if policy.TenantID == nil {
		// Core: only GoldCopy Admin
		return HasRole(ac, RoleGoldCopyAdmin)
	}

	// Tenant: GoldCopy Admin or GlobalOps (not TenantAdmin for safety)
	return HasRole(ac, RoleGoldCopyAdmin) || HasRole(ac, RoleGlobalOps)
}

// CanTriggerExperiment checks if user can start an A/B experiment
func CanTriggerExperiment(ac *AuthContext, opt *ASOOptimization) bool {
	// Same as apply
	return CanApplyOptimization(ac, opt, nil)
}

// ============================================================================
// Context Helpers
// ============================================================================

type authContextKey struct{}

// WithAuthContext adds auth context to context
func WithAuthContext(ctx context.Context, ac *AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey{}, ac)
}

// AuthFromContext retrieves auth context from context
func AuthFromContext(ctx context.Context) *AuthContext {
	ac, _ := ctx.Value(authContextKey{}).(*AuthContext)
	return ac
}

// ============================================================================
// Authorization Result
// ============================================================================

// AuthorizationResult contains authorization check result
type AuthorizationResult struct {
	Allowed bool
	Reason  string
}

// AuthorizeOptimizationAction performs full authorization check
func AuthorizeOptimizationAction(ac *AuthContext, action string, opt *ASOOptimization, policy *ASOPolicy) AuthorizationResult {
	if ac == nil {
		return AuthorizationResult{Allowed: false, Reason: "Not authenticated"}
	}

	switch action {
	case "view":
		if CanViewOptimization(ac, opt) {
			return AuthorizationResult{Allowed: true}
		}
		return AuthorizationResult{Allowed: false, Reason: "Not authorized to view this optimization"}

	case "approve":
		if CanApproveOptimization(ac, opt) {
			return AuthorizationResult{Allowed: true}
		}
		return AuthorizationResult{Allowed: false, Reason: "Not authorized to approve this optimization"}

	case "apply":
		if CanApplyOptimization(ac, opt, policy) {
			return AuthorizationResult{Allowed: true}
		}
		return AuthorizationResult{Allowed: false, Reason: "Not authorized to apply this optimization"}

	case "reject":
		if CanRejectOptimization(ac, opt) {
			return AuthorizationResult{Allowed: true}
		}
		return AuthorizationResult{Allowed: false, Reason: "Not authorized to reject this optimization"}

	default:
		return AuthorizationResult{Allowed: false, Reason: "Unknown action"}
	}
}

// ============================================================================
// Permission Matrix (for UI display)
// ============================================================================

// PermissionSet describes what actions a user can take
type PermissionSet struct {
	CanView     bool `json:"can_view"`
	CanApprove  bool `json:"can_approve"`
	CanApply    bool `json:"can_apply"`
	CanReject   bool `json:"can_reject"`
	CanSimulate bool `json:"can_simulate"`
}

// GetPermissions returns the permission set for an optimization
func GetPermissions(ac *AuthContext, opt *ASOOptimization, policy *ASOPolicy) PermissionSet {
	return PermissionSet{
		CanView:     CanViewOptimization(ac, opt),
		CanApprove:  CanApproveOptimization(ac, opt),
		CanApply:    CanApplyOptimization(ac, opt, policy),
		CanReject:   CanRejectOptimization(ac, opt),
		CanSimulate: CanViewOptimization(ac, opt), // Anyone who can view can simulate
	}
}
