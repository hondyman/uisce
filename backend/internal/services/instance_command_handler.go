package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/metadata"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/security"
)

// InstanceCommandHandler handles all CRUD commands for Business Object Instances
// This follows the same pattern as BOCommandHandler for consistency and scalability
//
// Architecture:
// - Receives CRUD commands from the command bus (RabbitMQ)
// - Extracts business logic from HTTP handlers
// - Executes service layer operations (CreateInstance, UpdateInstance, DeleteInstance)
// - Publishes events to the event store for audit trail and event sourcing
// - Returns CommandResponse with status and result data
//
// This handler is registered with the CommandConsumer and routes:
// - command.instance.create -> HandleCreateInstance
// - command.instance.update -> HandleUpdateInstance
// - command.instance.delete -> HandleDeleteInstance
type InstanceCommandHandler struct {
	boService      *metadata.BusinessObjectService
	eventPublisher *EventPublisher
	audit          security.ImpersonationAuditLogger
	policy         security.ImpersonationPolicy
}

// NewInstanceCommandHandler creates a new instance command handler
func NewInstanceCommandHandler(
	boService *metadata.BusinessObjectService,
	eventPublisher *EventPublisher,
	audit security.ImpersonationAuditLogger,
	policy security.ImpersonationPolicy,
) *InstanceCommandHandler {
	return &InstanceCommandHandler{
		boService:      boService,
		eventPublisher: eventPublisher,
		audit:          audit,
		policy:         policy,
	}
}

// HandleCreateInstance handles command.instance.create commands
// Expected command.Data structure:
//
//	{
//	  "tenantID": "uuid",
//	  "userID": "user@company.com",
//	  "businessObjectKey": "Customer",
//	  "instance": {
//	    "businessObjectID": "uuid",
//	    "businessObjectKey": "Customer",
//	    "datasourceID": "uuid",
//	    "coreFieldValues": { "name": "John Doe", "email": "john@company.com" },
//	    "customFieldValues": { "department": "Sales" }
//	  }
//	}
func (ich *InstanceCommandHandler) HandleCreateInstance(ctx context.Context, command *Command) (*CommandResponse, error) {
	// Extract request from command data
	reqMap, ok := command.Data.(map[string]interface{})
	if !ok {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Error:         "Invalid command data format",
		}, nil
	}

	// Extract tenant, user, and instance data
	tenantID := getStringField(reqMap, "tenantID")
	userID := getStringField(reqMap, "userID")
	boKey := getStringField(reqMap, "businessObjectKey")

	if tenantID == "" || userID == "" || boKey == "" {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       "Missing required fields: tenantID, userID, businessObjectKey",
			Error:         "validation error",
			Timestamp:     command.Timestamp,
		}, nil
	}

	instanceData := getMapField(reqMap, "instance")
	if instanceData == nil {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       "Missing required field: instance",
			Error:         "validation error",
			Timestamp:     command.Timestamp,
		}, nil
	}

	// Build BusinessObjectInstance from command data
	instance := &models.BusinessObjectInstance{
		BusinessObjectKey: boKey,
		SubtypeKey:        getStringField(instanceData, "subtypeKey"),
	}

	// Extract optional fields if present
	if boID, ok := instanceData["businessObjectID"].(string); ok {
		instance.BusinessObjectID = boID
	}
	if dsID, ok := instanceData["datasourceID"].(string); ok {
		instance.DatasourceID = dsID
	}
	if subtypeID, ok := instanceData["subtypeID"].(string); ok && subtypeID != "" {
		instance.SubtypeID = sql.NullString{String: subtypeID, Valid: true}
	}

	// Extract field values
	if coreFields, ok := instanceData["coreFieldValues"].(map[string]interface{}); ok {
		instance.CoreFieldValues = coreFields
	}
	if customFields, ok := instanceData["customFieldValues"].(map[string]interface{}); ok {
		instance.CustomFieldValues = customFields
	}

	// Execute service layer operation inside a transaction so impersonation audit
	// is committed atomically with the BO mutation.
	db, err := ich.boService.TenantDB(tenantID)
	if err != nil {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       fmt.Sprintf("Failed to resolve tenant database: %v", err),
			Error:         err.Error(),
			Timestamp:     command.Timestamp,
		}, nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       fmt.Sprintf("Failed to begin transaction: %v", err),
			Error:         err.Error(),
			Timestamp:     command.Timestamp,
		}, nil
	}
	defer tx.Rollback()

	created, err := ich.boService.CreateInstanceTx(ctx, tx, tenantID, userID, instance)
	if err != nil {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       fmt.Sprintf("Failed to create instance: %v", err),
			Error:         err.Error(),
			Timestamp:     command.Timestamp,
		}, nil
	}

	// Synchronous impersonation action audit (abort on audit failure rolls back tx).
	if auditErr := ich.auditImpersonationActionTx(ctx, tx, boKey, created.ID, "CREATE", command.Data); auditErr != nil {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       fmt.Sprintf("Failed to audit impersonation action: %v", auditErr),
			Error:         auditErr.Error(),
			Timestamp:     command.Timestamp,
		}, nil
	}

	if err := tx.Commit(); err != nil {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       fmt.Sprintf("Failed to commit transaction: %v", err),
			Error:         err.Error(),
			Timestamp:     command.Timestamp,
		}, nil
	}

	// Publish event for audit trail and event sourcing
	if ich.eventPublisher != nil {
		ich.eventPublisher.PublishInstanceCreated(ctx, created, userID)
	}

	return &CommandResponse{
		ID:            command.ID,
		CorrelationID: command.CorrelationID,
		Status:        CommandStatusSuccess,
		Message:       fmt.Sprintf("Instance created successfully: %s", created.ID),
		Data: map[string]interface{}{
			"instance_id": created.ID,
			"bo_key":      created.BusinessObjectKey,
			"instance":    created,
		},
		Timestamp: command.Timestamp,
	}, nil
}

