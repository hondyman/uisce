package bo

import (
	"context"
	"database/sql"
	"fmt"

	"go.uber.org/zap"

	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/security"
)

// ============================================================================
// BO COMMAND HANDLER INTERFACES
// ============================================================================
//
// These interfaces allow the bo domain to interact with cross-cutting
// services (metadata, events, security) WITHOUT importing them directly.
// This preserves Cardinal Rule 3 (no cycles): internal/bo can ONLY depend
// on libs/* + stdlib + zap. Concrete implementations live in services/
// and other internal/ packages.
//
// The actual command handler implementations remain in services/ for now
// and are wired in via the BOCommandBus below.

// BusinessObjectServiceProvider defines the contract that any BO service
// implementation must satisfy. The services.BusinessObjectService satisfies
// this interface, allowing internal/bo to invoke operations without coupling.
type BusinessObjectServiceProvider interface {
	CreateBusinessObject(ctx context.Context, secCtx *security.Context, req models.CreateBusinessObjectRequest, userID string) (*models.BusinessObjectDefinition, error)
	UpdateBusinessObject(ctx context.Context, secCtx *security.Context, key string, req models.UpdateBusinessObjectRequest, userID string) (*models.BusinessObjectDefinition, error)
	DeleteBusinessObject(ctx context.Context, secCtx *security.Context, key, userID string) error
	CloneBusinessObject(ctx context.Context, secCtx *security.Context, req models.CloneBORequest, userID string) (*models.BusinessObjectDefinition, error)
}

// EventPublisherProvider is satisfied by services.EventPublisher. It is
// the bridge between internal/bo and the events layer.
type EventPublisherProvider interface {
	PublishBOCreated(ctx context.Context, bo *models.BusinessObjectDefinition, userID string)
	PublishBOUpdated(ctx context.Context, bo *models.BusinessObjectDefinition, userID string)
	PublishBODeleted(ctx context.Context, tenantID, key, userID string)
	PublishBOCloned(ctx context.Context, clonedBO *models.BusinessObjectDefinition, sourceKey, userID string)
}

// CommandData represents a generic command payload in the bo domain.
type CommandData interface{}

// BOCommand is a generic command envelope used by the bo domain.
type BOCommand struct {
	ID            string
	CorrelationID string
	TenantID      string
	UserID        string
	Data          CommandData
}

// BOCommandResponse is a generic response envelope used by the bo domain.
type BOCommandResponse struct {
	ID            string
	CorrelationID string
	Status        string
	Message       string
	Error         string
	Data          interface{}
}

// BOCommandStatus enumerates command lifecycle states.
const (
	BOCommandStatusSuccess = "success"
	BOCommandStatusFailed  = "failed"
)

// BOCommandBus wires together a business object service and event publisher
// to execute BO commands. This is the new home for command bus logic that
// was previously in services.bo_command_handler.go.
type BOCommandBus struct {
	boService      BusinessObjectServiceProvider
	eventPublisher EventPublisherProvider
	log            *zap.Logger
}

// NewBOCommandBus constructs a BOCommandBus.
func NewBOCommandBus(boService BusinessObjectServiceProvider, eventPublisher EventPublisherProvider, log *zap.Logger) *BOCommandBus {
	return &BOCommandBus{
		boService:      boService,
		eventPublisher: eventPublisher,
		log:            log,
	}
}

// NewBOCommandHandler is the legacy constructor that delegates to NewBOCommandBus.
// Kept for backward compatibility with cmd/bo-service/main.go.
func NewBOCommandHandler(boService BusinessObjectServiceProvider, eventPublisher EventPublisherProvider) *BOCommandBus {
	log := zap.NewNop()
	return NewBOCommandBus(boService, eventPublisher, log)
}

// HandleCreateBO processes a CreateBO command.
func (bch *BOCommandBus) HandleCreateBO(ctx context.Context, command *BOCommand) *BOCommandResponse {
	if bch.log != nil {
		bch.log.Info("Handling CreateBO command", zap.String("command_id", command.ID))
	}

	reqMap, ok := command.Data.(map[string]interface{})
	if !ok {
		return &BOCommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        BOCommandStatusFailed,
			Error:         "Invalid command data format",
		}
	}

	req := models.CreateBusinessObjectRequest{
		Name:         getStringField(reqMap, "name"),
		DisplayName:  getStringField(reqMap, "displayName"),
		Description:  getStringField(reqMap, "description"),
		Icon:         getStringField(reqMap, "icon"),
		Category:     getStringField(reqMap, "category"),
		CloneFromKey: getStringField(reqMap, "cloneFromKey"),
	}

	result, err := bch.boService.CreateBusinessObject(ctx, nil, req, command.UserID)
	if err != nil {
		if bch.log != nil {
			bch.log.Error("Failed to create BO", zap.Error(err))
		}
		return &BOCommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        BOCommandStatusFailed,
			Error:         err.Error(),
		}
	}

	if bch.eventPublisher != nil {
		bch.eventPublisher.PublishBOCreated(ctx, result, command.UserID)
	}

	if bch.log != nil {
		bch.log.Info("BO created", zap.String("key", result.Key))
	}

	return &BOCommandResponse{
		ID:            command.ID,
		CorrelationID: command.CorrelationID,
		Status:        BOCommandStatusSuccess,
		Message:       fmt.Sprintf("BO created: %s", result.Key),
		Data:          result,
	}
}

