package services

import (
	"context"
	"fmt"
	"log"

	"github.com/hondyman/semlayer/backend/internal/metadata"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/security"
)

// ============================================================================
// BO MICROSERVICE COMMAND HANDLER
// ============================================================================
// This service handles commands for Business Objects and publishes events
// It implements the command handler pattern where each command type gets
// a dedicated handler that executes business logic and publishes events.
//
// This is the core of the business logic layer that would eventually run
// in a separate microservice container.
// ============================================================================

// BOCommandHandler handles Business Object commands
type BOCommandHandler struct {
	boService      *metadata.BusinessObjectService
	eventPublisher *EventPublisher
}

// NewBOCommandHandler creates a new BO command handler
func NewBOCommandHandler(
	boService *metadata.BusinessObjectService,
	eventPublisher *EventPublisher,
) *BOCommandHandler {
	return &BOCommandHandler{
		boService:      boService,
		eventPublisher: eventPublisher,
	}
}

// HandleCreateBO handles the CreateBO command
func (bch *BOCommandHandler) HandleCreateBO(ctx context.Context, command *Command) (*CommandResponse, error) {
	log.Printf("⚙️  Handling CreateBO command: %s", command.ID)

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

	// Convert to CreateBusinessObjectRequest
	req := models.CreateBusinessObjectRequest{
		Name:         getStringField(reqMap, "name"),
		DisplayName:  getStringField(reqMap, "displayName"),
		Description:  getStringField(reqMap, "description"),
		Icon:         getStringField(reqMap, "icon"),
		Category:     getStringField(reqMap, "category"),
		CloneFromKey: getStringField(reqMap, "cloneFromKey"),
	}

	// Execute business logic
	secCtx := &security.Context{TenantID: command.TenantID}
	bo, err := bch.boService.CreateBusinessObject(ctx, secCtx, req, command.UserID)
	if err != nil {
		log.Printf("❌ Failed to create BO: %v", err)
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Error:         err.Error(),
		}, nil
	}

	// Publish event
	if bch.eventPublisher != nil {
		bch.eventPublisher.PublishBOCreated(ctx, bo, command.UserID)
	}

	log.Printf("✅ BO created: %s", bo.Key)

	return &CommandResponse{
		ID:            command.ID,
		CorrelationID: command.CorrelationID,
		Status:        CommandStatusSuccess,
		Message:       fmt.Sprintf("BO created: %s", bo.Key),
		Data:          bo,
	}, nil
}

// HandleUpdateBO handles the UpdateBO command
func (bch *BOCommandHandler) HandleUpdateBO(ctx context.Context, command *Command) (*CommandResponse, error) {
	log.Printf("⚙️  Handling UpdateBO command: %s", command.ID)

	// Extract request from command data
	cmdMap, ok := command.Data.(map[string]interface{})
	if !ok {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Error:         "Invalid command data format",
		}, nil
	}

	key := getStringField(cmdMap, "key")
	if key == "" {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Error:         "Missing key in command",
		}, nil
	}

	dataMap := getMapField(cmdMap, "data")
	req := models.UpdateBusinessObjectRequest{
		DisplayName: getStringField(dataMap, "displayName"),
		Description: getStringField(dataMap, "description"),
		Icon:        getStringField(dataMap, "icon"),
		Category:    getStringField(dataMap, "category"),
	}

	// Execute business logic
	secCtx := &security.Context{TenantID: command.TenantID}
	bo, err := bch.boService.UpdateBusinessObject(ctx, secCtx, key, req, command.UserID)
	if err != nil {
		log.Printf("❌ Failed to update BO: %v", err)
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Error:         err.Error(),
		}, nil
	}

	// Publish event
	if bch.eventPublisher != nil {
		bch.eventPublisher.PublishBOUpdated(ctx, bo, command.UserID)
	}

	log.Printf("✅ BO updated: %s", bo.Key)

	return &CommandResponse{
		ID:            command.ID,
		CorrelationID: command.CorrelationID,
		Status:        CommandStatusSuccess,
		Message:       fmt.Sprintf("BO updated: %s", bo.Key),
		Data:          bo,
	}, nil
}

// HandleDeleteBO handles the DeleteBO command
func (bch *BOCommandHandler) HandleDeleteBO(ctx context.Context, command *Command) (*CommandResponse, error) {
	log.Printf("⚙️  Handling DeleteBO command: %s", command.ID)

	// Extract request from command data
	cmdMap, ok := command.Data.(map[string]interface{})
	if !ok {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Error:         "Invalid command data format",
		}, nil
	}

	key := getStringField(cmdMap, "key")
	if key == "" {
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Error:         "Missing key in command",
		}, nil
	}

	// Execute business logic
	secCtx := &security.Context{TenantID: command.TenantID}
	err := bch.boService.DeleteBusinessObject(ctx, secCtx, key, command.UserID)
	if err != nil {
		log.Printf("❌ Failed to delete BO: %v", err)
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Error:         err.Error(),
		}, nil
	}

	// Publish event
	if bch.eventPublisher != nil {
		bch.eventPublisher.PublishBODeleted(ctx, command.TenantID, key, command.UserID)
	}

	log.Printf("✅ BO deleted: %s", key)

	return &CommandResponse{
		ID:            command.ID,
		CorrelationID: command.CorrelationID,
		Status:        CommandStatusSuccess,
		Message:       fmt.Sprintf("BO deleted: %s", key),
	}, nil
}

// HandleCloneBO handles the CloneBO command
func (bch *BOCommandHandler) HandleCloneBO(ctx context.Context, command *Command) (*CommandResponse, error) {
	log.Printf("⚙️  Handling CloneBO command: %s", command.ID)

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

	req := models.CloneBORequest{
		SourceBOKey: getStringField(reqMap, "sourceBOKey"),
		NewName:     getStringField(reqMap, "newName"),
		Description: getStringField(reqMap, "description"),
		Icon:        getStringField(reqMap, "icon"),
	}

	// Execute business logic
	secCtx := &security.Context{TenantID: command.TenantID}
	bo, err := bch.boService.CloneBusinessObject(ctx, secCtx, req, command.UserID)
	if err != nil {
		log.Printf("❌ Failed to clone BO: %v", err)
		return &CommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        CommandStatusFailed,
			Error:         err.Error(),
		}, nil
	}

	// Publish event
	if bch.eventPublisher != nil {
		bch.eventPublisher.PublishBOCloned(ctx, bo, req.SourceBOKey, command.UserID)
	}

	log.Printf("✅ BO cloned: %s -> %s", req.SourceBOKey, bo.Key)

	return &CommandResponse{
		ID:            command.ID,
		CorrelationID: command.CorrelationID,
		Status:        CommandStatusSuccess,
		Message:       fmt.Sprintf("BO cloned: %s", bo.Key),
		Data:          bo,
	}, nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func getStringField(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getMapField(m map[string]interface{}, key string) map[string]interface{} {
	if val, ok := m[key]; ok {
		if mapVal, ok := val.(map[string]interface{}); ok {
			return mapVal
		}
	}
	return make(map[string]interface{})
}
