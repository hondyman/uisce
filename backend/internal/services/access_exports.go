// Package services - Access domain facade shim.
//
// This file re-exports types from the new internal/access domain package
// so existing services-package consumers (collaboration_service.go,
// load_tester.go, qos_manager.go, etc.) can gradually migrate to direct
// internal/access imports without breaking the monolithic package.
//
// Cardinal Rule 3 (no cycles): This shim only depends on internal/access,
// which itself does NOT depend on internal/services. Zero back-coupling.
//
// Phase 2 of microservice extraction: existing code keeps using
// services.AccessIntelligenceX aliases while new code can import
// internal/access directly. The facade will be deleted in a future
// commit once all consumers migrate.
package services

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/access"
	"github.com/hondyman/uisce/backend/models"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// ACCESS DOMAIN TYPE ALIASES
// ============================================================================

// AccessIntelligenceSvc aliases the canonical access service type.
// The original services.AccessIntelligenceService remains until
// consumers migrate.
type AccessIntelligenceSvc = access.AccessIntelligenceService

// ============================================================================
// POLICY REPOSITORY CONSTRUCTOR
// ============================================================================

// NewAccessPolicyRepository delegates to internal/access.NewSqlAccessPolicyRepository.
func NewAccessPolicyRepository(db *sqlx.DB) access.AccessPolicyRepository {
	return access.NewSqlAccessPolicyRepository(db)
}

// ============================================================================
// LEGACY POLICY REPOSITORY FACTORY (backward compat)
// ============================================================================

// NewPgAccessPolicyRepository returns the canonical access policy repository.
// (Renamed from services package for backward compatibility.)
func NewPgAccessPolicyRepository(db *sqlx.DB) access.AccessPolicyRepository {
	return access.NewSqlAccessPolicyRepository(db)
}

// Suppress unused warnings for uuid, models, and sql (used by callers, not here).
var (
	_ = uuid.Nil
	_ = models.AccessControlPolicy{}
	_ sql.DB
	_ = sqlx.DB{}
)