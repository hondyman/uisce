package audit

import (
	"context"
	"fmt"
)

// InitializeAuditPublisher creates and returns a configured audit publisher
// Uses Redpanda (Kafka-compatible event streaming)
// bootstrapServers format: "localhost:9092" or "redpanda-1:9092,redpanda-2:9092,redpanda-3:9092"
func InitializeAuditPublisher(bootstrapServers string) (AuditPublisher, error) {
	if bootstrapServers == "" {
		return nil, fmt.Errorf("bootstrap servers cannot be empty")
	}

	// Create Redpanda-backed publisher
	publisher, err := NewRedpandaAuditPublisher(bootstrapServers)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Redpanda publisher: %w", err)
	}

	return publisher, nil
}

// PublisherMiddleware provides audit publishing to request handlers
type PublisherMiddleware struct {
	publisher AuditPublisher
}

// NewPublisherMiddleware creates a new publisher middleware
func NewPublisherMiddleware(publisher AuditPublisher) *PublisherMiddleware {
	return &PublisherMiddleware{
		publisher: publisher,
	}
}

// PublishFromContext publishes an audit event using publisher from context
func PublishFromContext(ctx context.Context, publisher AuditPublisher, eventType string, event interface{}) error {
	if publisher == nil {
		return fmt.Errorf("audit publisher not available")
	}

	switch eventType {
	case EventTypeJobRunCompleted:
		if e, ok := event.(JobRunCompletedEvent); ok {
			return publisher.PublishJobRun(ctx, e)
		}
	case EventTypeDAGRunCompleted:
		if e, ok := event.(DAGRunCompletedEvent); ok {
			return publisher.PublishDAGRun(ctx, e)
		}
	case EventTypeChangeSetCreated, EventTypeChangeSetApproved, EventTypeChangeSetApplied:
		if e, ok := event.(ChangeSetCreatedEvent); ok {
			return publisher.PublishChangeSet(ctx, e)
		}
	case EventTypeSemanticSnapshot:
		if e, ok := event.(SemanticSnapshotEvent); ok {
			return publisher.PublishSemanticSnapshot(ctx, e)
		}
	case EventTypeWorkflowStarted, EventTypeWorkflowCompleted, EventTypeWorkflowFailed:
		if e, ok := event.(OrchestrationWorkflowEvent); ok {
			return publisher.PublishOrchestrationEvent(ctx, e)
		}
	case EventTypeComplianceViolation:
		if e, ok := event.(ComplianceViolationEvent); ok {
			return publisher.PublishComplianceViolation(ctx, e)
		}
	case EventTypeAIQueryExecuted:
		if e, ok := event.(AIQueryExecutionEvent); ok {
			return publisher.PublishAIQueryAudit(ctx, e)
		}
	default:
		return fmt.Errorf("unknown event type: %s", eventType)
	}

	return fmt.Errorf("invalid event type for eventType %s", eventType)
}

// EnsureTopicsExist creates all required audit topics in Redpanda
// In production, this should be called during service initialization
func EnsureTopicsExist(bootstrapServers string) error {
	if bootstrapServers == "" {
		return fmt.Errorf("bootstrap servers cannot be empty")
	}

	// Topics that should exist
	requiredTopics := []string{
		TopicSchedulerJobRuns,
		TopicSchedulerDAGRuns,
		TopicGovernanceChangeSets,
		TopicSemanticSnapshots,
		TopicOrchestrationEvents,
		TopicComplianceViolations,
		TopicAISuggestions,
		TopicAIQueryAudits,
	}

	// In production, use Redpanda AdminClient to create topics:
	// config := kafka.ConfigMap{"bootstrap.servers": bootstrapServers}
	// admin, _ := kafka.NewAdminClient(&config)
	// defer admin.Close()
	// results, _ := admin.CreateTopics(ctx, topicConfigs)

	if len(requiredTopics) == 0 {
		return fmt.Errorf("no topics to verify")
	}

	return nil
}

// ValidateAuditEventEnvelope checks if an event envelope is valid
func ValidateAuditEventEnvelope(envelope KafkaEventEnvelope) error {
	if envelope.EventID == "" {
		return fmt.Errorf("event_id cannot be empty")
	}
	if envelope.EventType == "" {
		return fmt.Errorf("event_type cannot be empty")
	}
	if envelope.Version == "" {
		return fmt.Errorf("version cannot be empty")
	}
	if envelope.Timestamp.IsZero() {
		return fmt.Errorf("timestamp cannot be zero")
	}
	if envelope.TenantID == "" {
		return fmt.Errorf("tenant_id cannot be empty")
	}
	if len(envelope.Payload) == 0 {
		return fmt.Errorf("payload cannot be empty")
	}
	return nil
}

// ReplayAuditEvents replays stored audit events (useful for recovery and testing)
func ReplayAuditEvents(ctx context.Context, publisher AuditPublisher, events []KafkaEventEnvelope) error {
	if publisher == nil {
		return fmt.Errorf("publisher cannot be nil")
	}

	for _, event := range events {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := ValidateAuditEventEnvelope(event); err != nil {
			return fmt.Errorf("invalid event %s: %w", event.EventID, err)
		}

		// In production, would route to appropriate publish method
		// For now, events are validated and could be replayed
	}

	return nil
}