// HandleUpdateInstance handles command.instance.update commands
// Expected command.Data structure:
//
//	{
//	  "tenantID": "uuid",
//	  "userID": "user@company.com",
//	  "instanceID": "uuid",
//	  "coreFieldUpdates": { "name": "Jane Doe" },
//	  "customFieldUpdates": { "department": "Marketing" }
//	}
func (ich *InstanceCommandHandler) HandleUpdateInstance(ctx context.Context, command *Command) (*CommandResponse, error) {
	// Extract request from command data
	reqMap, ok := command.Data.(map[string]interface{})
	if !ok {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Error:         "Invalid command data format",
		}, nil
	}

	// Extract tenant, user, and instance data
	tenantID := getStringField(reqMap, "tenantID")
	userID := getStringField(reqMap, "userID")
	instanceID := getStringField(reqMap, "instanceID")

	if tenantID == "" || userID == "" || instanceID == "" {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       "Missing required fields: tenantID, userID, instanceID",
			Error:         "validation error",
			Timestamp:     command.Timestamp,
		}, nil
	}

	// Extract field updates
	coreUpdates := getMapField(reqMap, "coreFieldUpdates")
	customUpdates := getMapField(reqMap, "customFieldUpdates")

	// Execute service layer operation inside a transaction so impersonation audit
	// is committed atomically with the BO mutation.
	db, err := ich.boService.TenantDB(tenantID)
	if err != nil {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       fmt.Sprintf("Failed to resolve tenant database: %v", err),
			Error:         err.Error(),
			Timestamp:     command.Timestamp,
		}, nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       fmt.Sprintf("Failed to begin transaction: %v", err),
			Error:         err.Error(),
			Timestamp:     command.Timestamp,
		}, nil
	}
	defer tx.Rollback()

	updated, err := ich.boService.UpdateInstanceTx(ctx, tx, tenantID, instanceID, userID, coreUpdates, customUpdates)
	if err != nil {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       fmt.Sprintf("Failed to update instance: %v", err),
			Error:         err.Error(),
			Timestamp:     command.Timestamp,
		}, nil
	}

	// Synchronous impersonation action audit (abort on audit failure rolls back tx).
	if auditErr := ich.auditImpersonationActionTx(ctx, tx, updated.BusinessObjectKey, updated.ID, "UPDATE", command.Data); auditErr != nil {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       fmt.Sprintf("Failed to audit impersonation action: %v", auditErr),
			Error:         auditErr.Error(),
			Timestamp:     command.Timestamp,
		}, nil
	}

	if err := tx.Commit(); err != nil {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       fmt.Sprintf("Failed to commit transaction: %v", err),
			Error:         err.Error(),
			Timestamp:     command.Timestamp,
		}, nil
	}

	// Publish event for audit trail and event sourcing
	if ich.eventPublisher != nil {
		ich.eventPublisher.PublishInstanceUpdated(ctx, updated, userID)
	}

	return &CommandResponse{
		ID:            command.ID,
		CorrelationID: command.CorrelationID,
		Status:        CommandStatusSuccess,
		Message:       fmt.Sprintf("Instance updated successfully: %s", updated.ID),
		Data: map[string]interface{}{
			"instance_id": updated.ID,
			"instance":    updated,
		},
		Timestamp: command.Timestamp,
	}, nil
}

