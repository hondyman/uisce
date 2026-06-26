package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// LogEvent writes audit immediately to Postgres
func (s *Service) LogEvent(ctx context.Context, event UnifiedAuditRecord) error {
	// Map UnifiedAuditRecord to workflow_audit_log
	// This is a best-effort mapping to the existing table structure
	var oldJSON, newJSON []byte

	// Try to extract old/new values from metadata if present
	if oldMap, ok := event.Metadata["old_value"].(map[string]interface{}); ok {
		oldJSON, _ = json.Marshal(oldMap)
	}
	if newMap, ok := event.Metadata["new_value"].(map[string]interface{}); ok {
		newJSON, _ = json.Marshal(newMap)
	}

	// Use Object ID as Instance ID if generic
	instanceID := event.ObjectID
	if event.CorrelationID != "" {
		instanceID = event.CorrelationID
	}

	// Determine step key (not present in generic audit, use event type suffix?)
	stepKey := ""

	// Determine role
	actorRole := ""
	if len(event.Roles) > 0 {
		actorRole = event.Roles[0]
	}

	ipAddr, _ := event.Metadata["ip_address"].(string)
	userAgent, _ := event.Metadata["user_agent"].(string)

	// Ensure ID is generated
	id := event.AuditID
	if id == "" {
		id = uuid.New().String()
	}

	// Write to Postgres
	_, err := s.db.ExecContext(ctx, `
        INSERT INTO workflow_audit_log
        (id, instance_id, event_type, step_key, actor_id, actor_role, reason, ip_address, user_agent, old_value, new_value, created_at, tenant_id)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
    `, id, instanceID, event.EventType, stepKey, event.ActorID, actorRole, event.Narrative, ipAddr, userAgent, oldJSON, newJSON, event.Timestamp, event.TenantID)
	return err
}

// LogDataAccess logs a data access event
func (s *Service) LogDataAccess(ctx context.Context, actorID, tenantID, objectID, objectType, objectName, action string, details map[string]interface{}) error {
	event := UnifiedAuditRecord{
		EventType:  "data_access",
		TenantID:   tenantID,
		ActorID:    actorID,
		ObjectType: objectType,
		ObjectID:   objectID,
		Narrative:  fmt.Sprintf("%s %s %s", action, objectType, objectName),
		Metadata:   details,
		Timestamp:  time.Now(),
	}
	return s.LogEvent(ctx, event)
}

// LogDataModification logs a data modification event
func (s *Service) LogDataModification(ctx context.Context, actorID, tenantID, objectID, objectType, objectName, action string, oldData, newData interface{}) error {
	metadata := make(map[string]interface{})
	metadata["old_value"] = oldData
	metadata["new_value"] = newData
	metadata["object_name"] = objectName

	event := UnifiedAuditRecord{
		EventType:  "data_modify",
		TenantID:   tenantID,
		ActorID:    actorID,
		ObjectType: objectType,
		ObjectID:   objectID,
		Narrative:  fmt.Sprintf("%s %s %s", action, objectType, objectName),
		Metadata:   metadata,
		Timestamp:  time.Now(),
	}
	return s.LogEvent(ctx, event)
}

func (s *Service) VerifyChain(ctx context.Context, objectID string) (bool, error) {
	// Query all events for this object ordered by timestamp
	query := `
		SELECT event_hash, previous_hash, timestamp
		FROM workflow_audit_log
		WHERE object_id = $1
		ORDER BY timestamp ASC
	`

	rows, err := s.db.QueryContext(ctx, query, objectID)
	if err != nil {
		return false, fmt.Errorf("failed to query audit chain: %w", err)
	}
	defer rows.Close()

	var previousHash string
	isFirst := true

	for rows.Next() {
		var eventHash, storedPreviousHash string
		var timestamp time.Time

		if err := rows.Scan(&eventHash, &storedPreviousHash, &timestamp); err != nil {
			return false, fmt.Errorf("failed to scan audit event: %w", err)
		}

		if isFirst {
			// First event should have empty previous hash
			if storedPreviousHash != "" {
				return false, fmt.Errorf("chain broken: first event has non-empty previous hash")
			}
			isFirst = false
		} else {
			// Verify chain link
			if storedPreviousHash != previousHash {
				return false, fmt.Errorf("chain broken: previous hash mismatch at %s", timestamp)
			}
		}

		previousHash = eventHash
	}

	return true, nil
}

// QueryEvents retrieves audit events based on filter
func (s *Service) QueryEvents(ctx context.Context, filter *models.AuditEventFilter) ([]models.AuditEvent, error) {
	query := `SELECT id, instance_id, event_type, actor_id, actor_role, reason, created_at, ip_address, user_agent 
	          FROM workflow_audit_log WHERE 1=1`
	var args []interface{}
	idx := 1

	if filter.TenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", idx) // Assuming table has tenant_id, if not, skip
		args = append(args, *filter.TenantID)
		idx++
	}
	if filter.UserID != nil {
		query += fmt.Sprintf(" AND actor_id = $%d", idx)
		args = append(args, *filter.UserID)
		idx++
	}
	// ... minimal implementation for compilation ...

	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.AuditEvent
	for rows.Next() {
		var e models.AuditEvent
		var instanceID, actorRole, ip, ua, reason string

		if err := rows.Scan(&e.ID, &instanceID, &e.EventType, &e.UserID, &actorRole, &reason, &e.Timestamp, &ip, &ua); err != nil {
			continue
		}

		e.ResourceID = instanceID
		e.Action = reason
		events = append(events, e)
	}
	return events, nil
}

// GetAuditSummary returns summary stats
func (s *Service) GetAuditSummary(ctx context.Context, tenantID *string, start, end time.Time) (*models.AuditSummary, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			event_type,
			COUNT(DISTINCT actor_id) as unique_actors
		FROM workflow_audit_log
		WHERE created_at >= $1 AND created_at <= $2
	`
	args := []interface{}{start, end}

	if tenantID != nil {
		query += " AND tenant_id = $3"
		args = append(args, *tenantID)
	}

	query += " GROUP BY event_type"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit summary: %w", err)
	}
	defer rows.Close()

	summary := &models.AuditSummary{
		EventsByType: map[models.AuditEventType]int64{},
	}

	for rows.Next() {
		var total int64
		var eventType string
		var uniqueActors int64

		if err := rows.Scan(&total, &eventType, &uniqueActors); err != nil {
			continue
		}

		summary.TotalEvents += total
		summary.EventsByType[models.AuditEventType(eventType)] = total
	}

	return summary, nil
}

// CleanupOldEvents deletes old logs
func (s *Service) CleanupOldEvents(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM workflow_audit_log WHERE created_at < NOW() - INTERVAL '90 days'")
	return err
}
