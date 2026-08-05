package services

import (
	"context"
	"database/sql"
)

// OAuthAuditService handles recording of audit events
type OAuthAuditService struct {
	db *sql.DB
}

// NewOAuthAuditService creates a new audit service
func NewOAuthAuditService(db *sql.DB) *OAuthAuditService {
	return &OAuthAuditService{
		db: db,
	}
}

// OAuthAuditEvent represents an OAuth-related audit event
type OAuthAuditEvent struct {
	TenantID  string      `json:"tenant_id"`
	UserID    string      `json:"user_id"`
	Action    string      `json:"action"`    // "token_saved", "token_refreshed", "token_deleted"
	Provider  string      `json:"provider"`  // "google"
	Metadata  interface{} `json:"metadata"`  // JSONB with non-sensitive details
	IPAddress string      `json:"ip_address"`
	UserAgent string      `json:"user_agent"`
	Success   bool        `json:"success"`
	Error     string      `json:"error"`
}

// RecordOAuthEvent logs OAuth-related actions for compliance
func (s *OAuthAuditService) RecordOAuthEvent(ctx context.Context, event OAuthAuditEvent) error {
	query := `
		INSERT INTO oauth_audit_logs (tenant_id, user_id, action, provider, metadata, ip_address, user_agent, success, error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`
	_, err := s.db.ExecContext(ctx, query,
		event.TenantID, event.UserID, event.Action, event.Provider,
		event.Metadata, event.IPAddress, event.UserAgent, event.Success, event.Error,
	)
	return err
}
