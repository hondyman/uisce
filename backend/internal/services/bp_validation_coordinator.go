package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	kafka "github.com/segmentio/kafka-go"
)

// ============================================================================
// BUSINESS PROCESS VALIDATION COORDINATOR
// ============================================================================
// Orchestrates rule engine + async validator for complete BP validation workflow
// Integrates with Workday-like low-code designer patterns

// BPValidationRequest represents a BP validation request
type BPValidationRequest struct {
	TenantID   string                 `json:"tenant_id"`
	BPName     string                 `json:"bp_name"`
	StepName   string                 `json:"step_name"`
	FormData   map[string]interface{} `json:"form_data"`
	UserID     string                 `json:"user_id"`
	ContextID  string                 `json:"context_id"`  // For tracking through workflow
	ReturnSync bool                   `json:"return_sync"` // If true, wait for result; else async
}

// BPValidationResponse holds validation outcome
type BPValidationResponse struct {
	ID            string                 `json:"id"`
	Passed        bool                   `json:"passed"`
	Errors        []string               `json:"errors"`
	Warnings      []string               `json:"warnings"`
	ActionsToTake []string               `json:"actions_to_take"`
	Details       map[string]interface{} `json:"details"`
	Timestamp     time.Time              `json:"timestamp"`
}

// BPValidationCoordinator orchestrates BP validation
type BPValidationCoordinator interface {
	// Validate a BP step (synchronous evaluation + async routing)
	ValidateBPStep(ctx context.Context, req *BPValidationRequest) (*BPValidationResponse, error)

	// Queue async validation with result callback
	QueueBPValidation(ctx context.Context, req *BPValidationRequest) (string, error)

	// Get validation result
	GetValidationResult(ctx context.Context, validationID string) (*BPValidationResponse, error)

	// Subscribe to validation events
	SubscribeToValidationEvents(ctx context.Context, bpName string, stepName string) (<-chan *BPValidationResponse, error)
}

// BPValidationCoordinatorImpl implements BPValidationCoordinator
type BPValidationCoordinatorImpl struct {
	db             *sqlx.DB
	ruleEngine     ValidationRuleEngine
	asyncValidator AsyncValidator
	writer         *kafka.Writer
	results        map[string]*BPValidationResponse
	eventChannels  map[string]chan *BPValidationResponse
}

// NewBPValidationCoordinator creates a new BP validation coordinator
func NewBPValidationCoordinator(
	db *sqlx.DB,
	ruleEngine ValidationRuleEngine,
	asyncValidator AsyncValidator,
	writer *kafka.Writer,
) BPValidationCoordinator {
	return &BPValidationCoordinatorImpl{
		db:             db,
		ruleEngine:     ruleEngine,
		asyncValidator: asyncValidator,
		writer:         writer,
		results:        make(map[string]*BPValidationResponse),
		eventChannels:  make(map[string]chan *BPValidationResponse),
	}
}

// ============================================================================
// Synchronous BP Validation
// ============================================================================

// ValidateBPStep validates a business process step synchronously
func (bvc *BPValidationCoordinatorImpl) ValidateBPStep(ctx context.Context, req *BPValidationRequest) (*BPValidationResponse, error) {
	log.Printf("[BPCoordinator] Validating BP step: %s/%s for tenant %s", req.BPName, req.StepName, req.TenantID)

	startTime := time.Now()

	// Evaluate all rules for this BP step
	ruleResults, err := bvc.ruleEngine.EvaluateBPStep(ctx, req.TenantID, req.BPName, req.StepName, req.FormData)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate BP step: %w", err)
	}

	// Build validation response
	response := &BPValidationResponse{
		ID:            fmt.Sprintf("val_%d", time.Now().UnixNano()),
		Passed:        true,
		Errors:        []string{},
		Warnings:      []string{},
		ActionsToTake: []string{},
		Details: map[string]interface{}{
			"bp_name":       req.BPName,
			"step_name":     req.StepName,
			"rule_count":    len(ruleResults),
			"evaluation_ms": time.Since(startTime).Milliseconds(),
		},
		Timestamp: time.Now(),
	}

	// Process rule results
	for _, ruleResult := range ruleResults {
		if !ruleResult.Passed {
			response.Passed = false
			response.Errors = append(response.Errors, ruleResult.ErrorMessage)
		}

		if ruleResult.ActionToTake != "" {
			response.ActionsToTake = append(response.ActionsToTake, ruleResult.ActionToTake)
		}
	}

	// Route actions (send to RabbitMQ or trigger workflows)
	err = bvc.routeActions(ctx, response, req)
	if err != nil {
		log.Printf("[BPCoordinator] Warning: error routing actions: %v", err)
	}

	// Audit the validation
	bvc.auditValidation(ctx, req, response)

	return response, nil
}

// ============================================================================
// Asynchronous BP Validation
// ============================================================================