// HandleDeleteInstance handles command.instance.delete commands
// Expected command.Data structure:
//
//	{
//	  "tenantID": "uuid",
//	  "userID": "user@company.com",
//	  "instanceID": "uuid",
//	  "businessObjectKey": "Customer"
//	}
func (ich *InstanceCommandHandler) HandleDeleteInstance(ctx context.Context, command *Command) (*CommandResponse, error) {
	// Extract request from command data
	reqMap, ok := command.Data.(map[string]interface{})
	if !ok {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Error:         "Invalid command data format",
		}, nil
	}

	// Extract tenant, user, and instance data
	tenantID := getStringField(reqMap, "tenantID")
	userID := getStringField(reqMap, "userID")
	instanceID := getStringField(reqMap, "instanceID")
	boKey := getStringField(reqMap, "businessObjectKey")

	if tenantID == "" || userID == "" || instanceID == "" {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       "Missing required fields: tenantID, userID, instanceID",
			Error:         "validation error",
			Timestamp:     command.Timestamp,
		}, nil
	}

	// Execute service layer operation inside a transaction so impersonation audit
	// is committed atomically with the BO mutation.
	db, err := ich.boService.TenantDB(tenantID)
	if err != nil {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       fmt.Sprintf("Failed to resolve tenant database: %v", err),
			Error:         err.Error(),
			Timestamp:     command.Timestamp,
		}, nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       fmt.Sprintf("Failed to begin transaction: %v", err),
			Error:         err.Error(),
			Timestamp:     command.Timestamp,
		}, nil
	}
	defer tx.Rollback()

	if err := ich.boService.DeleteInstanceTx(ctx, tx, tenantID, instanceID, userID); err != nil {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       fmt.Sprintf("Failed to delete instance: %v", err),
			Error:         err.Error(),
			Timestamp:     command.Timestamp,
		}, nil
	}

	// Synchronous impersonation action audit (abort on audit failure rolls back tx).
	if auditErr := ich.auditImpersonationActionTx(ctx, tx, boKey, instanceID, "DELETE", command.Data); auditErr != nil {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       fmt.Sprintf("Failed to audit impersonation action: %v", auditErr),
			Error:         auditErr.Error(),
			Timestamp:     command.Timestamp,
		}, nil
	}

	if err := tx.Commit(); err != nil {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Message:       fmt.Sprintf("Failed to commit transaction: %v", err),
			Error:         err.Error(),
			Timestamp:     command.Timestamp,
		}, nil
	}

	// Publish event for audit trail and event sourcing
	if ich.eventPublisher != nil {
		ich.eventPublisher.PublishInstanceDeleted(ctx, tenantID, boKey, instanceID, userID)
	}

	return &CommandResponse{
		ID:            command.ID,
		CorrelationID: command.CorrelationID,
		Status:        CommandStatusSuccess,
		Message:       fmt.Sprintf("Instance deleted successfully: %s", instanceID),
		Data: map[string]interface{}{
			"instance_id": instanceID,
		},
		Timestamp: command.Timestamp,
	}, nil
}

// auditImpersonationActionTx writes a synchronous micro-audit record inside
// the active transaction when the request is executing inside an impersonation
// session. No-op for normal users.
func (ich *InstanceCommandHandler) auditImpersonationActionTx(
	ctx context.Context,
	tx *sql.Tx,
	boKey string,
	instanceID string,
	transition string,
	payload any,
) error {
	if ich.audit == nil {
		return nil
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal audit payload: %w", err)
	}

	return security.LogBOActionIfImpersonating(
		ctx,
		ich.policy,
		ich.audit,
		tx,
		boKey,
		instanceID,
		transition,
		payloadBytes,
	)
}
