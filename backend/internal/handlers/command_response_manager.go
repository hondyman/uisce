package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"os"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/metadata"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/internal/services"
	kafka "github.com/segmentio/kafka-go"
)

// CommandResponseManager handles the asynchronous command responses
type CommandResponseManager struct {
	commandBus *services.CommandBus
	boService  *metadata.BusinessObjectService
	enabled    bool
}

// NewCommandResponseManager creates a new CommandResponseManager
func NewCommandResponseManager(
	commandBus *services.CommandBus,
	boService *metadata.BusinessObjectService,
) *CommandResponseManager {
	manager := &CommandResponseManager{
		commandBus: commandBus,
		boService:  boService,
	}

	// Enable command bus if available
	if commandBus != nil && commandBus.IsEnabled() {
		manager.enabled = true
	}

	return manager
}

// waitForCommandResponse waits for a command response with timeout
func (m *CommandResponseManager) waitForCommandResponse(ctx context.Context, correlationID string, timeout time.Duration) (*services.CommandResponse, error) {
	if !m.enabled {
		return nil, fmt.Errorf("command bus not available")
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	if len(brokers) == 0 || brokers[0] == "" {
		brokers = []string{"localhost:9092"}
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     fmt.Sprintf("cmd-response-%s", correlationID),
		Topic:       "semlayer.replies",
		MinBytes:    10e3,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset,
	})
	defer r.Close()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("command response timeout")
		default:
			mmsg, err := r.FetchMessage(ctx)
			if err != nil {
				continue
			}
			if string(mmsg.Key) != correlationID {
				r.CommitMessages(ctx, mmsg)
				continue
			}
			var response services.CommandResponse
			if err := json.Unmarshal(mmsg.Value, &response); err != nil {
				return nil, fmt.Errorf("failed to unmarshal response: %w", err)
			}
			r.CommitMessages(ctx, mmsg)
			return &response, nil
		}
	}
}

// ExecuteCreateBO executes a create BO command
func (m *CommandResponseManager) ExecuteCreateBO(ctx context.Context, tenantID, userID string, req models.CreateBusinessObjectRequest) (interface{}, error) {
	// If command bus disabled, use direct service call
	if !m.enabled {
		return m.directCreateBO(ctx, tenantID, userID, req)
	}

	// Publish command
	correlationID, err := m.commandBus.PublishCommand(
		ctx,
		services.CommandCreateBO,
		tenantID,
		userID,
		req,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to publish command: %w", err)
	}

	// Wait for response
	response, err := m.waitForCommandResponse(ctx, correlationID, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}

	if response.Status != services.CommandStatusSuccess {
		return nil, errors.New(response.Error)
	}

	return response.Data, nil
}

// ExecuteUpdateBO executes an update BO command
func (m *CommandResponseManager) ExecuteUpdateBO(ctx context.Context, tenantID, userID, key string, req models.UpdateBusinessObjectRequest) (interface{}, error) {
	// If command bus disabled, use direct service call
	if !m.enabled {
		return m.directUpdateBO(ctx, tenantID, userID, key, req)
	}

	updateCmd := map[string]interface{}{
		"key":  key,
		"data": req,
	}

	correlationID, err := m.commandBus.PublishCommand(
		ctx,
		services.CommandUpdateBO,
		tenantID,
		userID,
		updateCmd,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to publish command: %w", err)
	}

	response, err := m.waitForCommandResponse(ctx, correlationID, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}

	if response.Status != services.CommandStatusSuccess {
		return nil, errors.New(response.Error)
	}

	return response.Data, nil
}

// ExecuteDeleteBO executes a delete BO command
func (m *CommandResponseManager) ExecuteDeleteBO(ctx context.Context, tenantID, userID, key string) error {
	// If command bus disabled, use direct service call
	if !m.enabled {
		return m.directDeleteBO(ctx, tenantID, userID, key)
	}

	deleteCmd := map[string]interface{}{
		"key": key,
	}

	correlationID, err := m.commandBus.PublishCommand(
		ctx,
		services.CommandDeleteBO,
		tenantID,
		userID,
		deleteCmd,
	)
	if err != nil {
		return fmt.Errorf("failed to publish command: %w", err)
	}

	response, err := m.waitForCommandResponse(ctx, correlationID, 10*time.Second)
	if err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	if response.Status != services.CommandStatusSuccess {
		return errors.New(response.Error)
	}

	return nil
}

// ExecuteCloneBO executes a clone BO command
func (m *CommandResponseManager) ExecuteCloneBO(ctx context.Context, tenantID, userID string, req models.CloneBORequest) (interface{}, error) {
	// If command bus disabled, use direct service call
	if !m.enabled {
		return m.directCloneBO(ctx, tenantID, userID, req)
	}

	correlationID, err := m.commandBus.PublishCommand(
		ctx,
		services.CommandCloneBO,
		tenantID,
		userID,
		req,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to publish command: %w", err)
	}

	response, err := m.waitForCommandResponse(ctx, correlationID, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}

	if response.Status != services.CommandStatusSuccess {
		return nil, errors.New(response.Error)
	}

	return response.Data, nil
}

