package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"calendar-service/internal/middleware"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// AuditService handles audit logging for all mutations
type AuditService interface {
	RecordCreate(ctx context.Context, tenantID, entityType, entityID string, newValues map[string]interface{}, actorID string) error
	RecordUpdate(ctx context.Context, tenantID, entityType, entityID string, oldValues, newValues map[string]interface{}, actorID string) error
	RecordDelete(ctx context.Context, tenantID, entityType, entityID string, oldValues map[string]interface{}, actorID string) error
	Record(ctx context.Context, entry AuditEntry) error
}

// AuditEntry represents a complete audit record
type AuditEntry struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	EntityType string                 `json:"entity_type"` // calendar, profile, blackout, etc
	EntityID   string                 `json:"entity_id"`
	Action     string                 `json:"action"` // CREATE, UPDATE, DELETE
	OldValues  map[string]interface{} `json:"old_values,omitempty"`
	NewValues  map[string]interface{} `json:"new_values,omitempty"`
	ChangedBy  string                 `json:"changed_by"` // user_id from JWT
	ChangedAt  time.Time              `json:"changed_at"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	Reason     string                 `json:"reason,omitempty"`
}

// AuditServiceImpl provides audit logging functionality
type AuditServiceImpl struct {
	logger *logrus.Entry
	// In-memory storage for demo (replace with DB in production)
	entries []AuditEntry
}

// NewAuditService creates a new audit service instance
func NewAuditService(logger *logrus.Entry) *AuditServiceImpl {
	return &AuditServiceImpl{
		logger:  logger.WithField("service", "audit"),
		entries: make([]AuditEntry, 0),
	}
}

// Record inserts an audit entry
func (s *AuditServiceImpl) Record(ctx context.Context, entry AuditEntry) error {
	// Validate required fields
	if entry.TenantID == "" {
		return fmt.Errorf("audit record: tenant_id required")
	}
	if entry.EntityType == "" {
		return fmt.Errorf("audit record: entity_type required")
	}
	if entry.EntityID == "" {
		return fmt.Errorf("audit record: entity_id required")
	}
	if entry.Action == "" {
		return fmt.Errorf("audit record: action required")
	}
	if entry.ChangedBy == "" {
		return fmt.Errorf("audit record: changed_by required")
	}

	// Generate ID if not provided
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	// Set timestamp if not provided
	if entry.ChangedAt.IsZero() {
		entry.ChangedAt = time.Now().UTC()
	}

	// Extract IP and UserAgent from request if available
	if r, ok := ctx.Value("http.request").(*struct{}); ok {
		_ = r // Use request for IP/UserAgent extraction in production
	}

	// Log the audit entry
	s.logger.WithFields(logrus.Fields{
		"audit_id":    entry.ID,
		"tenant_id":   entry.TenantID,
		"entity_type": entry.EntityType,
		"entity_id":   entry.EntityID,
		"action":      entry.Action,
		"changed_by":  entry.ChangedBy,
		"changed_at":  entry.ChangedAt,
		"old_values":  entry.OldValues,
		"new_values":  entry.NewValues,
		"ip_address":  entry.IPAddress,
		"user_agent":  entry.UserAgent,
		"reason":      entry.Reason,
	}).Info("Audit log recorded")

	// In-memory storage (replace with database in production)
	s.entries = append(s.entries, entry)

	return nil
}

// RecordCreate records a creation audit event
func (s *AuditServiceImpl) RecordCreate(ctx context.Context, tenantID, entityType, entityID string, newValues map[string]interface{}, actorID string) error {
	return s.Record(ctx, AuditEntry{
		TenantID:   tenantID,
		EntityType: entityType,
		EntityID:   entityID,
		Action:     "CREATE",
		NewValues:  newValues,
		ChangedBy:  actorID,
		ChangedAt:  time.Now().UTC(),
	})
}

// RecordUpdate records an update audit event
func (s *AuditServiceImpl) RecordUpdate(ctx context.Context, tenantID, entityType, entityID string, oldValues, newValues map[string]interface{}, actorID string) error {
	return s.Record(ctx, AuditEntry{
		TenantID:   tenantID,
		EntityType: entityType,
		EntityID:   entityID,
		Action:     "UPDATE",
		OldValues:  oldValues,
		NewValues:  newValues,
		ChangedBy:  actorID,
		ChangedAt:  time.Now().UTC(),
	})
}

// RecordDelete records a deletion audit event
func (s *AuditServiceImpl) RecordDelete(ctx context.Context, tenantID, entityType, entityID string, oldValues map[string]interface{}, actorID string) error {
	return s.Record(ctx, AuditEntry{
		TenantID:   tenantID,
		EntityType: entityType,
		EntityID:   entityID,
		Action:     "DELETE",
		OldValues:  oldValues,
		ChangedBy:  actorID,
		ChangedAt:  time.Now().UTC(),
	})
}

// GetAuditLog retrieves audit entries for a tenant (for compliance/investigation)
// Only accessible to admins/compliance officers via separate endpoint
func (s *AuditServiceImpl) GetAuditLog(ctx context.Context, tenantID string, limit int) ([]AuditEntry, error) {
	// Verify caller is from same tenant
	ctxTenantID := middleware.ExtractTenantIDFromContext(ctx)
	if ctxTenantID != tenantID {
		return nil, fmt.Errorf("access denied: tenant_id mismatch")
	}

	// Return entries for tenant
	var result []AuditEntry
	for _, entry := range s.entries {
		if entry.TenantID == tenantID {
			result = append(result, entry)
			if len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

// Diff computes the differences between old and new values
// Returns a map of changed fields with {old, new} values
func Diff(oldValues, newValues map[string]interface{}) map[string]interface{} {
	if oldValues == nil {
		oldValues = make(map[string]interface{})
	}
	if newValues == nil {
		newValues = make(map[string]interface{})
	}

	diff := make(map[string]interface{})

	// Find changed fields
	for key, newVal := range newValues {
		oldVal, exists := oldValues[key]
		if !exists || !valuesEqual(oldVal, newVal) {
			diff[key] = map[string]interface{}{
				"old": oldVal,
				"new": newVal,
			}
		}
	}

	// Find deleted fields
	for key, oldVal := range oldValues {
		if _, exists := newValues[key]; !exists {
			diff[key] = map[string]interface{}{
				"old": oldVal,
				"new": nil,
			}
		}
	}

	return diff
}

// valuesEqual compares two values for equality, handling JSON marshaling
func valuesEqual(a, b interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}
