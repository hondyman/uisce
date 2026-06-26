package catalogsync

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/linkedin/goavro/v2"
	"github.com/riferrei/srclient"
	kafka "github.com/segmentio/kafka-go"
)

const confluentWirePrefix = 0

// AvroPublisher writes catalog change events to Kafka using Schema Registry.
type AvroPublisher struct {
	writer       *kafka.Writer
	registry     *srclient.SchemaRegistryClient
	subject      string
	codecCacheID int
	codec        *goavro.Codec
}

// NoopPublisher is used when Kafka/SchemaRegistry is not available
type NoopPublisher struct{}

func (p *NoopPublisher) Publish(ctx context.Context, event CatalogChangeEvent) error {
	return nil
}
func (p *NoopPublisher) Close(ctx context.Context) error {
	return nil
}

// NewAvroPublisher configures a Kafka writer and Schema Registry client.
func NewAvroPublisher(brokers []string, topic, registryURL, subject string) (*AvroPublisher, error) {
	client := srclient.CreateSchemaRegistryClient(registryURL)
	// Try a ping/fetch to ensure it's available
	_, err := client.GetLatestSchema(subject)
	if err != nil {
		return nil, fmt.Errorf("schema registry check failed: %w", err)
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
	}
	return &AvroPublisher{writer: w, registry: client, subject: subject}, nil
}

func (p *AvroPublisher) ensureCodec(ctx context.Context) error {
	if p.codec != nil {
		return nil
	}
	schema, err := p.registry.GetLatestSchema(p.subject)
	if err != nil {
		return fmt.Errorf("fetch schema: %w", err)
	}
	codec, err := goavro.NewCodec(schema.Schema())
	if err != nil {
		return fmt.Errorf("build codec: %w", err)
	}
	p.codec = codec
	p.codecCacheID = schema.ID()
	return nil
}

// Publish encodes the event to Avro (Confluent wire format) and writes to Kafka.
func (p *AvroPublisher) Publish(ctx context.Context, event CatalogChangeEvent) error {
	if err := p.ensureCodec(ctx); err != nil {
		return err
	}

	native := map[string]any{
		"eventId":    event.EventID,
		"entityType": event.EntityType,
		"changeType": string(event.ChangeType),
		"tenantId":   event.TenantID,
		"occurredAt": event.OccurredAt.UnixMilli(),
		"before":     event.Before,
		"after":      event.After,
		"source":     event.Source,
	}

	binaryValue, err := p.codec.BinaryFromNative(nil, native)
	if err != nil {
		return fmt.Errorf("encode avro: %w", err)
	}

	payload := encodeConfluent(p.codecCacheID, binaryValue)
	msg := kafka.Message{
		Key:   []byte(event.EntityType + ":" + event.TenantID),
		Value: payload,
		Time:  time.Now(),
	}

	return p.writer.WriteMessages(ctx, msg)
}

func (p *AvroPublisher) Close(ctx context.Context) error {
	return p.writer.Close()
}

func encodeConfluent(schemaID int, avroPayload []byte) []byte {
	// Confluent framing: magic byte (0) + 4-byte schema ID + avro payload.
	buffer := make([]byte, 5+len(avroPayload))
	buffer[0] = confluentWirePrefix
	binary.BigEndian.PutUint32(buffer[1:5], uint32(schemaID))
	copy(buffer[5:], avroPayload)
	return buffer
}
