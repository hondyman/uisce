package workflows

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// Advanced Nodes: Publish Event, Alert, Switch
// ============================================================================

// ========================= PUBLISH EVENT ====================================

// Supported message broker types
const (
	BrokerRabbitMQ        = "rabbitmq"
	BrokerKafka           = "kafka"
	BrokerAWSSQS          = "aws_sqs"
	BrokerAWSSNS          = "aws_sns"
	BrokerAzureServiceBus = "azure_servicebus"
	BrokerAzureEventHub   = "azure_eventhub"
	BrokerGooglePubSub    = "gcp_pubsub"
)

// PublishEventConfig defines configuration for publishing events to message brokers
type PublishEventConfig struct {
	// Common fields
	EventName   string            `json:"event_name"`   // Event type name
	BrokerType  string            `json:"broker_type"`  // rabbitmq, kafka, aws_sqs, aws_sns, azure_servicebus, etc.
	Payload     map[string]string `json:"payload"`      // Key-value payload mapping from state
	ContentType string            `json:"content_type"` // application/json, etc.

	// RabbitMQ-specific (legacy AMQP fields; prefer Kafka topics)
	Exchange   string `json:"exchange"`    // RabbitMQ exchange (legacy)
	RoutingKey string `json:"routing_key"` // RabbitMQ routing key (legacy)

	// Kafka specific
	Topic     string `json:"topic"`     // Kafka topic
	Partition int    `json:"partition"` // Kafka partition (-1 for auto)
	Key       string `json:"key"`       // Message key for partitioning

	// AWS specific
	QueueURL string `json:"queue_url"` // SQS queue URL
	TopicARN string `json:"topic_arn"` // SNS topic ARN
	Region   string `json:"region"`    // AWS region

	// Azure specific
	Namespace    string `json:"namespace"`     // Azure namespace
	QueueName    string `json:"queue_name"`    // Azure queue/topic name
	EventHubName string `json:"eventhub_name"` // Azure Event Hub name

	// GCP specific
	ProjectID string `json:"project_id"` // GCP project ID
	TopicName string `json:"topic_name"` // Pub/Sub topic name
}

// PublishEventResult holds the result of publishing an event
type PublishEventResult struct {
	Published   bool   `json:"published"`
	EventID     string `json:"event_id"`
	EventName   string `json:"event_name"`
	BrokerType  string `json:"broker_type"`
	Destination string `json:"destination"` // Topic/Queue/Exchange used
}

// ExecutePublishEventNode publishes an event to the configured message broker
func ExecutePublishEventNode(
	ctx workflow.Context,
	config PublishEventConfig,
	currentState map[string]interface{},
) (*PublishEventResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Publishing event", "eventName", config.EventName, "broker", config.BrokerType)

	// Build payload from state mapping
	payload := make(map[string]interface{})
	for targetKey, sourcePath := range config.Payload {
		value, err := resolveDataPath(sourcePath, currentState)
		if err != nil {
			logger.Warn("Failed to resolve payload field", "key", targetKey, "path", sourcePath)
			continue
		}
		payload[targetKey] = value
	}

	// Add metadata
	payload["_event_name"] = config.EventName
	payload["_workflow_id"] = workflow.GetInfo(ctx).WorkflowExecution.ID
	payload["_timestamp"] = workflow.Now(ctx).Unix()
	payload["_broker"] = config.BrokerType

	// Determine destination based on broker type
	destination := ""
	activityName := "ActivityPublishEvent"
	activityParams := map[string]interface{}{
		"payload":      payload,
		"content_type": config.ContentType,
		"broker_type":  config.BrokerType,
	}

	switch config.BrokerType {
	case BrokerRabbitMQ:
		// Legacy: explicit RabbitMQ (AMQP) publishing
		destination = config.Exchange + "/" + config.RoutingKey
		activityParams["exchange"] = config.Exchange
		activityParams["routing_key"] = config.RoutingKey
		activityName = "ActivityPublishRabbitMQ"

	case BrokerKafka:
		// Kafka / Redpanda publishing
		destination = config.Topic
		activityParams["topic"] = config.Topic
		activityParams["partition"] = config.Partition
		activityParams["key"] = config.Key
		activityName = "ActivityPublishKafka"

	case BrokerAWSSQS:
		destination = config.QueueURL
		activityParams["queue_url"] = config.QueueURL
		activityParams["region"] = config.Region
		activityName = "ActivityPublishSQS"

	case BrokerAWSSNS:
		destination = config.TopicARN
		activityParams["topic_arn"] = config.TopicARN
		activityParams["region"] = config.Region
		activityName = "ActivityPublishSNS"

	case BrokerAzureServiceBus:
		destination = config.Namespace + "/" + config.QueueName
		activityParams["namespace"] = config.Namespace
		activityParams["queue_name"] = config.QueueName
		activityName = "ActivityPublishServiceBus"

	case BrokerAzureEventHub:
		destination = config.Namespace + "/" + config.EventHubName
		activityParams["namespace"] = config.Namespace
		activityParams["eventhub_name"] = config.EventHubName
		activityName = "ActivityPublishEventHub"

	case BrokerGooglePubSub:
		destination = config.ProjectID + "/" + config.TopicName
		activityParams["project_id"] = config.ProjectID
		activityParams["topic_name"] = config.TopicName
		activityName = "ActivityPublishPubSub"
	}

	// Execute appropriate activity
	var result PublishEventResult
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: workflow.GetInfo(ctx).WorkflowExecutionTimeout / 10,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	err := workflow.ExecuteActivity(ctx, activityName, activityParams).Get(ctx, &result)

	if err != nil {
		logger.Error("Failed to publish event", "broker", config.BrokerType, "error", err)
		return &PublishEventResult{
			Published:   false,
			EventName:   config.EventName,
			BrokerType:  config.BrokerType,
			Destination: destination,
		}, nil
	}

	logger.Info("Event published", "eventId", result.EventID, "broker", config.BrokerType)
	return &PublishEventResult{
		Published:   true,
		EventID:     result.EventID,
		EventName:   config.EventName,
		BrokerType:  config.BrokerType,
		Destination: destination,
	}, nil
}

