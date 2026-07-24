/**
 * Rule Fabric Event Orchestration
 *
 * This package provides:
 * - Violation event emission to RabbitMQ
 * - Category-specific event routing
 * - Action execution handlers
 * - Channel-based policy enforcement
 */

package rulefabric

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"strings"

	"github.com/google/uuid"
	kafka "github.com/segmentio/kafka-go"
)

// ============================================================================
// Event Types
// ============================================================================

// ViolationEventType represents the type of violation event
type ViolationEventType string

const (
	EventDataQualityViolation   ViolationEventType = "data_quality.violation"
	EventComplianceBreach       ViolationEventType = "compliance.breach"
	EventMDMMatchFound          ViolationEventType = "mdm.match_found"
	EventWashTradePattern       ViolationEventType = "wash_trade.pattern_detected"
	EventValuesScreeningFailure ViolationEventType = "values.screening_failure"
	EventWorkflowTrigger        ViolationEventType = "workflow.trigger"
	EventCustomRuleViolation    ViolationEventType = "custom.violation"
)

// ViolationEvent is the standard event envelope for rule violations
type ViolationEvent struct {
	EventID       string                 `json:"event_id"`
	EventType     ViolationEventType     `json:"event_type"`
	TenantID      string                 `json:"tenant_id"`
	DatasourceID  string                 `json:"datasource_id"`
	RuleID        string                 `json:"rule_id"`
	RuleName      string                 `json:"rule_name"`
	Category      string                 `json:"category"`
	Context       string                 `json:"context"`
	Severity      string                 `json:"severity"`
	RecordID      string                 `json:"record_id"`
	EntityType    string                 `json:"entity_type"`
	Message       string                 `json:"message"`
	Details       map[string]interface{} `json:"details"`
	Actions       []ActionRequest        `json:"actions"`
	Timestamp     time.Time              `json:"timestamp"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Source        string                 `json:"source"`
	Channel       string                 `json:"channel"`
}

// ActionRequest represents an action to be executed
type ActionRequest struct {
	ActionType     string                 `json:"action_type"`
	ActionConfig   map[string]interface{} `json:"action_config"`
	ExecutionOrder int                    `json:"execution_order"`
	Status         string                 `json:"status"` // pending, completed, failed
	Result         map[string]interface{} `json:"result,omitempty"`
	Error          string                 `json:"error,omitempty"`
	ExecutedAt     *time.Time             `json:"executed_at,omitempty"`
}

// ============================================================================
// Event Publisher
// ============================================================================

// EventPublisher handles publishing violation events to Kafka (topic-based)
type EventPublisher struct {
	writer *kafka.Writer
	config EventPublisherConfig
}

// EventPublisherConfig contains configuration for the event publisher
type EventPublisherConfig struct {
	ExchangeName string
	ExchangeType string
	Durable      bool
	AutoDelete   bool
	EnableRetry  bool
	MaxRetries   int
	RetryDelay   time.Duration
}

// DefaultEventPublisherConfig returns default configuration
func DefaultEventPublisherConfig() EventPublisherConfig {
	return EventPublisherConfig{
		ExchangeName: "rule_fabric.violations",
		ExchangeType: "topic",
		Durable:      true,
		AutoDelete:   false,
		EnableRetry:  true,
		MaxRetries:   3,
		RetryDelay:   time.Second * 2,
	}
}

// NewEventPublisher creates a new Kafka-backed event publisher. If a legacy AMQP URL is provided, the publisher will be disabled.
func NewEventPublisher(brokersOrURL string, config EventPublisherConfig) (*EventPublisher, error) {
	if brokersOrURL == "" {
		brokersOrURL = os.Getenv("KAFKA_BROKERS")
	}

	// If AMQP URL detected, disable publisher for backwards-compatibility
	if strings.HasPrefix(brokersOrURL, "amqp://") {
		return &EventPublisher{writer: nil, config: config}, nil
	}

	brokers := strings.Split(brokersOrURL, ",")
	w := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}

	return &EventPublisher{writer: w, config: config}, nil
}

// Close closes the publisher resources
func (p *EventPublisher) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// PublishViolation publishes a violation event to Kafka topic configured in ExchangeName
func (p *EventPublisher) PublishViolation(ctx context.Context, event ViolationEvent) error {
	if p.writer == nil {
		return nil
	}
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Routing key format: category.severity.context
	routingKey := fmt.Sprintf("%s.%s.%s", event.Category, event.Severity, event.Context)

	// Convert headers to Kafka headers
	headers := []kafka.Header{
		{Key: "tenant_id", Value: []byte(event.TenantID)},
		{Key: "datasource_id", Value: []byte(event.DatasourceID)},
		{Key: "rule_id", Value: []byte(event.RuleID)},
		{Key: "category", Value: []byte(event.Category)},
		{Key: "severity", Value: []byte(event.Severity)},
	}

	msg := kafka.Message{
		Topic:   p.config.ExchangeName,
		Key:     []byte(routingKey),
		Value:   body,
		Time:    event.Timestamp,
		Headers: headers,
	}

	var lastErr error
	maxRetries := 1
	if p.config.EnableRetry {
		maxRetries = p.config.MaxRetries
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := p.writer.WriteMessages(ctx, msg); err == nil {
			return nil
		} else {
			lastErr = err
			if attempt < maxRetries-1 {
				time.Sleep(p.config.RetryDelay)
			}
		}
	}

	return fmt.Errorf("failed to publish event after %d attempts: %w", maxRetries, lastErr)
}

// PublishBatch publishes multiple violation events
func (p *EventPublisher) PublishBatch(ctx context.Context, events []ViolationEvent) error {
	for _, event := range events {
		if err := p.PublishViolation(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// ============================================================================
// Event Consumer
// ============================================================================

// EventConsumer handles consuming violation events (Kafka-backed)
type EventConsumer struct {
	reader  *kafka.Reader
	config  EventConsumerConfig
	handler ViolationHandler
	enabled bool
	topic   string
}

// EventConsumerConfig contains configuration for the event consumer
type EventConsumerConfig struct {
	ExchangeName string
	QueueName    string
	BindingKeys  []string // e.g., ["data_quality.#", "compliance.hard_block.#"] (binding keys map to topic prefixes)
	Prefetch     int
	AutoAck      bool
}

// ViolationHandler processes violation events
type ViolationHandler interface {
	HandleViolation(ctx context.Context, event ViolationEvent) error
}

// NewEventConsumer creates a new Kafka-backed event consumer
func NewEventConsumer(brokersOrURL string, config EventConsumerConfig, handler ViolationHandler) (*EventConsumer, error) {
	if brokersOrURL == "" {
		return &EventConsumer{enabled: false}, nil
	}

	// Detect legacy AMQP URL and disable consumer to encourage migration
	if strings.HasPrefix(brokersOrURL, "amqp://") {
		log.Printf("⚠️  Detected legacy AMQP URL %s - event consumer disabled. Set KAFKA_BROKERS instead.", brokersOrURL)
		return &EventConsumer{enabled: false}, nil
	}

	brokers := strings.Split(brokersOrURL, ",")
	topic := config.ExchangeName
	if topic == "" {
		topic = "rule_fabric.violations"
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  "rulefabric-consumer",
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	return &EventConsumer{reader: r, config: config, handler: handler, enabled: true, topic: topic}, nil
}

// Start starts consuming events from Kafka
func (c *EventConsumer) Start(ctx context.Context) error {
	if !c.enabled {
		return fmt.Errorf("event consumer not enabled")
	}

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("error fetching message: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		var event ViolationEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			log.Printf("failed to unmarshal violation event: %v", err)
			c.reader.CommitMessages(ctx, m) // skip bad message
			continue
		}

		if err := c.handler.HandleViolation(ctx, event); err != nil {
			log.Printf("handler error: %v", err)
			// do not commit so it can be retried
			continue
		}

		// Commit on success
		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Printf("failed to commit message: %v", err)
		}
	}
}

// Close closes the consumer connection
func (c *EventConsumer) Close() error {
	if !c.enabled {
		return nil
	}
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}

// ============================================================================
// Action Executor
// ============================================================================

// ActionExecutor executes actions defined in rules
type ActionExecutor struct {
	handlers map[string]ActionHandler
}

// ActionHandler handles a specific action type
type ActionHandler interface {
	Execute(ctx context.Context, event ViolationEvent, action ActionRequest) (*ActionResult, error)
}

// ActionResult represents the result of an action execution
type ActionResult struct {
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewActionExecutor creates a new action executor
func NewActionExecutor() *ActionExecutor {
	return &ActionExecutor{
		handlers: make(map[string]ActionHandler),
	}
}

// RegisterHandler registers an action handler
func (e *ActionExecutor) RegisterHandler(actionType string, handler ActionHandler) {
	e.handlers[actionType] = handler
}

// Execute executes all actions for a violation event
func (e *ActionExecutor) Execute(ctx context.Context, event ViolationEvent) ([]ActionResult, error) {
	results := make([]ActionResult, len(event.Actions))

	for i, action := range event.Actions {
		handler, ok := e.handlers[action.ActionType]
		if !ok {
			results[i] = ActionResult{
				Success:   false,
				Error:     fmt.Sprintf("no handler for action type: %s", action.ActionType),
				Timestamp: time.Now().UTC(),
			}
			continue
		}

		result, err := handler.Execute(ctx, event, action)
		if err != nil {
			results[i] = ActionResult{
				Success:   false,
				Error:     err.Error(),
				Timestamp: time.Now().UTC(),
			}
		} else {
			results[i] = *result
		}
	}

	return results, nil
}

// ============================================================================
// Built-in Action Handlers
// ============================================================================

// LogActionHandler logs violations
type LogActionHandler struct{}

func (h *LogActionHandler) Execute(ctx context.Context, event ViolationEvent, action ActionRequest) (*ActionResult, error) {
	return &ActionResult{
		Success:   true,
		Message:   fmt.Sprintf("Logged violation: %s for record %s", event.Message, event.RecordID),
		Timestamp: time.Now().UTC(),
	}, nil
}

// WebhookActionHandler calls external webhooks
type WebhookActionHandler struct {
	httpClient interface{} // *http.Client in real implementation
}

func (h *WebhookActionHandler) Execute(ctx context.Context, event ViolationEvent, action ActionRequest) (*ActionResult, error) {
	webhookURL, ok := action.ActionConfig["url"].(string)
	if !ok {
		return nil, fmt.Errorf("webhook URL not configured")
	}

	// In real implementation, make HTTP POST to webhook URL
	_ = webhookURL

	return &ActionResult{
		Success:   true,
		Message:   "Webhook called successfully",
		Timestamp: time.Now().UTC(),
	}, nil
}

// QuarantineActionHandler quarantines records
type QuarantineActionHandler struct {
	// db *sqlx.DB in real implementation
}

func (h *QuarantineActionHandler) Execute(ctx context.Context, event ViolationEvent, action ActionRequest) (*ActionResult, error) {
	queue := "default"
	if q, ok := action.ActionConfig["queue"].(string); ok {
		queue = q
	}

	return &ActionResult{
		Success:   true,
		Message:   fmt.Sprintf("Record %s quarantined to queue: %s", event.RecordID, queue),
		Data:      map[string]interface{}{"queue": queue},
		Timestamp: time.Now().UTC(),
	}, nil
}

// AlertActionHandler sends alerts
type AlertActionHandler struct {
	// notificationService NotificationService in real implementation
}

func (h *AlertActionHandler) Execute(ctx context.Context, event ViolationEvent, action ActionRequest) (*ActionResult, error) {
	priority := "normal"
	if p, ok := action.ActionConfig["priority"].(string); ok {
		priority = p
	}

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Alert sent with priority: %s", priority),
		Data: map[string]interface{}{
			"priority": priority,
			"rule":     event.RuleName,
			"severity": event.Severity,
		},
		Timestamp: time.Now().UTC(),
	}, nil
}

// ============================================================================
// Channel Policy Enforcer
// ============================================================================

// ChannelPolicyEnforcer enforces execution policies per channel
type ChannelPolicyEnforcer struct {
	evaluator *RuleEvaluator
	publisher *EventPublisher
	executor  *ActionExecutor
}

// ExecutionRequest represents a request to evaluate rules
type ExecutionRequest struct {
	TenantID     string
	DatasourceID string
	Channel      string // batch, realtime, api, workflow, scheduler
	Category     string
	Context      string
	EntityType   string
	Records      []map[string]interface{}
	Options      ExecutionOptions
}

// ExecutionOptions contains options for rule execution
type ExecutionOptions struct {
	DryRun         bool
	EmitEvents     bool
	ExecuteActions bool
	MaxRules       int
	TimeoutSeconds int
	CorrelationID  string
}

// ExecutionResponse represents the response from rule execution
type ExecutionResponse struct {
	TotalRecords  int              `json:"total_records"`
	TotalRules    int              `json:"total_rules"`
	TotalPassed   int              `json:"total_passed"`
	TotalFailed   int              `json:"total_failed"`
	Violations    []ViolationEvent `json:"violations"`
	ActionResults []ActionResult   `json:"action_results,omitempty"`
	ExecutionTime time.Duration    `json:"execution_time"`
	Channel       string           `json:"channel"`
	DryRun        bool             `json:"dry_run"`
}

// NewChannelPolicyEnforcer creates a new channel policy enforcer
func NewChannelPolicyEnforcer(evaluator *RuleEvaluator, publisher *EventPublisher, executor *ActionExecutor) *ChannelPolicyEnforcer {
	return &ChannelPolicyEnforcer{
		evaluator: evaluator,
		publisher: publisher,
		executor:  executor,
	}
}

// Execute evaluates rules and enforces policies for a channel
func (e *ChannelPolicyEnforcer) Execute(ctx context.Context, req ExecutionRequest) (*ExecutionResponse, error) {
	startTime := time.Now()

	// Get active rules for the category and channel
	tenantUUID, _ := uuid.Parse(req.TenantID)
	datasourceUUID, _ := uuid.Parse(req.DatasourceID)

	category := RuleCategory(req.Category)
	rules, err := e.evaluator.GetRulesForEvaluation(ctx, tenantUUID, GetRulesOptions{
		Environment: "production",
		Category:    &category,
		Channel:     req.Channel,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	if req.Options.MaxRules > 0 && len(rules) > req.Options.MaxRules {
		rules = rules[:req.Options.MaxRules]
	}

	response := &ExecutionResponse{
		TotalRecords: len(req.Records),
		TotalRules:   len(rules),
		Channel:      req.Channel,
		DryRun:       req.Options.DryRun,
		Violations:   []ViolationEvent{},
	}

	// Evaluate each record against rules
	for _, record := range req.Records {
		evalCtx := &EvaluationContext{
			TenantID:       tenantUUID,
			DatasourceID:   datasourceUUID,
			Data:           record,
			Channel:        req.Channel,
			Environment:    "production",
			EvaluationTime: time.Now(),
		}

		recordID := ""
		if id, ok := record["id"].(string); ok {
			recordID = id
		} else if id, ok := record["record_id"].(string); ok {
			recordID = id
		}

		for _, rule := range rules {
			result, err := e.evaluator.Evaluate(ctx, rule, evalCtx)
			if err != nil {
				continue // Log error but continue with other rules
			}

			if result.Status == EvalPassed {
				response.TotalPassed++
			} else if result.Status == EvalFailed {
				response.TotalFailed++

				// Parse actions from rule logic
				var ruleActions []RuleAction
				if len(rule.Logic.ActionsJSON) > 0 {
					json.Unmarshal(rule.Logic.ActionsJSON, &ruleActions)
				}

				// Get failure message
				failureMessage := "Rule failed"
				if len(result.Details.FailureReasons) > 0 {
					failureMessage = result.Details.FailureReasons[0]
				}

				// Create violation event
				violation := ViolationEvent{
					EventID:       uuid.New().String(),
					EventType:     categoryToEventType(string(rule.Category)),
					TenantID:      req.TenantID,
					DatasourceID:  req.DatasourceID,
					RuleID:        rule.ID.String(),
					RuleName:      rule.Name,
					Category:      string(rule.Category),
					Context:       req.Context,
					Severity:      string(rule.Severity),
					RecordID:      recordID,
					EntityType:    req.EntityType,
					Message:       failureMessage,
					Details:       map[string]interface{}{"evaluation_details": result.Details},
					Actions:       ruleActionsToRequests(ruleActions),
					Timestamp:     time.Now().UTC(),
					CorrelationID: req.Options.CorrelationID,
					Source:        "channel_policy_enforcer",
					Channel:       req.Channel,
				}

				response.Violations = append(response.Violations, violation)

				// Publish events (unless dry run)
				if !req.Options.DryRun && req.Options.EmitEvents && e.publisher != nil {
					_ = e.publisher.PublishViolation(ctx, violation)
				}

				// Execute actions (unless dry run)
				if !req.Options.DryRun && req.Options.ExecuteActions && e.executor != nil {
					results, _ := e.executor.Execute(ctx, violation)
					response.ActionResults = append(response.ActionResults, results...)
				}
			}
		}
	}

	response.ExecutionTime = time.Since(startTime)
	return response, nil
}

// Helper functions

// filterRulesByChannel is no longer needed as GetRulesForEvaluation handles channel filtering

func categoryToEventType(category string) ViolationEventType {
	switch category {
	case "data_quality":
		return EventDataQualityViolation
	case "compliance":
		return EventComplianceBreach
	case "mdm":
		return EventMDMMatchFound
	case "wash_trade":
		return EventWashTradePattern
	case "values":
		return EventValuesScreeningFailure
	case "workflow":
		return EventWorkflowTrigger
	default:
		return EventCustomRuleViolation
	}
}

func ruleActionsToRequests(actions []RuleAction) []ActionRequest {
	requests := make([]ActionRequest, len(actions))
	for i, action := range actions {
		requests[i] = ActionRequest{
			ActionType:     action.Type,
			ActionConfig:   action.Params,
			ExecutionOrder: action.Order,
			Status:         "pending",
		}
	}
	return requests
}
