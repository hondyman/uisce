package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"strings"

	"github.com/hondyman/semlayer/backend/internal/models"
	kafka "github.com/segmentio/kafka-go"
)

// ============================================================================
// COMMAND TYPES (For RabbitMQ Command Bus)
// ============================================================================

type CommandType string

const (
	// BO Commands
	CommandCreateBO CommandType = "command.bo.create"
	CommandUpdateBO CommandType = "command.bo.update"
	CommandDeleteBO CommandType = "command.bo.delete"
	CommandCloneBO  CommandType = "command.bo.clone"

	// Instance Commands
	CommandCreateInstance CommandType = "command.instance.create"
	CommandUpdateInstance CommandType = "command.instance.update"
	CommandDeleteInstance CommandType = "command.instance.delete"
)

// Command is the base structure for all commands
type Command struct {
	ID            string      `json:"id"`
	Type          CommandType `json:"type"`
	TenantID      string      `json:"tenant_id"`
	UserID        string      `json:"user_id"`
	Data          interface{} `json:"data"`
	Timestamp     time.Time   `json:"timestamp"`
	CorrelationID string      `json:"correlation_id"` // Track request through system
}

// ============================================================================
// EVENT TYPES (For Domain Events)
// ============================================================================

type EventType string

const (
	// BO Events
	EventBOCreated EventType = "event.bo.created"
	EventBOUpdated EventType = "event.bo.updated"
	EventBODeleted EventType = "event.bo.deleted"
	EventBOCloned  EventType = "event.bo.cloned"

	// Internal Event Events
	EventInternalEventCreated EventType = "event.internal_event.created"
	EventInternalEventUpdated EventType = "event.internal_event.updated"
	EventInternalEventDeleted EventType = "event.internal_event.deleted"

	// Instance Events
	EventInstanceCreated EventType = "event.instance.created"
	EventInstanceUpdated EventType = "event.instance.updated"
	EventInstanceDeleted EventType = "event.instance.deleted"

	// Workflow Events
	EventWorkflowStarted   EventType = "event.workflow.started"
	EventWorkflowProgress  EventType = "event.workflow.progress"
	EventWorkflowCompleted EventType = "event.workflow.completed"
	EventWorkflowFailed    EventType = "event.workflow.failed"
)

// Event represents a domain event (immutable fact)
type Event struct {
	ID            string                 `json:"id"`
	Type          EventType              `json:"type"`
	TenantID      string                 `json:"tenant_id"`
	EntityType    string                 `json:"entity_type"` // business_object, instance
	EntityID      string                 `json:"entity_id"`
	EntityKey     string                 `json:"entity_key"`
	Data          interface{}            `json:"data"`
	UserID        string                 `json:"user_id"`
	Timestamp     time.Time              `json:"timestamp"`
	CorrelationID string                 `json:"correlation_id"` // Link to command
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// BOEvent is an alias for backward compatibility
type BOEvent = Event

// ============================================================================
// EVENT PUBLISHER
// ============================================================================

// EventPublisher publishes events to Kafka (Redpanda)
type EventPublisher struct {
	writer  *kafka.Writer
	topic   string
	enabled bool
}

// NewEventPublisher creates a new Kafka-backed event publisher.
// Accepts either a Kafka brokers list (comma-separated) or, for legacy callers, an AMQP URL (deprecated).
func NewEventPublisher(brokersOrURL string) (*EventPublisher, error) {
	if brokersOrURL == "" {
		log.Println("⚠️  Event publisher not configured - events disabled")
		return &EventPublisher{enabled: false}, nil
	}

	// Detect legacy AMQP URL and disable (encourage migration)
	if strings.HasPrefix(brokersOrURL, "amqp://") {
		log.Printf("⚠️  Detected legacy AMQP URL %s - event publishing disabled. Set KAFKA_BROKERS instead.", brokersOrURL)
		return &EventPublisher{enabled: false}, nil
	}

	brokers := strings.Split(brokersOrURL, ",")
	w := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}

	log.Println("✅ Kafka writer initialized for event publishing")

	return &EventPublisher{
		writer:  w,
		topic:   "semlayer.bo",
		enabled: true,
	}, nil
}

