package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"log"
	"github.com/hondyman/semlayer/backend/internal/audit"
)

// BitemporalCDCWorker processes CDC events and tracks them in the bitemporal audit system
type BitemporalCDCWorker struct {
	tracker *audit.BitemporalTracker
}

// NewBitemporalCDCWorker creates a new bitemporal CDC worker
func NewBitemporalCDCWorker(tracker *audit.BitemporalTracker) *BitemporalCDCWorker {
	return &BitemporalCDCWorker{
		tracker: tracker,
		
	}
}

// ProcessTenantChange processes a CDC event for the tenants table
func (w *BitemporalCDCWorker) ProcessTenantChange(ctx context.Context, op string, before, after map[string]interface{}) error {
	// Determine change type
	changeType := w.mapOperationToChangeType(op)
	if changeType == "" {
		return nil // Skip snapshot reads
	}

	// Use 'after' for INSERT/UPDATE, 'before' for DELETE
	entityData := after
	if op == "d" {
		entityData = before
	}

	if entityData == nil {
		return fmt.Errorf("no entity data available for operation %s", op)
	}

	// Extract tenant ID
	tenantID, err := w.extractID(entityData, "id")
	if err != nil {
		return fmt.Errorf("failed to extract tenant ID: %w", err)
	}

	// Track the change
	return w.tracker.TrackEntityChange(ctx, audit.EntityChange{
		EntityType:   "tenant",
		EntityID:     tenantID,
		ChangeType:   changeType,
		ValidFrom:    time.Now(),
		EntityData:   entityData,
		ChangedBy:    "system-cdc",
		ChangeReason: fmt.Sprintf("CDC event: %s", op),
	})
}

// ProcessInstanceChange processes a CDC event for the tenant_instance table
func (w *BitemporalCDCWorker) ProcessInstanceChange(ctx context.Context, op string, before, after map[string]interface{}) error {
	changeType := w.mapOperationToChangeType(op)
	if changeType == "" {
		return nil
	}

	entityData := after
	if op == "d" {
		entityData = before
	}

	if entityData == nil {
		return fmt.Errorf("no entity data available for operation %s", op)
	}

	instanceID, err := w.extractID(entityData, "id")
	if err != nil {
		return fmt.Errorf("failed to extract instance ID: %w", err)
	}

	return w.tracker.TrackEntityChange(ctx, audit.EntityChange{
		EntityType:   "instance",
		EntityID:     instanceID,
		ChangeType:   changeType,
		ValidFrom:    time.Now(),
		EntityData:   entityData,
		ChangedBy:    "system-cdc",
		ChangeReason: fmt.Sprintf("CDC event: %s", op),
	})
}

// ProcessConnectionChange processes a CDC event for the connections table
func (w *BitemporalCDCWorker) ProcessConnectionChange(ctx context.Context, op string, before, after map[string]interface{}) error {
	changeType := w.mapOperationToChangeType(op)
	if changeType == "" {
		return nil
	}

	entityData := after
	if op == "d" {
		entityData = before
	}

	if entityData == nil {
		return fmt.Errorf("no entity data available for operation %s", op)
	}

	connectionID, err := w.extractID(entityData, "id")
	if err != nil {
		return fmt.Errorf("failed to extract connection ID: %w", err)
	}

	// Sanitize sensitive data before storing
	sanitizedData := w.sanitizeConnectionData(entityData)

	return w.tracker.TrackEntityChange(ctx, audit.EntityChange{
		EntityType:   "connection",
		EntityID:     connectionID,
		ChangeType:   changeType,
		ValidFrom:    time.Now(),
		EntityData:   sanitizedData,
		ChangedBy:    "system-cdc",
		ChangeReason: fmt.Sprintf("CDC event: %s", op),
	})
}

// ProcessProductChange processes a CDC event for the tenant_product table
func (w *BitemporalCDCWorker) ProcessProductChange(ctx context.Context, op string, before, after map[string]interface{}) error {
	changeType := w.mapOperationToChangeType(op)
	if changeType == "" {
		return nil
	}

	entityData := after
	if op == "d" {
		entityData = before
	}

	if entityData == nil {
		return fmt.Errorf("no entity data available for operation %s", op)
	}

	productID, err := w.extractID(entityData, "id")
	if err != nil {
		return fmt.Errorf("failed to extract product ID: %w", err)
	}

	return w.tracker.TrackEntityChange(ctx, audit.EntityChange{
		EntityType:   "product",
		EntityID:     productID,
		ChangeType:   changeType,
		ValidFrom:    time.Now(),
		EntityData:   entityData,
		ChangedBy:    "system-cdc",
		ChangeReason: fmt.Sprintf("CDC event: %s", op),
	})
}

// Helper methods

func (w *BitemporalCDCWorker) mapOperationToChangeType(op string) string {
	switch op {
	case "c":
		return "INSERT"
	case "u":
		return "UPDATE"
	case "d":
		return "DELETE"
	case "r":
		// Skip snapshot reads - we only track actual changes
		return ""
	default:
		log.Printf("Unknown CDC operation: %s", op)
		return ""
	}
}

func (w *BitemporalCDCWorker) extractID(data map[string]interface{}, field string) (string, error) {
	idValue, ok := data[field]
	if !ok {
		return "", fmt.Errorf("field %s not found in data", field)
	}

	// Handle different ID formats
	switch v := idValue.(type) {
	case string:
		// Validate UUID format
		if _, err := uuid.Parse(v); err != nil {
			return "", fmt.Errorf("invalid UUID format: %w", err)
		}
		return v, nil
	case []byte:
		return string(v), nil
	default:
		// Try to marshal and unmarshal as string
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal ID: %w", err)
		}
		var idStr string
		if err := json.Unmarshal(jsonBytes, &idStr); err != nil {
			return "", fmt.Errorf("failed to unmarshal ID: %w", err)
		}
		return idStr, nil
	}
}

func (w *BitemporalCDCWorker) sanitizeConnectionData(data map[string]interface{}) map[string]interface{} {
	// Create a copy to avoid modifying the original
	sanitized := make(map[string]interface{})
	for k, v := range data {
		sanitized[k] = v
	}

	// Remove sensitive fields
	sensitiveFields := []string{"password", "api_key", "username"}
	for _, field := range sensitiveFields {
		if _, exists := sanitized[field]; exists {
			sanitized[field] = "***REDACTED***"
		}
	}

	return sanitized
}