// ExecuteCreateInstance executes a create instance command
func (m *CommandResponseManager) ExecuteCreateInstance(ctx context.Context, tenantID, userID string, instance *models.BusinessObjectInstance) (interface{}, error) {
	// If command bus disabled, use direct service call
	if !m.enabled {
		return m.directCreateInstance(ctx, tenantID, userID, instance)
	}

	commandData := map[string]interface{}{
		"tenantID":          tenantID,
		"userID":            userID,
		"businessObjectKey": instance.BusinessObjectKey,
		"instance": map[string]interface{}{
			"businessObjectID":  instance.BusinessObjectID,
			"businessObjectKey": instance.BusinessObjectKey,
			"datasourceID":      instance.DatasourceID,
			"subtypeID":         instance.SubtypeID.String,
			"subtypeKey":        instance.SubtypeKey,
			"coreFieldValues":   instance.CoreFieldValues,
			"customFieldValues": instance.CustomFieldValues,
		},
	}

	correlationID, err := m.commandBus.PublishCommand(ctx, services.CommandCreateInstance, tenantID, userID, commandData)
	if err != nil {
		return nil, fmt.Errorf("failed to publish command: %w", err)
	}

	response, err := m.waitForCommandResponse(ctx, correlationID, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}

	if response.Status != services.CommandStatusSuccess {
		return nil, errors.New(response.Message)
	}

	return response.Data, nil
}

// ExecuteUpdateInstance executes an update instance command
func (m *CommandResponseManager) ExecuteUpdateInstance(ctx context.Context, tenantID, userID, instanceID string, coreFields, customFields map[string]interface{}) (interface{}, error) {
	// If command bus disabled, use direct service call
	if !m.enabled {
		return m.directUpdateInstance(ctx, tenantID, userID, instanceID, coreFields, customFields)
	}

	commandData := map[string]interface{}{
		"tenantID":           tenantID,
		"userID":             userID,
		"instanceID":         instanceID,
		"coreFieldUpdates":   coreFields,
		"customFieldUpdates": customFields,
	}

	correlationID, err := m.commandBus.PublishCommand(ctx, services.CommandUpdateInstance, tenantID, userID, commandData)
	if err != nil {
		return nil, fmt.Errorf("failed to publish command: %w", err)
	}

	response, err := m.waitForCommandResponse(ctx, correlationID, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}

	if response.Status != services.CommandStatusSuccess {
		return nil, errors.New(response.Message)
	}

	return response.Data, nil
}

// ExecuteDeleteInstance executes a delete instance command
func (m *CommandResponseManager) ExecuteDeleteInstance(ctx context.Context, tenantID, userID, instanceID, boKey string) error {
	// If command bus disabled, use direct service call
	if !m.enabled {
		return m.directDeleteInstance(ctx, tenantID, userID, instanceID, boKey)
	}

	commandData := map[string]interface{}{
		"tenantID":          tenantID,
		"userID":            userID,
		"instanceID":        instanceID,
		"businessObjectKey": boKey,
	}

	correlationID, err := m.commandBus.PublishCommand(ctx, services.CommandDeleteInstance, tenantID, userID, commandData)
	if err != nil {
		return fmt.Errorf("failed to publish command: %w", err)
	}

	response, err := m.waitForCommandResponse(ctx, correlationID, 10*time.Second)
	if err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	if response.Status != services.CommandStatusSuccess {
		return errors.New(response.Message)
	}

	return nil
}

// ============================================================================
// DIRECT SERVICE CALL FALLBACKS
// ============================================================================

func (m *CommandResponseManager) directCreateBO(ctx context.Context, tenantID, userID string, req models.CreateBusinessObjectRequest) (interface{}, error) {
	secCtx := &security.Context{TenantID: tenantID}
	return m.boService.CreateBusinessObject(ctx, secCtx, req, userID)
}

func (m *CommandResponseManager) directUpdateBO(ctx context.Context, tenantID, userID, key string, req models.UpdateBusinessObjectRequest) (interface{}, error) {
	secCtx := &security.Context{TenantID: tenantID}
	return m.boService.UpdateBusinessObject(ctx, secCtx, key, req, userID)
}

func (m *CommandResponseManager) directDeleteBO(ctx context.Context, tenantID, userID, key string) error {
	secCtx := &security.Context{TenantID: tenantID}
	return m.boService.DeleteBusinessObject(ctx, secCtx, key, userID)
}

func (m *CommandResponseManager) directCloneBO(ctx context.Context, tenantID, userID string, req models.CloneBORequest) (interface{}, error) {
	secCtx := &security.Context{TenantID: tenantID}
	return m.boService.CloneBusinessObject(ctx, secCtx, req, userID)
}

func (m *CommandResponseManager) directCreateInstance(ctx context.Context, tenantID, userID string, instance *models.BusinessObjectInstance) (interface{}, error) {
	return m.boService.CreateInstance(ctx, tenantID, userID, instance)
}

func (m *CommandResponseManager) directUpdateInstance(ctx context.Context, tenantID, userID, instanceID string, coreFields, customFields map[string]interface{}) (interface{}, error) {
	return m.boService.UpdateInstance(ctx, tenantID, instanceID, userID, coreFields, customFields)
}

func (m *CommandResponseManager) directDeleteInstance(ctx context.Context, tenantID, userID, instanceID, boKey string) error {
	return m.boService.DeleteInstance(ctx, tenantID, instanceID, userID)
}
