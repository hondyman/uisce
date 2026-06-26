package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/events"
	"github.com/hondyman/semlayer/backend/internal/logging"
)

// TrinoAuditService handles writing audit logs to Iceberg via Trino
type TrinoAuditService struct {
	db *sql.DB
}

// NewTrinoAuditService creates a new audit service
func NewTrinoAuditService(db *sql.DB) *TrinoAuditService {
	return &TrinoAuditService{db: db}
}

// WriteEvent implements the AuditService interface for GoldCopyActivities
func (s *TrinoAuditService) WriteEvent(ctx context.Context, event events.GoldCopyConnectionEvent) error {
	var userID string
	if event.UserID != nil {
		userID = *event.UserID
	}

	return s.LogEvent(
		ctx,
		event.TenantID,
		userID,
		"", // Email not available in event
		"", // Name not available in event
		event.Action,
		"connection",
		event.ConnectionID,
		event.ConnectionData,
	)
}

// LogEvent writes a generic audit event to Trino
func (s *TrinoAuditService) LogEvent(ctx context.Context, tenantID, userID, userEmail, userName, action, resourceType, resourceID string, details map[string]interface{}) error {
	if s.db == nil {
		return fmt.Errorf("trino db connection is nil")
	}

	id := uuid.New().String()
	timestamp := time.Now().UTC()

	// Marshal details to JSON string
	var detailsStr string
	if details != nil {
		bytes, err := json.Marshal(details)
		if err == nil {
			detailsStr = string(bytes)
		} else {
			detailsStr = "{}"
		}
	} else {
		detailsStr = "{}"
	}

	// Execute INSERT
	// Note: We use the schema iceberg.audit.audit_logs we created earlier
	query := `
		INSERT INTO iceberg.audit.audit_logs 
		(id, tenant_id, timestamp, user_name, user_email, action, resource, resource_type, details)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Trino go client supports ? placeholders
	_, err := s.db.ExecContext(ctx, query,
		id,
		tenantID,
		timestamp,
		userName,
		userEmail,
		action,
		resourceID,
		resourceType,
		detailsStr,
	)

	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to write audit log to Trino: %v", err)
		return err
	}

	logging.GetLogger().Sugar().Infof("Audit logged: %s %s %s (Tenant: %s)", action, resourceType, resourceID, tenantID)
	return nil
}
