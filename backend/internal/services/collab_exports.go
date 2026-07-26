// Package services - Collaboration domain facade shim.
//
// This file re-exports the CollaborationService from the new
// internal/collab domain package so existing services-package consumers
// (handlers/collaboration_handler.go) can gradually migrate to direct
// internal/collab imports without breaking the monolithic package.
//
// Cardinal Rule 3 (no cycles): This shim only depends on internal/collab,
// which itself does NOT depend on internal/services. Zero back-coupling.
package services

import (
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/collab"
)

// ============================================================================
// COLLAB DOMAIN TYPE ALIASES
// ============================================================================

// CollaborationService aliases the canonical collaboration service.
type CollaborationService = collab.CollaborationService

// ============================================================================
// COLLAB CONSTRUCTOR WRAPPER
// ============================================================================

// NewCollaborationService delegates to internal/collab.NewCollaborationService.
func NewCollaborationService(db *sqlx.DB) *collab.CollaborationService {
	return collab.NewCollaborationService(db)
}