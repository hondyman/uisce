// Package services - ABAC domain facade shim.
//
// This file re-exports types from the new internal/abac domain package
// so existing services-package consumers can gradually migrate to direct
// internal/abac imports without breaking the monolithic package.
//
// Cardinal Rule 3 (no cycles): This shim only depends on internal/abac,
// which itself does NOT depend on internal/services. Zero back-coupling.
//
// Phase 3 of microservice extraction: existing code keeps using the
// services-package types while new code can import internal/abac directly.
// The facade will be deleted in a future commit once all consumers migrate.
package services

import (
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/abac"
)

// ============================================================================
// ABAC DOMAIN TYPE ALIASES (non-conflicting only)
// ============================================================================

// Types that conflict with existing services-package types
// (AccessRuleRepository interface, AccessDecision struct, Principal struct)
// remain defined in services/ until consumers migrate.
//
// We re-export only the truly canonical types here.

// ABAccessLevel is the canonical access level enum.
type ABAccessLevel = abac.AccessLevel

// ABAccessDecision is the canonical composed decision struct.
type ABAccessDecision = abac.AccessDecision

// ABPrincipal is the canonical principal carrier.
type ABPrincipal = abac.Principal

// ============================================================================
// CONSTRUCTOR WRAPPERS
// ============================================================================

// NewAbAccessRuleRepository delegates to internal/abac.NewPgAccessRuleRepository.
func NewAbAccessRuleRepository(db *sqlx.DB) abac.AccessRuleRepository {
	return abac.NewPgAccessRuleRepository(db)
}