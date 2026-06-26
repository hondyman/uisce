package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

// SecurityAuditLog represents a security event for audit logging
type SecurityAuditLog struct {
	UserID        string
	TenantID      string
	IsGlobalAdmin bool
	Action        string
	Resource      string
	ResourceID    string
	IPAddress     string
	UserAgent     string
	SessionID     string
}

// LogSecurityEvent logs a security event to the security_audit_log table
func LogSecurityEvent(ctx context.Context, db *sql.DB, event SecurityAuditLog) {
	if db == nil {
		return
	}

	_, err := db.ExecContext(ctx, `
		INSERT INTO security_audit_log 
		(user_id, tenant_id, is_global_admin, action, resource, resource_id, ip_address, user_agent, session_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, event.UserID, event.TenantID, event.IsGlobalAdmin, event.Action, event.Resource,
		event.ResourceID, event.IPAddress, event.UserAgent, event.SessionID)

	if err != nil {
		log.Printf("[SECURITY-AUDIT] Failed to log security event: %v", err)
	}
}

// IsGlobalAdmin checks if the user is a global administrator (Uisce organization)
func IsGlobalAdmin(db *sql.DB, userID string) bool {
	if db == nil {
		return false
	}

	var isAdmin bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM users 
			WHERE id = $1 
			AND organization = 'uisce' 
			AND role = 'admin'
		)
	`, userID).Scan(&isAdmin)

	if err != nil {
		log.Printf("[SECURITY] Failed to check global admin status for user %s: %v", userID, err)
		return false
	}

	return isAdmin
}

// GetTenantIDForUser retrieves the tenant ID for a given user
func GetTenantIDForUser(db *sql.DB, userID string) (string, error) {
	if db == nil {
		return "", fmt.Errorf("database connection is nil")
	}

	var tenantID sql.NullString
	err := db.QueryRow(`
		SELECT tenant_id FROM users WHERE id = $1
	`, userID).Scan(&tenantID)

	if err != nil {
		return "", fmt.Errorf("failed to get tenant for user %s: %w", userID, err)
	}

	if !tenantID.Valid {
		return "", fmt.Errorf("user %s has no tenant_id", userID)
	}

	return tenantID.String, nil
}

// SetTenantContext sets the PostgreSQL session variable for RLS
func SetTenantContext(ctx context.Context, db *sql.DB, tenantID string) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	_, err := db.ExecContext(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID)
	if err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	return nil
}

// SetGlobalAdminContext sets the session variable indicating global admin access
func SetGlobalAdminContext(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	_, err := db.ExecContext(ctx, "SET LOCAL app.is_global_admin = 'true'")
	if err != nil {
		return fmt.Errorf("failed to set global admin context: %w", err)
	}

	return nil
}
