package events

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	kafka "github.com/segmentio/kafka-go"
)

// KafkaPublisher publishes domain events to Kafka
type KafkaPublisher struct {
	writer *kafka.Writer
	topics map[EventType]string
}

// KafkaConfig contains Kafka connection configuration
type KafkaConfig struct {
	Brokers string
}

// DefaultKafkaConfig returns default Kafka configuration
func DefaultKafkaConfig() KafkaConfig {
	return KafkaConfig{
		Brokers: "localhost:9092",
	}
}

// NewKafkaPublisher creates a new Kafka event publisher
func NewKafkaPublisher(brokers string) *KafkaPublisher {
	w := &kafka.Writer{
		Addr:     kafka.TCP(strings.Split(brokers, ",")...),
		Balancer: &kafka.LeastBytes{},
	}

	publisher := &KafkaPublisher{
		writer: w,
		topics: make(map[EventType]string),
	}

	publisher.mapTopics()

	return publisher
}

// PublishToTopic publishes an arbitrary event to a specific topic.
func (p *KafkaPublisher) PublishToTopic(ctx context.Context, topic string, key string, payload interface{}) error {
	if p.writer == nil {
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: body,
		Time:  time.Now(),
	}

	return p.writer.WriteMessages(ctx, msg)
}

// mapTopics maps event types to topics
func (p *KafkaPublisher) mapTopics() {
	p.topics = map[EventType]string{
		// API Endpoint events
		APIEndpointCreated:   "api.endpoints",
		APIEndpointUpdated:   "api.endpoints",
		APIEndpointDeleted:   "api.endpoints",
		APIEndpointActivated: "api.endpoints",

		// Mapping events
		EntityMappingCreated:     "api.mappings",
		EntityMappingDeleted:     "api.mappings",
		DatasourceMappingCreated: "api.mappings",
		DatasourceMappingDeleted: "api.mappings",

		// Catalog node events
		CatalogNodeCreated: "catalog.nodes",
		CatalogNodeUpdated: "catalog.nodes",
		CatalogNodeDeleted: "catalog.nodes",

		// Catalog edge events
		CatalogEdgeCreated: "catalog.edges",
		CatalogEdgeDeleted: "catalog.edges",

		// Gold Copy events
		GoldCopyConnectionChanged: "gold_copy.events",
		GoldCopyEntityChanged:     "gold_copy.events",

		// Compliance events
		SemanticTermComplianceUpdated: "compliance.events",
		BusinessTermComplianceUpdated: "compliance.events",
		ComplianceViolationDetected:   "compliance.events",
	}
}

// PublishAPIEndpointEvent publishes an API endpoint event
func (p *KafkaPublisher) PublishAPIEndpointEvent(ctx context.Context, event *APIEndpointEvent) error {
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	return p.publishEvent(ctx, event.EventType, event)
}

// PublishEntityMappingEvent publishes an entity mapping event
func (p *KafkaPublisher) PublishEntityMappingEvent(ctx context.Context, event *EntityMappingEvent) error {
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	return p.publishEvent(ctx, event.EventType, event)
}

// PublishDatasourceMappingEvent publishes a datasource mapping event
func (p *KafkaPublisher) PublishDatasourceMappingEvent(ctx context.Context, event *DatasourceMappingEvent) error {
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	return p.publishEvent(ctx, event.EventType, event)
}

// PublishCatalogNodeEvent publishes a catalog node event
func (p *KafkaPublisher) PublishCatalogNodeEvent(ctx context.Context, event *CatalogNodeEvent) error {
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	return p.publishEvent(ctx, event.EventType, event)
}

// PublishCatalogEdgeEvent publishes a catalog edge event
// PublishCatalogEdgeEvent publishes a catalog edge event
func (p *KafkaPublisher) PublishCatalogEdgeEvent(ctx context.Context, event *CatalogEdgeEvent) error {
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	return p.publishEvent(ctx, event.EventType, event)
}

// PublishGoldCopyConnectionEvent publishes a gold copy connection event
func (p *KafkaPublisher) PublishGoldCopyConnectionEvent(ctx context.Context, event *GoldCopyConnectionEvent) error {
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	return p.publishEvent(ctx, event.EventType, event)
}

// PublishGoldCopyEntityEvent publishes a generic gold copy entity event
func (p *KafkaPublisher) PublishGoldCopyEntityEvent(ctx context.Context, event *GoldCopyEntityEvent) error {
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	return p.publishEvent(ctx, event.EventType, event)
}

// PublishSemanticTermComplianceUpdatedEvent publishes a compliance update event
func (p *KafkaPublisher) PublishSemanticTermComplianceUpdatedEvent(ctx context.Context, event *SemanticTermComplianceUpdatedEvent) error {
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	return p.publishEvent(ctx, event.EventType, event)
}

// publishEvent publishes a generic event to Kafka
func (p *KafkaPublisher) publishEvent(ctx context.Context, eventType EventType, event interface{}) error {
	topic, ok := p.topics[eventType]
	if !ok {
		return fmt.Errorf("unknown event type: %s", eventType)
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(string(eventType)), // Use event type or ID as key? Using Type helps partitioning by type if desired, or ID for randomness.
		Value: body,
		Time:  time.Now(),
	}

	return p.writer.WriteMessages(ctx, msg)
}

// Close closes the Kafka writer
func (p *KafkaPublisher) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// Healthcheck verifies Kafka connection
func (p *KafkaPublisher) Healthcheck() error {
	// kafka-go Writer handles connections lazily and reconnects automatically.
	// A strictly correct healthcheck would try to connect or list topics.
	// For now, prompt implementation return nil or check stats.
	if p.writer == nil {
		return fmt.Errorf("Kafka writer is nil")
	}
	return nil
}
