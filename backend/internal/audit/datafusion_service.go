package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/datafusion"
	"github.com/hondyman/uisce/backend/internal/events"
	"github.com/hondyman/uisce/backend/internal/logging"
)

type AuditLogger interface {
	LogEvent(ctx context.Context, tenantID, userID, userEmail, userName, action, resourceType, resourceID string, details map[string]interface{}) error
}

type DataFusionAuditService struct {
	client *datafusion.Client
}

func NewDataFusionAuditService(client *datafusion.Client) *DataFusionAuditService {
	return &DataFusionAuditService{client: client}
}

func (s *DataFusionAuditService) WriteEvent(ctx context.Context, event events.GoldCopyConnectionEvent) error {
	var userID string
	if event.UserID != nil {
		userID = *event.UserID
	}

	return s.LogEvent(
		ctx,
		event.TenantID,
		userID,
		"",
		"",
		event.Action,
		"connection",
		event.ConnectionID,
		event.ConnectionData,
	)
}

func (s *DataFusionAuditService) LogEvent(ctx context.Context, tenantID, userID, userEmail, userName, action, resourceType, resourceID string, details map[string]interface{}) error {
	if s.client == nil {
		return fmt.Errorf("datafusion client is nil")
	}

	id := uuid.New().String()
	timestamp := time.Now().UTC()

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

	query := fmt.Sprintf(`
		INSERT INTO iceberg.audit.audit_logs
		(id, tenant_id, timestamp, user_name, user_email, action, resource, resource_type, details)
		VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s')
	`,
		id,
		tenantID,
		timestamp.Format(time.RFC3339Nano),
		userName,
		userEmail,
		action,
		resourceID,
		resourceType,
		detailsStr,
	)

	_, err := s.client.QueryToMap(ctx, query)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to write audit log to DataFusion: %v", err)
		return err
	}

	logging.GetLogger().Sugar().Infof("Audit logged: %s %s %s (Tenant: %s)", action, resourceType, resourceID, tenantID)
	return nil
}