// Publish publishes a generic event
func (ep *EventPublisher) publish(ctx context.Context, event *BOEvent) error {
	if !ep.enabled {
		return nil // Silently skip if not enabled
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	routingKey := fmt.Sprintf("%s.%s", event.EntityType, event.Type)
	msg := kafka.Message{
		Topic: ep.topic,
		Key:   []byte(routingKey),
		Value: data,
		Time:  time.Now(),
	}

	return ep.writer.WriteMessages(ctx, msg)
}

// PublishBOCreated publishes a BO creation event
func (ep *EventPublisher) PublishBOCreated(ctx context.Context, bo *models.BusinessObjectDefinition, userID string) {
	event := &BOEvent{
		ID:         bo.ID,
		Type:       EventBOCreated,
		TenantID:   bo.TenantID,
		EntityType: "business_object",
		EntityID:   bo.ID,
		EntityKey:  bo.Key,
		Data:       bo,
		UserID:     userID,
		Timestamp:  time.Now(),
	}
	if err := ep.publish(ctx, event); err != nil {
		log.Printf("Error publishing BO created event: %v", err)
	}
}

// PublishBOUpdated publishes a BO update event
func (ep *EventPublisher) PublishBOUpdated(ctx context.Context, bo *models.BusinessObjectDefinition, userID string) {
	event := &BOEvent{
		ID:         bo.ID,
		Type:       EventBOUpdated,
		TenantID:   bo.TenantID,
		EntityType: "business_object",
		EntityID:   bo.ID,
		EntityKey:  bo.Key,
		Data:       bo,
		UserID:     userID,
		Timestamp:  time.Now(),
	}
	if err := ep.publish(ctx, event); err != nil {
		log.Printf("Error publishing BO updated event: %v", err)
	}
}

// PublishBODeleted publishes a BO deletion event
func (ep *EventPublisher) PublishBODeleted(ctx context.Context, tenantID, boKey, userID string) {
	event := &BOEvent{
		ID:         fmt.Sprintf("%s-%s", tenantID, boKey),
		Type:       EventBODeleted,
		TenantID:   tenantID,
		EntityType: "business_object",
		EntityKey:  boKey,
		UserID:     userID,
		Timestamp:  time.Now(),
	}
	if err := ep.publish(ctx, event); err != nil {
		log.Printf("Error publishing BO deleted event: %v", err)
	}
}

// PublishBOCloned publishes a BO clone event
func (ep *EventPublisher) PublishBOCloned(ctx context.Context, clonedBO *models.BusinessObjectDefinition, sourceKey, userID string) {
	event := &BOEvent{
		ID:         clonedBO.ID,
		Type:       EventBOCloned,
		TenantID:   clonedBO.TenantID,
		EntityType: "business_object",
		EntityID:   clonedBO.ID,
		EntityKey:  clonedBO.Key,
		Data:       clonedBO,
		UserID:     userID,
		Timestamp:  time.Now(),
		Metadata: map[string]interface{}{
			"source_key": sourceKey,
		},
	}
	if err := ep.publish(ctx, event); err != nil {
		log.Printf("Error publishing BO cloned event: %v", err)
	}
}

// PublishInstanceCreated publishes an instance creation event
func (ep *EventPublisher) PublishInstanceCreated(ctx context.Context, instance *models.BusinessObjectInstance, userID string) {
	event := &BOEvent{
		ID:         instance.ID,
		Type:       EventInstanceCreated,
		TenantID:   instance.TenantID,
		EntityType: "instance",
		EntityID:   instance.ID,
		EntityKey:  instance.BusinessObjectKey,
		Data:       instance,
		UserID:     userID,
		Timestamp:  time.Now(),
	}
	if err := ep.publish(ctx, event); err != nil {
		log.Printf("Error publishing instance created event: %v", err)
	}
}

// PublishInstanceUpdated publishes an instance update event
func (ep *EventPublisher) PublishInstanceUpdated(ctx context.Context, instance *models.BusinessObjectInstance, userID string) {
	event := &BOEvent{
		ID:         instance.ID,
		Type:       EventInstanceUpdated,
		TenantID:   instance.TenantID,
		EntityType: "instance",
		EntityID:   instance.ID,
		EntityKey:  instance.BusinessObjectKey,
		Data:       instance,
		UserID:     userID,
		Timestamp:  time.Now(),
	}
	if err := ep.publish(ctx, event); err != nil {
		log.Printf("Error publishing instance updated event: %v", err)
	}
}

// PublishInstanceDeleted publishes an instance deletion event
func (ep *EventPublisher) PublishInstanceDeleted(ctx context.Context, tenantID, boKey, instanceID, userID string) {
	event := &BOEvent{
		ID:         instanceID,
		Type:       EventInstanceDeleted,
		TenantID:   tenantID,
		EntityType: "instance",
		EntityID:   instanceID,
		EntityKey:  boKey,
		UserID:     userID,
		Timestamp:  time.Now(),
	}
	if err := ep.publish(ctx, event); err != nil {
		log.Printf("Error publishing instance deleted event: %v", err)
	}
}

// PublishInternalEventCreated publishes an internal event creation event
func (ep *EventPublisher) PublishInternalEventCreated(ctx context.Context, event *models.InternalEvent, userID string) {
	evt := &BOEvent{
		ID:         event.ID.String(),
		Type:       EventInternalEventCreated,
		TenantID:   event.TenantID.String(),
		EntityType: "internal_event",
		EntityID:   event.ID.String(),
		Data:       event,
		UserID:     userID,
		Timestamp:  time.Now(),
	}
	if err := ep.publish(ctx, evt); err != nil {
		log.Printf("Error publishing internal event created: %v", err)
	}
}

// PublishInternalEventUpdated publishes an internal event update event
func (ep *EventPublisher) PublishInternalEventUpdated(ctx context.Context, event *models.InternalEvent, userID string) {
	evt := &BOEvent{
		ID:         event.ID.String(),
		Type:       EventInternalEventUpdated,
		TenantID:   event.TenantID.String(),
		EntityType: "internal_event",
		EntityID:   event.ID.String(),
		Data:       event,
		UserID:     userID,
		Timestamp:  time.Now(),
	}
	if err := ep.publish(ctx, evt); err != nil {
		log.Printf("Error publishing internal event updated: %v", err)
	}
}

// PublishInternalEventDeleted publishes an internal event deletion event
func (ep *EventPublisher) PublishInternalEventDeleted(ctx context.Context, eventID, tenantID, userID string) {
	evt := &BOEvent{
		ID:         eventID,
		Type:       EventInternalEventDeleted,
		TenantID:   tenantID,
		EntityType: "internal_event",
		EntityID:   eventID,
		UserID:     userID,
		Timestamp:  time.Now(),
	}
	if err := ep.publish(ctx, evt); err != nil {
		log.Printf("Error publishing internal event deleted: %v", err)
	}
}

// PublishWorkflowEvent publishes a workflow state event
func (ep *EventPublisher) PublishWorkflowEvent(ctx context.Context, eventType EventType, workflowID, tenantID, userID string, data map[string]interface{}) {
	event := &BOEvent{
		ID:         workflowID,
		Type:       eventType,
		TenantID:   tenantID,
		EntityType: "workflow",
		EntityID:   workflowID,
		Data:       data,
		UserID:     userID,
		Timestamp:  time.Now(),
	}
	if err := ep.publish(ctx, event); err != nil {
		log.Printf("Error publishing workflow event: %v", err)
	}
}

// Close closes the publisher connection
func (ep *EventPublisher) Close() error {
	if !ep.enabled {
		return nil
	}
	if ep.writer != nil {
		return ep.writer.Close()
	}
	return nil
}

// ============================================================================
// EVENT CONSUMER (For microservices)
// ============================================================================

// EventConsumer consumes events from Kafka topic `semlayer.bo`
type EventConsumer struct {
	reader  *kafka.Reader
	topic   string
	enabled bool
}

// NewEventConsumer creates a new event consumer
// Accepts either a Kafka brokers list or legacy AMQP URL (deprecated). If AMQP URL is provided, consumer is disabled.
func NewEventConsumer(brokersOrURL, groupID string) (*EventConsumer, error) {
	if brokersOrURL == "" {
		return &EventConsumer{enabled: false}, nil
	}

	// Detect legacy AMQP URL and disable (encourage migration)
	if strings.HasPrefix(brokersOrURL, "amqp://") {
		log.Printf("⚠️  Detected legacy AMQP URL %s - event consumer disabled. Set KAFKA_BROKERS instead.", brokersOrURL)
		return &EventConsumer{enabled: false}, nil
	}

	brokers := strings.Split(brokersOrURL, ",")
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    "semlayer.bo",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	return &EventConsumer{
		reader:  r,
		topic:   "semlayer.bo",
		enabled: true,
	}, nil
}

// Subscribe subscribes to events matching a pattern and starts consuming from Kafka
func (ec *EventConsumer) Subscribe(pattern string, handler func(*BOEvent) error) (<-chan *BOEvent, error) {
	if !ec.enabled {
		return nil, fmt.Errorf("event consumer not enabled")
	}

	events := make(chan *BOEvent, 100)

	go func() {
		defer close(events)
		for {
			m, err := ec.reader.FetchMessage(context.Background())
			if err != nil {
				// reader returns errors when context cancelled or broker issues
				continue
			}

			// Optional pattern matching on message key (simple prefix/# support)
			if pattern != "" {
				if strings.HasSuffix(pattern, "#") {
					prefix := strings.TrimSuffix(pattern, "#")
					if !strings.HasPrefix(string(m.Key), prefix) {
						ec.reader.CommitMessages(context.Background(), m)
						continue
					}
				} else if pattern != string(m.Key) {
					ec.reader.CommitMessages(context.Background(), m)
					continue
				}
			}

			var evt BOEvent
			if err := json.Unmarshal(m.Value, &evt); err != nil {
				log.Printf("Failed to unmarshal event: %v", err)
				ec.reader.CommitMessages(context.Background(), m)
				continue
			}

			// Call handler if provided
			if handler != nil {
				if err := handler(&evt); err != nil {
					log.Printf("Event handler error: %v", err)
				}
			}

			events <- &evt

			if err := ec.reader.CommitMessages(context.Background(), m); err != nil {
				log.Printf("failed to commit message: %v", err)
			}
		}
	}()

	return events, nil
}

// Close closes the consumer connection
func (ec *EventConsumer) Close() error {
	if !ec.enabled {
		return nil
	}
	if ec.reader != nil {
		return ec.reader.Close()
	}
	return nil
}