// QueueBPValidation queues a validation for async processing
func (bvc *BPValidationCoordinatorImpl) QueueBPValidation(ctx context.Context, req *BPValidationRequest) (string, error) {
	log.Printf("[BPCoordinator] Queueing BP validation async: %s/%s", req.BPName, req.StepName)

	validationID := fmt.Sprintf("bpval_%d", time.Now().UnixNano())

	// Create async validation task
	task := &ValidationTask{
		ID:         validationID,
		EntityID:   fmt.Sprintf("%s.%s", req.BPName, req.StepName),
		EntityType: "BP_VALIDATION",
		TenantID:   req.TenantID,
		EntityData: req.FormData,
		CreatedAt:  time.Now(),
		Status:     "pending",
	}

	// Submit to async validator
	err := bvc.asyncValidator.SubmitValidationTask(ctx, task)
	if err != nil {
		return "", fmt.Errorf("failed to queue validation: %w", err)
	}

	// Emit event for tracking
	bvc.publishValidationQueued(ctx, req, validationID)

	return validationID, nil
}

// GetValidationResult retrieves a validation result
func (bvc *BPValidationCoordinatorImpl) GetValidationResult(ctx context.Context, validationID string) (*BPValidationResponse, error) {
	// Try in-memory results first
	if result, exists := bvc.results[validationID]; exists {
		return result, nil
	}

	// Try database
	result, err := bvc.getValidationResultFromDB(ctx, validationID)
	if err != nil {
		return nil, fmt.Errorf("validation result not found: %s", validationID)
	}

	return result, nil
}

// SubscribeToValidationEvents subscribes to validation events for a BP step
func (bvc *BPValidationCoordinatorImpl) SubscribeToValidationEvents(ctx context.Context, bpName string, stepName string) (<-chan *BPValidationResponse, error) {
	eventKey := fmt.Sprintf("%s.%s", bpName, stepName)

	// Create event channel
	eventCh := make(chan *BPValidationResponse, 100)
	bvc.eventChannels[eventKey] = eventCh

	log.Printf("[BPCoordinator] Subscribed to validation events for %s", eventKey)

	return eventCh, nil
}

// ============================================================================
// Event Routing & Actions
// ============================================================================

// routeActions routes validation outcome to configured destinations
func (bvc *BPValidationCoordinatorImpl) routeActions(ctx context.Context, response *BPValidationResponse, req *BPValidationRequest) error {
	for _, action := range response.ActionsToTake {
		if err := bvc.executeAction(ctx, action, response, req); err != nil {
			log.Printf("[BPCoordinator] Error executing action: %v", err)
		}
	}
	return nil
}

// executeAction performs a single action (route to queue, notify, etc.)
func (bvc *BPValidationCoordinatorImpl) executeAction(ctx context.Context, action string, response *BPValidationResponse, req *BPValidationRequest) error {
	// Parse action: "route:queue_name" or "notify:email" or "webhook:url"
	parts := bvc.parseAction(action)
	if len(parts) < 2 {
		return fmt.Errorf("invalid action format: %s", action)
	}

	actionType := parts[0]
	actionTarget := parts[1]

	switch actionType {
	case "route":
		return bvc.routeToQueue(ctx, actionTarget, response, req)

	case "notify":
		return bvc.sendNotification(ctx, actionTarget, response, req)

	case "webhook":
		return bvc.callWebhook(ctx, actionTarget, response, req)

	default:
		log.Printf("[BPCoordinator] Unknown action type: %s", actionType)
		return nil
	}
}

// routeToQueue publishes validation outcome to RabbitMQ queue
func (bvc *BPValidationCoordinatorImpl) routeToQueue(ctx context.Context, queueName string, response *BPValidationResponse, req *BPValidationRequest) error {
	event := map[string]interface{}{
		"validation_id": response.ID,
		"bp_name":       req.BPName,
		"step_name":     req.StepName,
		"tenant_id":     req.TenantID,
		"user_id":       req.UserID,
		"context_id":    req.ContextID,
		"passed":        response.Passed,
		"errors":        response.Errors,
		"form_data":     req.FormData,
		"timestamp":     time.Now(),
	}

	eventJSON, _ := json.Marshal(event)

	if bvc.writer == nil {
		return fmt.Errorf("no kafka writer configured for routing to %s", queueName)
	}

	msg := kafka.Message{
		Topic: queueName,
		Key:   []byte(response.ID),
		Value: eventJSON,
		Time:  time.Now(),
	}

	if err := bvc.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to route to topic %s: %w", queueName, err)
	}

	log.Printf("[BPCoordinator] Routed validation to topic: %s", queueName)
	return nil
}

