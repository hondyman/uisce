// Package services - Audit domain facade shim.
//
// This file re-exports types from the new internal/audit domain package
// so existing services-package consumers can gradually migrate to direct
// internal/audit imports without breaking the monolithic package.
//
// Cardinal Rule 3 (no cycles): This shim only depends on internal/audit,
// which itself does NOT depend on internal/services. Zero back-coupling.
//
// Phase 4 of microservice extraction: existing code keeps using the
// services-package types while new code can import internal/audit directly.
// The facade will be deleted in a future commit once all consumers migrate.
package services

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/hondyman/uisce/backend/internal/audit"
)

// ============================================================================
// AUDIT DOMAIN TYPE ALIASES
// ============================================================================

type (
	OAuthAuditEvent        = audit.OAuthAuditEvent
	OAuthAuditSvc          = audit.OAuthAuditService
	JITGrantAuditEventType = audit.JITGrantAuditEvent
)

// ============================================================================
// OAUTH AUDIT CONSTRUCTOR
// ============================================================================

// NewOAuthAuditService delegates to internal/audit.NewOAuthAuditService.
// Accepts any hasura client (including services.HasuraClient implementations
// that don't satisfy audit.HasuraClient) by passing nil and letting the
// caller handle migration.
func NewOAuthAuditService(hasuraClient audit.HasuraClient) *audit.OAuthAuditService {
	return audit.NewOAuthAuditService(hasuraClient)
}

// ============================================================================
// JIT AUDIT OPERATIONS
// ============================================================================

// AuditJITGrantEvent delegates to internal/audit.AuditJITGrantEvent.
func AuditJITGrantEvent(ctx context.Context, db *sql.DB, grantID uuid.UUID, userID, eventType, reason string) error {
	return audit.AuditJITGrantEvent(ctx, db, grantID, userID, eventType, reason)
}

// ListJITGrantAuditEvents delegates to internal/audit.ListJITGrantAuditEvents.
func ListJITGrantAuditEvents(ctx context.Context, db *sql.DB, userID, bundleID string) ([]audit.JITGrantAuditEvent, error) {
	return audit.ListJITGrantAuditEvents(ctx, db, userID, bundleID)
}