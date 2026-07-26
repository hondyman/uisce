// Package audit - OAuth-specific audit logging.
//
// This file contains audit event types and services for OAuth provider
// interactions. It was extracted from internal/services/audit_service.go.
//
// Cardinal Rule 3 (no cycles): only depends on internal/* + libs/* + stdlib.
// Cardinal Rule 7 (tenant security): every event carries tenantID.
package audit

import (
	"context"
)

// ============================================================================
// HASURA CLIENT INTERFACE
// ============================================================================

// HasuraClient is the minimal Hasura mutation interface required to record
// audit events. Concrete implementations live in internal/oauth and other
// packages; this avoids an internal/audit -> internal/services cycle.
type HasuraClient interface {
	Mutate(mutation string, variables map[string]interface{}) (interface{}, error)
}

// ============================================================================
// OAUTH AUDIT EVENT TYPES
// ============================================================================

// OAuthAuditEvent represents an OAuth-related audit event.
type OAuthAuditEvent struct {
	TenantID  string      `json:"tenant_id"`
	UserID    string      `json:"user_id"`
	Action    string      `json:"action"`   // "token_saved", "token_refreshed", "token_deleted"
	Provider  string      `json:"provider"` // "google"
	Metadata  interface{} `json:"metadata"` // JSONB with non-sensitive details
	IPAddress string      `json:"ip_address"`
	UserAgent string      `json:"user_agent"`
	Success   bool        `json:"success"`
	Error     string      `json:"error"`
}

// ============================================================================
// OAUTH AUDIT SERVICE
// ============================================================================

// OAuthAuditService records OAuth-related audit events for compliance.
type OAuthAuditService struct {
	hasuraClient HasuraClient
}

// NewOAuthAuditService constructs a new OAuthAuditService.
func NewOAuthAuditService(hasuraClient HasuraClient) *OAuthAuditService {
	return &OAuthAuditService{
		hasuraClient: hasuraClient,
	}
}

// RecordOAuthEvent logs an OAuth-related action for compliance.
func (s *OAuthAuditService) RecordOAuthEvent(ctx context.Context, event OAuthAuditEvent) error {
	if s.hasuraClient == nil {
		return nil
	}
	mutation := `
	mutation InsertOAuthAudit($object: oauth_audit_logs_insert_input!) {
		insert_oauth_audit_logs_one(object: $object) {
			id
			created_at
		}
	}`
	object := map[string]interface{}{
		"tenant_id":  event.TenantID,
		"user_id":    event.UserID,
		"action":     event.Action,
		"provider":   event.Provider,
		"metadata":   event.Metadata,
		"ip_address": event.IPAddress,
		"user_agent": event.UserAgent,
		"success":    event.Success,
		"error":      event.Error,
	}
	_, err := s.hasuraClient.Mutate(mutation, map[string]interface{}{"object": object})
	return err
}