// ParsePublishEventConfig extracts config from node
func ParsePublishEventConfig(config map[string]interface{}) (*PublishEventConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var cfg PublishEventConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse publish event config: %w", err)
	}

	if cfg.EventName == "" {
		return nil, fmt.Errorf("event_name is required for publishEvent node")
	}
	if cfg.BrokerType == "" {
		cfg.BrokerType = BrokerKafka // Default to Kafka/Redpanda
	}
	if cfg.ContentType == "" {
		cfg.ContentType = "application/json"
	}

	// Set defaults based on broker type
	switch cfg.BrokerType {
	case BrokerRabbitMQ:
		if cfg.Exchange == "" {
			cfg.Exchange = "titan.events"
		}
	case BrokerKafka:
		if cfg.Partition == 0 {
			cfg.Partition = -1 // Auto-partition
		}
	case BrokerAWSSQS, BrokerAWSSNS:
		if cfg.Region == "" {
			cfg.Region = "us-east-1"
		}
	}

	return &cfg, nil
}

// ========================= ALERT ============================================

// AlertConfig defines configuration for sending notifications/alerts
type AlertConfig struct {
	Channel    string            `json:"channel"`    // "email", "slack", "webhook", "sms"
	Severity   string            `json:"severity"`   // "info", "warning", "error", "critical"
	Subject    string            `json:"subject"`    // Alert subject/title
	Message    string            `json:"message"`    // Alert message body
	Template   string            `json:"template"`   // Optional template ID
	Data       map[string]string `json:"data"`       // Template data mapping
	Recipients []string          `json:"recipients"` // Target recipients
}

// AlertResult holds the result of sending an alert
type AlertResult struct {
	Sent       bool   `json:"sent"`
	AlertID    string `json:"alert_id"`
	Channel    string `json:"channel"`
	Recipients int    `json:"recipients"`
}

// ExecuteAlertNode sends an alert/notification
func ExecuteAlertNode(
	ctx workflow.Context,
	config AlertConfig,
	currentState map[string]interface{},
) (*AlertResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Sending alert", "channel", config.Channel, "severity", config.Severity)

	// Resolve template data from state
	resolvedData := make(map[string]interface{})
	for key, sourcePath := range config.Data {
		value, err := resolveDataPath(sourcePath, currentState)
		if err != nil {
			resolvedData[key] = sourcePath // Use literal value
		} else {
			resolvedData[key] = value
		}
	}

	// Interpolate message with state values
	message := config.Message
	for key, val := range resolvedData {
		message = strings.ReplaceAll(message, "{{"+key+"}}", fmt.Sprintf("%v", val))
	}

	// Execute activity to send alert
	var result AlertResult
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: workflow.GetInfo(ctx).WorkflowExecutionTimeout / 10,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	err := workflow.ExecuteActivity(ctx, "ActivitySendAlert", map[string]interface{}{
		"channel":    config.Channel,
		"severity":   config.Severity,
		"subject":    config.Subject,
		"message":    message,
		"template":   config.Template,
		"data":       resolvedData,
		"recipients": config.Recipients,
	}).Get(ctx, &result)

	if err != nil {
		logger.Error("Failed to send alert", "error", err)
		return &AlertResult{Sent: false, Channel: config.Channel}, nil
	}

	logger.Info("Alert sent", "alertId", result.AlertID, "recipients", result.Recipients)
	return &result, nil
}

