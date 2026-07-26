// Package services - BO domain facade shim.
//
// This file re-exports types from the new internal/bo domain package so that
// existing consumers (api.go, cmd/, workflows, etc.) can gradually migrate to
// direct internal/bo imports without breaking the monolithic package.
//
// Cardinal Rule 3 (no cycles): This shim only depends on internal/bo, which
// itself only depends on libs/* + stdlib + zap. Zero back-coupling.
//
// Cardinal Rule 7 (tenant security): The internal/bo types are the source of
// truth for governance decisions. Existing services-package types remain
// until consumers migrate.
//
// Phase 1 of microservice extraction: existing code keeps using
// services.BusinessObjectX aliases while new code can import internal/bo
// directly. The facade will be deleted in a future commit once all
// consumers migrate.
package services

import (
	"database/sql"

	"go.uber.org/zap"

	"github.com/hondyman/uisce/backend/internal/bo"
)

// ============================================================================
// BO DOMAIN TYPE ALIASES (non-conflicting only)
// ============================================================================

type (
	BusinessObjectSvc        = bo.BusinessObjectService
	BOAccessController       = bo.AccessController
	BOAccessPolicy           = bo.AccessPolicy
	BOBusinessProcess        = bo.BusinessProcess
	BOFieldClassification    = bo.FieldClassification
	BOFieldMaskResult        = bo.FieldMaskResult
	BOFieldSecurityConfig    = bo.FieldSecurityConfig
	BOFieldSecurityMasker    = bo.FieldSecurityMasker
	BOFieldType              = bo.FieldType
	BOFieldVisibility        = bo.FieldVisibility
	BOGovernanceHandlers     = bo.GovernanceHandlers
	BOPageLayout             = bo.PageLayout
	BOPolicyEngine           = bo.PolicyEngine
	BOPolicyEvalContext      = bo.PolicyEvalContext
	BOPolicyEvalResult       = bo.PolicyEvalResult
	BOProcessInstance        = bo.ProcessInstance
	BOProcessStep            = bo.ProcessStep
	BOValidationEngine       = bo.ValidationEngine
	BOValidationResult       = bo.ValidationResult
)

// ============================================================================
// BO DEPENDENCY INTERFACES (re-exported for legacy callers)
// ============================================================================

// BOHasuraClient is the Hasura client interface used by BO services.
// Concrete implementations in services/ satisfy this contract.
type BOHasuraClient = bo.HasuraClient

// BOAccessRuleRepository is the access rule repository interface.
type BOAccessRuleRepository = bo.AccessRuleRepository

// BOAccessRule is a single access rule from the repository.
type BOAccessRule = bo.AccessRule

// BOAccessLevel constants (NONE, READ, WRITE) re-exported.
const (
	BOAccessLevelNone  = bo.AccessLevelNone
	BOAccessLevelRead  = bo.AccessLevelRead
	BOAccessLevelWrite = bo.AccessLevelWrite
)

// ============================================================================
// CONSTRUCTOR WRAPPERS
// ============================================================================

func NewBOAccessController(db *sql.DB, log *zap.Logger) *bo.AccessController {
	return bo.NewAccessController(db, log)
}

func NewBOFieldSecurityMasker(db *sql.DB, log *zap.Logger) *bo.FieldSecurityMasker {
	return bo.NewFieldSecurityMasker(db, log)
}

func NewBOGovernanceHandlers(db *sql.DB, validator *bo.ValidationEngine, policies *bo.PolicyEngine, log *zap.Logger) *bo.GovernanceHandlers {
	return bo.NewGovernanceHandlers(db, validator, policies, nil, nil, log)
}

func NewBOPolicyEngine(db *sql.DB, log *zap.Logger) *bo.PolicyEngine {
	return bo.NewPolicyEngine(db, log)
}

func NewBOValidationEngine(db *sql.DB, log *zap.Logger) *bo.ValidationEngine {
	return bo.NewValidationEngine(db, log)
}

// NewBusinessObjectService delegates to internal/bo.NewBusinessObjectService.
// Accepts interface{} for backward compat (was *sqlx.DB or *sql.DB).
func NewBusinessObjectService(db interface{}) *bo.BusinessObjectService {
	return bo.NewBusinessObjectService(db)
}

// NewBusinessObjectServiceWithHasura delegates to internal/bo.
// accepts any hasura client implementing the BO HasuraClient interface.
func NewBusinessObjectServiceWithHasura(db interface{}, hasura bo.HasuraClient) *bo.BusinessObjectService {
	return bo.NewBusinessObjectServiceWithHasura(db, hasura)
}