// sendNotification sends a notification (placeholder)
func (bvc *BPValidationCoordinatorImpl) sendNotification(ctx context.Context, notificationType string, response *BPValidationResponse, req *BPValidationRequest) error {
	log.Printf("[BPCoordinator] Sending %s notification for %s", notificationType, req.BPName)
	// Integrate with notification service
	// Build notification payload
	notificationPayload := map[string]interface{}{
		"type":      notificationType,
		"bp_name":   req.BPName,
		"step_name": req.StepName,
		"user_id":   req.UserID,
		"passed":    response.Passed,
		"errors":    response.Errors,
	}

	// In production: Call actual notification service
	// - Email via SendGrid/SES
	// - SMS via Twilio/SNS
	// - In-app via WebSocket
	fmt.Printf("[BPCoordinator] Would send notification: %v\n", notificationPayload)

	return nil
}

// callWebhook calls an external webhook (placeholder)
func (bvc *BPValidationCoordinatorImpl) callWebhook(ctx context.Context, webhookURL string, response *BPValidationResponse, req *BPValidationRequest) error {
	log.Printf("[BPCoordinator] Calling webhook: %s", webhookURL)
	// Implement webhook invocation
	// Build webhook payload
	payload := map[string]interface{}{
		"validation_id": response.ID,
		"bp_name":       req.BPName,
		"step_name":     req.StepName,
		"passed":        response.Passed,
		"errors":        response.Errors,
		"form_data":     req.FormData,
		"timestamp":     time.Now(),
	}

	// In production: Make HTTP POST request
	// client := &http.Client{Timeout: 10 * time.Second}
	// payloadBytes, _ := json.Marshal(payload)
	// resp, err := client.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	// Handle response and retries

	fmt.Printf("[BPCoordinator] Would call webhook %s with payload: %v\n", webhookURL, payload)
	return nil
}

// ============================================================================
// Audit & Analytics
// ============================================================================

// auditValidation records validation execution for audit trail
func (bvc *BPValidationCoordinatorImpl) auditValidation(ctx context.Context, req *BPValidationRequest, response *BPValidationResponse) {
	query := `
		INSERT INTO bp_validation_executions (
			tenant_id, rule_id, bp_name, step_name, input_data,
			result_passed, error_message, executed_by, executed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	inputJSON, _ := json.Marshal(req.FormData)
	errorMsg := ""
	if !response.Passed && len(response.Errors) > 0 {
		errorMsg = response.Errors[0]
	}

	_, err := bvc.db.ExecContext(ctx, query,
		req.TenantID, "", req.BPName, req.StepName, string(inputJSON),
		response.Passed, errorMsg, req.UserID, time.Now(),
	)

	if err != nil {
		log.Printf("[BPCoordinator] Error recording audit: %v", err)
	}
}

// ============================================================================
// Helper Methods
// ============================================================================

func (bvc *BPValidationCoordinatorImpl) parseAction(action string) []string {
	// Split "route:queue_name" into ["route", "queue_name"]
	parts := make([]string, 0, 2)
	for i, part := range split(action, ':') {
		if i < 2 {
			parts = append(parts, part)
		}
	}
	return parts
}

func split(s string, sep rune) []string {
	var parts []string
	var current []rune

	for _, r := range s {
		if r == sep {
			parts = append(parts, string(current))
			current = []rune{}
		} else {
			current = append(current, r)
		}
	}

	if len(current) > 0 {
		parts = append(parts, string(current))
	}

	return parts
}

func (bvc *BPValidationCoordinatorImpl) publishValidationQueued(ctx context.Context, req *BPValidationRequest, validationID string) {
	event := map[string]interface{}{
		"validation_id": validationID,
		"bp_name":       req.BPName,
		"step_name":     req.StepName,
		"status":        "queued",
		"timestamp":     time.Now(),
	}

	eventJSON, _ := json.Marshal(event)

	if bvc.writer == nil {
		log.Printf("[BPCoordinator] Kafka writer not configured; cannot publish queued event")
		return
	}

	msg := kafka.Message{
		Topic: "validation.results",
		Key:   []byte("validation.queued"),
		Value: eventJSON,
		Time:  time.Now(),
	}

	if err := bvc.writer.WriteMessages(ctx, msg); err != nil {
		log.Printf("[BPCoordinator] Failed to publish queued event: %v", err)
	}
}

func (bvc *BPValidationCoordinatorImpl) getValidationResultFromDB(ctx context.Context, validationID string) (*BPValidationResponse, error) {
	// Implement DB lookup for persisted results
	// Query validation results table
	_ = `
		SELECT id, passed, errors, warnings, actions_to_take, details, timestamp
		FROM bp_validation_results
		WHERE id = $1
	`

	// For now, return not found
	// In production: Execute query and scan results
	// var result BPValidationResponse
	// err := bvc.db.QueryRowContext(ctx, query, validationID).Scan(...)

	return nil, fmt.Errorf("validation result not found in database: %s", validationID)
}