// HandleUpdateBO processes an UpdateBO command.
func (bch *BOCommandBus) HandleUpdateBO(ctx context.Context, command *BOCommand) *BOCommandResponse {
	if bch.log != nil {
		bch.log.Info("Handling UpdateBO command", zap.String("command_id", command.ID))
	}

	cmdMap, ok := command.Data.(map[string]interface{})
	if !ok {
		return &BOCommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        BOCommandStatusFailed,
			Error:         "Invalid command data format",
		}
	}

	key := getStringField(cmdMap, "key")
	if key == "" {
		return &BOCommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        BOCommandStatusFailed,
			Error:         "Missing key in command",
		}
	}

	dataMap := getMapField(cmdMap, "data")
	req := models.UpdateBusinessObjectRequest{
		DisplayName: getStringField(dataMap, "displayName"),
		Description: getStringField(dataMap, "description"),
		Icon:        getStringField(dataMap, "icon"),
		Category:    getStringField(dataMap, "category"),
	}

	result, err := bch.boService.UpdateBusinessObject(ctx, nil, key, req, command.UserID)
	if err != nil {
		if bch.log != nil {
			bch.log.Error("Failed to update BO", zap.Error(err))
		}
		return &BOCommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        BOCommandStatusFailed,
			Error:         err.Error(),
		}
	}

	if bch.eventPublisher != nil {
		bch.eventPublisher.PublishBOUpdated(ctx, result, command.UserID)
	}

	return &BOCommandResponse{
		ID:            command.ID,
		CorrelationID: command.CorrelationID,
		Status:        BOCommandStatusSuccess,
		Message:       fmt.Sprintf("BO updated: %s", result.Key),
		Data:          result,
	}
}

// HandleDeleteBO processes a DeleteBO command.
func (bch *BOCommandBus) HandleDeleteBO(ctx context.Context, command *BOCommand) *BOCommandResponse {
	if bch.log != nil {
		bch.log.Info("Handling DeleteBO command", zap.String("command_id", command.ID))
	}

	cmdMap, ok := command.Data.(map[string]interface{})
	if !ok {
		return &BOCommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        BOCommandStatusFailed,
			Error:         "Invalid command data format",
		}
	}

	key := getStringField(cmdMap, "key")
	if key == "" {
		return &BOCommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        BOCommandStatusFailed,
			Error:         "Missing key in command",
		}
	}

	err := bch.boService.DeleteBusinessObject(ctx, nil, key, command.UserID)
	if err != nil {
		if bch.log != nil {
			bch.log.Error("Failed to delete BO", zap.Error(err))
		}
		return &BOCommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        BOCommandStatusFailed,
			Error:         err.Error(),
		}
	}

	if bch.eventPublisher != nil {
		bch.eventPublisher.PublishBODeleted(ctx, command.TenantID, key, command.UserID)
	}

	return &BOCommandResponse{
		ID:            command.ID,
		CorrelationID: command.CorrelationID,
		Status:        BOCommandStatusSuccess,
		Message:       fmt.Sprintf("BO deleted: %s", key),
	}
}

// HandleCloneBO processes a CloneBO command.
func (bch *BOCommandBus) HandleCloneBO(ctx context.Context, command *BOCommand) *BOCommandResponse {
	if bch.log != nil {
		bch.log.Info("Handling CloneBO command", zap.String("command_id", command.ID))
	}

	reqMap, ok := command.Data.(map[string]interface{})
	if !ok {
		return &BOCommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        BOCommandStatusFailed,
			Error:         "Invalid command data format",
		}
	}

	req := models.CloneBORequest{
		SourceBOKey: getStringField(reqMap, "sourceBOKey"),
		NewName:     getStringField(reqMap, "newName"),
		Description: getStringField(reqMap, "description"),
		Icon:        getStringField(reqMap, "icon"),
	}

	result, err := bch.boService.CloneBusinessObject(ctx, nil, req, command.UserID)
	if err != nil {
		if bch.log != nil {
			bch.log.Error("Failed to clone BO", zap.Error(err))
		}
		return &BOCommandResponse{
			ID:            command.ID,
			CorrelationID: command.CorrelationID,
			Status:        BOCommandStatusFailed,
			Error:         err.Error(),
		}
	}

	if bch.eventPublisher != nil {
		bch.eventPublisher.PublishBOCloned(ctx, result, req.SourceBOKey, command.UserID)
	}

	return &BOCommandResponse{
		ID:            command.ID,
		CorrelationID: command.CorrelationID,
		Status:        BOCommandStatusSuccess,
		Message:       fmt.Sprintf("BO cloned: %s", result.Key),
		Data:          result,
	}
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

// Suppress unused import warnings for sql.DB (reserved for future use).
var _ = sql.DB{}