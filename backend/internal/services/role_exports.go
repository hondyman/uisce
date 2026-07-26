// Package services - Role domain facade shim.
//
// This file re-exports the RoleService interface and supporting types
// from the new internal/role domain package so that existing services-
// package consumers (handlers/role_handler.go) can gradually migrate to
// direct internal/role imports without breaking the monolithic package.
//
// Cardinal Rule 3 (no cycles): This shim only depends on internal/role,
// which itself does NOT depend on internal/services. Zero back-coupling.
//
// Cardinal Rule 7 (tenant security): tenantID is required on every role
// operation via the embedded User identity model.
package services

import (
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/platform"
	"github.com/hondyman/uisce/backend/internal/role"
)

// ============================================================================
// ROLE DOMAIN TYPE ALIASES
// ============================================================================

// RoleService is the canonical role service interface (type alias for migration).
type RoleService = role.RoleService

// RoleCreateInput aliases the role create input type.
type RoleCreateInput = role.RoleCreateInput

// RoleUpdateInput aliases the role update input type.
type RoleUpdateInput = role.RoleUpdateInput

// RoleBundleManager aliases the bundle manager interface.
type RoleBundleManager = role.BundleRoleManager

// ============================================================================
// ROLE ERRORS
// ============================================================================

var (
	ErrRoleNotFound                  = role.ErrRoleNotFound
	ErrRoleRepositoryUnavailable     = role.ErrRoleRepositoryUnavailable
)

// ============================================================================
// ROLE CONSTRUCTOR WRAPPER
// ============================================================================

// NewRoleService delegates to internal/role.NewRoleService.
// Accepts any BundleRoleManager implementation (concrete types in
// services/ satisfy the role.BundleRoleManager contract).
func NewRoleService(db *sqlx.DB, policyService platform.PolicyService, bundleAccess role.BundleRoleManager) role.RoleService {
	return role.NewRoleService(db, policyService, bundleAccess)
}

// NewRoleBundleManagerAdapter adapts a services.BundleRoleManager to a role.BundleRoleManager.
// Use this when constructing services that need cross-package compatibility.
func NewRoleBundleManagerAdapter(svc interface {
	AssignRoleToBundle(roleName, bundleID string) error
	UnassignRoleFromBundle(roleName, bundleID string) error
	GetBundleIDsForRole(roleName string) []string
	GetRolesForBundle(bundleID string) []string
}) role.BundleRoleManager {
	return &roleBundleManagerAdapter{inner: svc}
}

type roleBundleManagerAdapter struct {
	inner interface {
		AssignRoleToBundle(roleName, bundleID string) error
		UnassignRoleFromBundle(roleName, bundleID string) error
		GetBundleIDsForRole(roleName string) []string
		GetRolesForBundle(bundleID string) []string
	}
}

func (a *roleBundleManagerAdapter) AssignRoleToBundle(roleName, bundleID string) error {
	return a.inner.AssignRoleToBundle(roleName, bundleID)
}

func (a *roleBundleManagerAdapter) UnassignRoleFromBundle(roleName, bundleID string) error {
	return a.inner.UnassignRoleFromBundle(roleName, bundleID)
}

func (a *roleBundleManagerAdapter) GetBundleIDsForRole(roleName string) []string {
	return a.inner.GetBundleIDsForRole(roleName)
}

func (a *roleBundleManagerAdapter) GetRolesForBundle(bundleID string) []string {
	return a.inner.GetRolesForBundle(bundleID)
}