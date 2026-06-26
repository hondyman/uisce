package handlers

import (
	"context"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/services"
	"go.uber.org/zap"
)

// ValidationHandler manages business process validation within request flows
type ValidationHandler struct {
	bpCoordinator  services.BPValidationCoordinator
	asyncValidator services.AsyncValidator
	logger         *zap.Logger
}

// NewValidationHandler creates a new validation handler
func NewValidationHandler(
	bpCoordinator services.BPValidationCoordinator,
	asyncValidator services.AsyncValidator,
	logger *zap.Logger,
) *ValidationHandler {
	return &ValidationHandler{
		bpCoordinator:  bpCoordinator,
		asyncValidator: asyncValidator,
		logger:         logger,
	}
}

// ValidateBPStep synchronously validates a business process step
func (vh *ValidationHandler) ValidateBPStep(
	ctx context.Context,
	tenantID, userID, bpName, stepName string,
	formData map[string]interface{},
) (*services.BPValidationResponse, error) {
	if vh.bpCoordinator == nil {
		return nil, fmt.Errorf("bp validation coordinator not available")
	}

	// Build validation request
	req := &services.BPValidationRequest{
		TenantID:   tenantID,
		BPName:     bpName,
		StepName:   stepName,
		FormData:   formData,
		UserID:     userID,
		ReturnSync: true,
	}

	// Execute synchronous validation
	result, err := vh.bpCoordinator.ValidateBPStep(ctx, req)
	if err != nil {
		vh.logger.Error("BP step validation failed",
			zap.String("tenant_id", tenantID),
			zap.String("user_id", userID),
			zap.String("step_name", stepName),
			zap.Error(err),
		)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return result, nil
}

// QueueBPValidation queues a BP validation asynchronously
func (vh *ValidationHandler) QueueBPValidation(
	ctx context.Context,
	tenantID, userID, bpName, stepName string,
	formData map[string]interface{},
) (string, error) {
	if vh.bpCoordinator == nil {
		return "", fmt.Errorf("bp validation coordinator not available")
	}

	req := &services.BPValidationRequest{
		TenantID:   tenantID,
		BPName:     bpName,
		StepName:   stepName,
		FormData:   formData,
		UserID:     userID,
		ReturnSync: false,
	}

	// Queue async validation
	validationID, err := vh.bpCoordinator.QueueBPValidation(ctx, req)
	if err != nil {
		vh.logger.Error("Failed to queue BP validation",
			zap.String("tenant_id", tenantID),
			zap.String("step_name", stepName),
			zap.Error(err),
		)
		return "", fmt.Errorf("failed to queue validation: %w", err)
	}

	vh.logger.Info("BP validation queued",
		zap.String("tenant_id", tenantID),
		zap.String("validation_id", validationID),
		zap.String("step_name", stepName),
	)

	return validationID, nil
}

// GetValidationResult retrieves a BP validation result
func (vh *ValidationHandler) GetValidationResult(
	ctx context.Context,
	validationID string,
) (*services.BPValidationResponse, error) {
	if vh.bpCoordinator == nil {
		return nil, fmt.Errorf("bp validation coordinator not available")
	}

	result, err := vh.bpCoordinator.GetValidationResult(ctx, validationID)
	if err != nil {
		vh.logger.Error("Failed to get validation result",
			zap.String("validation_id", validationID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get validation result: %w", err)
	}

	return result, nil
}

// HandleValidationResponse processes BP validation response and logs results
func (vh *ValidationHandler) HandleValidationResponse(
	ctx context.Context,
	tenantID, userID string,
	result *services.BPValidationResponse,
) error {
	if result == nil {
		return fmt.Errorf("validation response is nil")
	}

	// Determine action based on result
	if result.Passed {
		vh.logger.Info("Validation passed",
			zap.String("tenant_id", tenantID),
			zap.String("validation_id", result.ID),
			zap.Int("warning_count", len(result.Warnings)),
		)
	} else {
		vh.logger.Warn("Validation failed",
			zap.String("tenant_id", tenantID),
			zap.String("validation_id", result.ID),
			zap.Strings("errors", result.Errors),
		)
	}

	// Record audit trail
	return vh.RecordAuditTrail(ctx, tenantID, userID, result)
}

// RecordAuditTrail logs validation execution for compliance and debugging
func (vh *ValidationHandler) RecordAuditTrail(
	ctx context.Context,
	tenantID, userID string,
	result *services.BPValidationResponse,
) error {
	auditData := map[string]interface{}{
		"passed":          result.Passed,
		"error_count":     len(result.Errors),
		"warning_count":   len(result.Warnings),
		"actions_to_take": result.ActionsToTake,
	}

	// Log audit entry (could also persist to database)
	vh.logger.Info("Validation audit trail",
		zap.String("tenant_id", tenantID),
		zap.String("user_id", userID),
		zap.String("validation_id", result.ID),
		zap.Any("audit_data", auditData),
	)

	return nil
}

// SubscribeToValidationEvents subscribes to BP validation events
func (vh *ValidationHandler) SubscribeToValidationEvents(
	ctx context.Context,
	bpName, stepName string,
) (<-chan *services.BPValidationResponse, error) {
	if vh.bpCoordinator == nil {
		return nil, fmt.Errorf("bp validation coordinator not available")
	}

	eventsChan, err := vh.bpCoordinator.SubscribeToValidationEvents(ctx, bpName, stepName)
	if err != nil {
		vh.logger.Error("Failed to subscribe to validation events",
			zap.String("bp_name", bpName),
			zap.String("step_name", stepName),
			zap.Error(err),
		)
		return nil, fmt.Errorf("subscription failed: %w", err)
	}

	return eventsChan, nil
}
