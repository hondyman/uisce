package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// AuditPublisher defines the interface for publishing audit events
type AuditPublisher interface {
	PublishJobRun(ctx context.Context, event JobRunCompletedEvent) error
	PublishDAGRun(ctx context.Context, event interface{}) error
	PublishChangeSet(ctx context.Context, event ChangeSetCreatedEvent) error
	PublishSemanticSnapshot(ctx context.Context, event SemanticSnapshotEvent) error
	PublishOrchestrationEvent(ctx context.Context, event OrchestrationWorkflowEvent) error
	PublishComplianceViolation(ctx context.Context, event ComplianceViolationEvent) error
	PublishAIQueryAudit(ctx context.Context, event AIQueryExecutionEvent) error
	Close() error
}

// RedpandaAuditPublisher publishes audit events to Redpanda topics
// Redpanda is Kafka-compatible and uses the same protocol as Kafka
type RedpandaAuditPublisher struct {
	writer *kafka.Writer
	mu     sync.RWMutex
	closed bool
}

// NewRedpandaAuditPublisher creates a new Redpanda-backed audit event publisher
// bootstrapServers should be in format: "host1:9092,host2:9092,host3:9092"
func NewRedpandaAuditPublisher(bootstrapServers string) (*RedpandaAuditPublisher, error) {
	if bootstrapServers == "" {
		return nil, fmt.Errorf("bootstrap servers cannot be empty")
	}

	w := &kafka.Writer{
		Addr:                   kafka.TCP(bootstrapServers),
		Balancer:               &kafka.LeastBytes{},
		RequiredAcks:           kafka.RequireAll,
		Async:                  false, // We wait for delivery confirmation in Publish
		Compression:            kafka.Snappy,
		BatchTimeout:           100 * time.Millisecond,
		BatchSize:              100,
		MaxAttempts:            3,
		ReadTimeout:            10 * time.Second,
		WriteTimeout:           10 * time.Second,
		AllowAutoTopicCreation: true,
	}

	return &RedpandaAuditPublisher{
		writer: w,
		closed: false,
	}, nil
}

// PublishJobRun publishes a job run completion event
func (p *RedpandaAuditPublisher) PublishJobRun(ctx context.Context, event JobRunCompletedEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal job run event: %w", err)
	}

	envelope := KafkaEventEnvelope{
		EventID:   uuid.New().String(),
		EventType: EventTypeJobRunCompleted,
		Version:   "1.0",
		Timestamp: time.Now().UTC(),
		TenantID:  event.TenantID,
		Source:    "scheduler",
		Payload:   payload,
	}

	return p.publishToRedpanda(ctx, TopicSchedulerJobRuns, &envelope, event.TenantID)
}

// PublishDAGRun publishes a DAG run completion event
func (p *RedpandaAuditPublisher) PublishDAGRun(ctx context.Context, event interface{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal DAG run event: %w", err)
	}

	// Extract tenant ID from event if possible, otherwise use default
	tenantID := "unknown"
	if dagEvent, ok := event.(DAGRunCompletedEvent); ok {
		tenantID = dagEvent.TenantID
	}

	envelope := KafkaEventEnvelope{
		EventID:   uuid.New().String(),
		EventType: EventTypeDAGRunCompleted,
		Version:   "1.0",
		Timestamp: time.Now().UTC(),
		TenantID:  tenantID,
		Source:    "scheduler",
		Payload:   payload,
	}

	return p.publishToRedpanda(ctx, TopicSchedulerDAGRuns, &envelope, tenantID)
}

// PublishChangeSet publishes a governance changeset event
func (p *RedpandaAuditPublisher) PublishChangeSet(ctx context.Context, event ChangeSetCreatedEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal changeset event: %w", err)
	}

	envelope := KafkaEventEnvelope{
		EventID:   uuid.New().String(),
		EventType: EventTypeChangeSetCreated,
		Version:   "1.0",
		Timestamp: time.Now().UTC(),
		TenantID:  event.TenantID,
		Source:    "governance",
		Payload:   payload,
	}

	return p.publishToRedpanda(ctx, TopicGovernanceChangeSets, &envelope, event.TenantID)
}

// PublishSemanticSnapshot publishes a semantic snapshot event
func (p *RedpandaAuditPublisher) PublishSemanticSnapshot(ctx context.Context, event SemanticSnapshotEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal semantic snapshot event: %w", err)
	}

	envelope := KafkaEventEnvelope{
		EventID:   uuid.New().String(),
		EventType: EventTypeSemanticSnapshot,
		Version:   "1.0",
		Timestamp: time.Now().UTC(),
		TenantID:  event.TenantID,
		Source:    "semantic",
		Payload:   payload,
	}

	return p.publishToRedpanda(ctx, TopicSemanticSnapshots, &envelope, event.TenantID)
}

// PublishOrchestrationEvent publishes a Temporal workflow event
func (p *RedpandaAuditPublisher) PublishOrchestrationEvent(ctx context.Context, event OrchestrationWorkflowEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal orchestration event: %w", err)
	}

	envelope := KafkaEventEnvelope{
		EventID:   uuid.New().String(),
		EventType: event.EventType,
		Version:   "1.0",
		Timestamp: time.Now().UTC(),
		TenantID:  event.TenantID,
		Source:    "orchestration",
		Payload:   payload,
	}

	return p.publishToRedpanda(ctx, TopicOrchestrationEvents, &envelope, event.TenantID)
}

// PublishComplianceViolation publishes a compliance violation event
func (p *RedpandaAuditPublisher) PublishComplianceViolation(ctx context.Context, event ComplianceViolationEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal compliance violation event: %w", err)
	}

	envelope := KafkaEventEnvelope{
		EventID:   uuid.New().String(),
		EventType: EventTypeComplianceViolation,
		Version:   "1.0",
		Timestamp: time.Now().UTC(),
		TenantID:  event.TenantID,
		Source:    "compliance",
		Payload:   payload,
	}

	return p.publishToRedpanda(ctx, TopicComplianceViolations, &envelope, event.TenantID)
}

// PublishAIQueryAudit publishes an AI query run audit event
func (p *RedpandaAuditPublisher) PublishAIQueryAudit(ctx context.Context, event AIQueryExecutionEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal AI query audit event: %w", err)
	}

	envelope := KafkaEventEnvelope{
		EventID:   uuid.New().String(),
		EventType: EventTypeAIQueryExecuted,
		Version:   "1.0",
		Timestamp: time.Now().UTC(),
		TenantID:  event.TenantID,
		Source:    "ai-query-engine",
		Payload:   payload,
	}

	return p.publishToRedpanda(ctx, TopicAIQueryAudits, &envelope, event.TenantID)
}

// publishToRedpanda sends an event to Redpanda with tenant-based partitioning
func (p *RedpandaAuditPublisher) publishToRedpanda(ctx context.Context, topic string, envelope *KafkaEventEnvelope, tenantID string) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("publisher is closed")
	}
	p.mu.RUnlock()

	// Serialize envelope to JSON
	payload, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal envelope: %w", err)
	}

	// Use tenant ID as key for partition assignment
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(tenantID),
		Value: payload,
		Headers: []kafka.Header{
			{Key: "event_id", Value: []byte(envelope.EventID)},
			{Key: "tenant_id", Value: []byte(tenantID)},
			{Key: "timestamp", Value: []byte(envelope.Timestamp.Format(time.RFC3339Nano))},
		},
	}

	// Send message synchronously
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	return nil
}

// Close closes the publisher and flushes pending messages
func (p *RedpandaAuditPublisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("failed to close Kafka writer: %w", err)
	}

	return nil
}
