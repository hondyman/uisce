package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// ErrAuditPublisherUnavailable is returned by the Recorder when no
// AuditPublisher is wired. Cardinal Rule 7 (Security Mandate) treats this as
// a denial-pending condition: any denial surfaced without an audit publish
// is a compliance violation.
var ErrAuditPublisherUnavailable = errors.New("audit publisher unavailable")

// AuditPublisher defines the interface for publishing audit events.
// Cardinal Rule 7: every Publish* method below is contractually synchronous
// — it MUST NOT return until Redpanda has acknowledged the message with
// RequiredAcks=RequireAll. Implementations that switch to Async MUST also
// remove the synchronous-emission guarantee from callers.
type AuditPublisher interface {
	// Workflow / orchestration events
	PublishJobRun(ctx context.Context, event JobRunCompletedEvent) error
	PublishDAGRun(ctx context.Context, event interface{}) error
	PublishChangeSet(ctx context.Context, event ChangeSetCreatedEvent) error
	PublishSemanticSnapshot(ctx context.Context, event SemanticSnapshotEvent) error
	PublishOrchestrationEvent(ctx context.Context, event OrchestrationWorkflowEvent) error
	PublishComplianceViolation(ctx context.Context, event ComplianceViolationEvent) error
	PublishAIQueryAudit(ctx context.Context, event AIQueryExecutionEvent) error

	// AI Gate events (Phase B — Cardinal Rule 7)
	PublishAIQueryGenerated(ctx context.Context, event AIQueryGeneratedEvent) error
	PublishAISemanticResolved(ctx context.Context, event AISemanticResolvedEvent) error
	PublishAIColumnMasked(ctx context.Context, event AIColumnMaskedEvent) error
	PublishAIABACEvaluated(ctx context.Context, event AIABACEvaluatedEvent) error
	PublishAIABACDenied(ctx context.Context, event AIABACDeniedEvent) error
	PublishAILineageResolved(ctx context.Context, event AILineageResolvedEvent) error

	// Catalog mutation events (B5; also feeds Phase A cache invalidation)
	PublishCatalogBOMutated(ctx context.Context, event CatalogBOMutatedEvent) error

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

// =============================================================================
// AI Gate Publishers (Phase B — Cardinal Rule 7)
// =============================================================================
//
// Each PublishAI* method below routes to TopicAIGate (or TopicAIDenials for
// denials, TopicCatalogMutations for catalog mutations). Cardinal Rule 7
// mandates synchronous, RequireAll delivery — this is already enforced by
// the underlying kafka.Writer configured in NewRedpandaAuditPublisher.
//
// Cardinal Rule 6 mandates that the partition key is the tenant ID so all
// of a tenant's audit events land in the same partition (preserving order).

// PublishAIQueryGenerated publishes a Cardinal-Rule-7 event for query generation.
func (p *RedpandaAuditPublisher) PublishAIQueryGenerated(ctx context.Context, event AIQueryGeneratedEvent) error {
	payload, err := MarshalEnvelopeJSON(ctx, EventTypeAIQueryGenerated, event)
	if err != nil {
		return fmt.Errorf("failed to marshal AI query generated envelope: %w", err)
	}
	envelope := KafkaEventEnvelope{
		EventID:   uuid.New().String(),
		EventType: EventTypeAIQueryGenerated,
		Version:   "1.0",
		Timestamp: time.Now().UTC(),
		TenantID:  event.TenantID,
		Source:    "ai-gate",
		Payload:   payload,
	}
	return p.publishToRedpanda(ctx, TopicAIGate, &envelope, event.TenantID)
}

// PublishAISemanticResolved publishes when a semantic request is resolved.
func (p *RedpandaAuditPublisher) PublishAISemanticResolved(ctx context.Context, event AISemanticResolvedEvent) error {
	payload, err := MarshalEnvelopeJSON(ctx, EventTypeAISemanticResolved, event)
	if err != nil {
		return fmt.Errorf("failed to marshal AI semantic resolved envelope: %w", err)
	}
	envelope := KafkaEventEnvelope{
		EventID:   uuid.New().String(),
		EventType: EventTypeAISemanticResolved,
		Version:   "1.0",
		Timestamp: time.Now().UTC(),
		TenantID:  event.TenantID,
		Source:    "ai-gate",
		Payload:   payload,
	}
	return p.publishToRedpanda(ctx, TopicAIGate, &envelope, event.TenantID)
}

// PublishAIColumnMasked publishes one event per PII/PHI/PCI column that was masked.
func (p *RedpandaAuditPublisher) PublishAIColumnMasked(ctx context.Context, event AIColumnMaskedEvent) error {
	payload, err := MarshalEnvelopeJSON(ctx, EventTypeAIColumnMasked, event)
	if err != nil {
		return fmt.Errorf("failed to marshal AI column masked envelope: %w", err)
	}
	envelope := KafkaEventEnvelope{
		EventID:   uuid.New().String(),
		EventType: EventTypeAIColumnMasked,
		Version:   "1.0",
		Timestamp: time.Now().UTC(),
		TenantID:  event.TenantID,
		Source:    "ai-gate",
		Payload:   payload,
	}
	return p.publishToRedpanda(ctx, TopicAIGate, &envelope, event.TenantID)
}

// PublishAIABACEvaluated publishes every ABAC evaluation outcome (incl. allow).
func (p *RedpandaAuditPublisher) PublishAIABACEvaluated(ctx context.Context, event AIABACEvaluatedEvent) error {
	payload, err := MarshalEnvelopeJSON(ctx, EventTypeAIABACEvaluated, event)
	if err != nil {
		return fmt.Errorf("failed to marshal AI ABAC evaluated envelope: %w", err)
	}
	envelope := KafkaEventEnvelope{
		EventID:   uuid.New().String(),
		EventType: EventTypeAIABACEvaluated,
		Version:   "1.0",
		Timestamp: time.Now().UTC(),
		TenantID:  event.TenantID,
		Source:    "ai-gate",
		Payload:   payload,
	}
	return p.publishToRedpanda(ctx, TopicAIGate, &envelope, event.TenantID)
}

// PublishAIABACDenied publishes a synchronous ABAC denial event to the
// dedicated denials topic (separate from the general AI Gate topic so SIEM/SOC
// tooling can subscribe to denials alone). Cardinal Rule 7:
// callers MUST NOT return the user-facing denial until this returns nil.
//
// The event payload is annotated with EmittedSync=true BEFORE marshalling so
// any downstream consumer reading from Redpanda can rely on the field as
// authoritative proof of synchronous emission.
func (p *RedpandaAuditPublisher) PublishAIABACDenied(ctx context.Context, event AIABACDeniedEvent) error {
	// Cardinal Rule 7: every persisted denial record carries EmittedSync=true.
	event.EmittedSync = true
	payload, err := MarshalEnvelopeJSON(ctx, EventTypeAIABACDenied, event)
	if err != nil {
		return fmt.Errorf("failed to marshal AI ABAC denied envelope: %w", err)
	}
	envelope := KafkaEventEnvelope{
		EventID:   uuid.New().String(),
		EventType: EventTypeAIABACDenied,
		Version:   "1.0",
		Timestamp: time.Now().UTC(),
		TenantID:  event.TenantID,
		Source:    "ai-gate",
		Payload:   payload,
	}
	// Denials go on a dedicated topic for SOC alerting.
	return p.publishToRedpanda(ctx, TopicAIDenials, &envelope, event.TenantID)
}

// PublishAILineageResolved publishes when ResolveGraphGovernanceContext traces lineage.
func (p *RedpandaAuditPublisher) PublishAILineageResolved(ctx context.Context, event AILineageResolvedEvent) error {
	payload, err := MarshalEnvelopeJSON(ctx, EventTypeAILineageResolved, event)
	if err != nil {
		return fmt.Errorf("failed to marshal AI lineage resolved envelope: %w", err)
	}
	envelope := KafkaEventEnvelope{
		EventID:   uuid.New().String(),
		EventType: EventTypeAILineageResolved,
		Version:   "1.0",
		Timestamp: time.Now().UTC(),
		TenantID:  event.TenantID,
		Source:    "ai-gate",
		Payload:   payload,
	}
	return p.publishToRedpanda(ctx, TopicAIGate, &envelope, event.TenantID)
}

// PublishCatalogBOMutated publishes a catalog BO mutation to TopicCatalogMutations.
// Phase A's cmd/cache-invalidator consumer reads this topic and drains Redis keys.
func (p *RedpandaAuditPublisher) PublishCatalogBOMutated(ctx context.Context, event CatalogBOMutatedEvent) error {
	payload, err := MarshalEnvelopeJSON(ctx, EventTypeCatalogBOMutated, event)
	if err != nil {
		return fmt.Errorf("failed to marshal catalog BO mutated envelope: %w", err)
	}
	envelope := KafkaEventEnvelope{
		EventID:   uuid.New().String(),
		EventType: EventTypeCatalogBOMutated,
		Version:   "1.0",
		Timestamp: time.Now().UTC(),
		TenantID:  event.TenantID,
		Source:    "catalog",
		Payload:   payload,
	}
	return p.publishToRedpanda(ctx, TopicCatalogMutations, &envelope, event.TenantID)
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
