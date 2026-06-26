package services

import (
	"context"
)

// OAuthAuditService handles recording of audit events
type OAuthAuditService struct {
	hasuraClient HasuraClient
}

// NewOAuthAuditService creates a new audit service
func NewOAuthAuditService(hasuraClient HasuraClient) *OAuthAuditService {
	return &OAuthAuditService{
		hasuraClient: hasuraClient,
	}
}

// OAuthAuditEvent represents an OAuth-related audit event
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

// RecordOAuthEvent logs OAuth-related actions for compliance
func (s *OAuthAuditService) RecordOAuthEvent(ctx context.Context, event OAuthAuditEvent) error {
	mutation := `
	mutation InsertOAuthAudit($object: oauth_audit_logs_insert_input!) {
		insert_oauth_audit_logs_one(object: $object) {
			id
			created_at
		}
	}
	`

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