// ParseAlertConfig extracts config from node
func ParseAlertConfig(config map[string]interface{}) (*AlertConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var cfg AlertConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse alert config: %w", err)
	}

	if cfg.Channel == "" {
		cfg.Channel = "email" // Default
	}
	if cfg.Severity == "" {
		cfg.Severity = "info"
	}

	return &cfg, nil
}

// ========================= SWITCH (Multi-way Branch) ========================

// SwitchConfig defines configuration for multi-way branching
type SwitchConfig struct {
	Expression string       `json:"expression"` // Field to evaluate (JSONPath)
	Cases      []SwitchCase `json:"cases"`      // Case definitions
	DefaultID  string       `json:"default_id"` // Default node if no match
}

// SwitchCase defines a single case in the switch
type SwitchCase struct {
	Value      interface{} `json:"value"`       // Value to match
	TargetNode string      `json:"target_node"` // Node to jump to
	Label      string      `json:"label"`       // Display label
}

// SwitchResult holds the result of switch evaluation
type SwitchResult struct {
	Expression   string      `json:"expression"`
	Value        interface{} `json:"value"`
	MatchedCase  string      `json:"matched_case"`
	TargetNodeID string      `json:"target_node_id"`
	IsDefault    bool        `json:"is_default"`
}

// ExecuteSwitchNode evaluates a switch expression and returns the target node
func ExecuteSwitchNode(
	ctx workflow.Context,
	config SwitchConfig,
	currentState map[string]interface{},
) (*SwitchResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Evaluating switch", "expression", config.Expression, "caseCount", len(config.Cases))

	// Resolve expression value from state
	value, err := resolveDataPath(config.Expression, currentState)
	if err != nil {
		logger.Warn("Failed to resolve switch expression", "expression", config.Expression, "error", err)
		// Use default
		return &SwitchResult{
			Expression:   config.Expression,
			Value:        nil,
			TargetNodeID: config.DefaultID,
			IsDefault:    true,
		}, nil
	}

	// Find matching case
	for _, c := range config.Cases {
		if matchesCase(value, c.Value) {
			logger.Info("Switch matched case", "value", value, "case", c.Label)
			return &SwitchResult{
				Expression:   config.Expression,
				Value:        value,
				MatchedCase:  c.Label,
				TargetNodeID: c.TargetNode,
				IsDefault:    false,
			}, nil
		}
	}

	// No match - use default
	if config.DefaultID == "" {
		return nil, temporal.NewApplicationError("no matching case and no default defined", "SWITCH_NO_MATCH")
	}

	logger.Info("Switch using default", "value", value)
	return &SwitchResult{
		Expression:   config.Expression,
		Value:        value,
		TargetNodeID: config.DefaultID,
		IsDefault:    true,
	}, nil
}

// matchesCase compares values with type coercion
func matchesCase(actual, expected interface{}) bool {
	// String comparison
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)
	return actualStr == expectedStr
}

// ParseSwitchConfig extracts config from node
func ParseSwitchConfig(config map[string]interface{}) (*SwitchConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var cfg SwitchConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse switch config: %w", err)
	}

	if cfg.Expression == "" {
		return nil, fmt.Errorf("expression is required for switch node")
	}
	if len(cfg.Cases) == 0 && cfg.DefaultID == "" {
		return nil, fmt.Errorf("switch node requires at least one case or a default")
	}

	return &cfg, nil
}

// ========================= HELPERS ==========================================

// IsAdvancedNode checks if a node type is an advanced node
func IsAdvancedNode(nodeType string) bool {
	switch strings.ToLower(nodeType) {
	case "publishevent", "publish_event", "emit":
		return true
	case "alert", "notify":
		return true
	case "switch":
		return true
	default:
		return false
	}
}
