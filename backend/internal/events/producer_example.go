//go:build ignore

// Example Avro producer: Protobuf in-memory -> Avro on Kafka with Schema Registry (Confluent wire format).
// Excluded from builds to avoid module import churn; copy into your service and wire real imports.
package events

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/linkedin/goavro/v2"
	"github.com/riferrei/srclient"
)

// CatalogChangeEventPB is a minimal stand-in for your generated Protobuf type.
// Replace this with your real catalogpb.CatalogChangeEvent.
type CatalogChangeEventPB struct {
	EventId    string
	EntityType string
	ChangeType string // "insert"|"update"|"delete"
	TenantId   string
	OccurredAt string // RFC3339
	Before     map[string]string
	After      map[string]string
	Source     string
}

// Producer publishes Avro-encoded events to Kafka using Schema Registry.
type Producer struct {
	writer    *kafka.Producer
	schema    *srclient.Schema
	registry  *srclient.SchemaRegistryClient
	subject   string
	avroCodec *goavro.Codec
}

// NewProducer builds a producer, auto-registering the Avro schema if missing.
func NewProducer(brokers, schemaRegistryURL, subject, avroSchema string) (*Producer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
		"acks":              "all",
	})
	if err != nil {
		return nil, err
	}

	reg := srclient.CreateSchemaRegistryClient(schemaRegistryURL)

	schema, err := reg.GetLatestSchema(subject)
	if err != nil {
		// Attempt to register if not found
		schema, err = reg.CreateSchema(subject, avroSchema, srclient.Avro)
		if err != nil {
			return nil, fmt.Errorf("schema lookup/register failed: %w", err)
		}
	}

	codec, err := goavro.NewCodec(schema.Schema())
	if err != nil {
		return nil, fmt.Errorf("build codec: %w", err)
	}

	return &Producer{
		writer:    p,
		schema:    schema,
		registry:  reg,
		subject:   subject,
		avroCodec: codec,
	}, nil
}

// Close flushes and closes the producer.
func (p *Producer) Close() {
	_ = p.writer.Flush(5000)
	p.writer.Close()
}

// Publish writes a single event. It uses Confluent wire format: 0 | schemaID | avroPayload.
func (p *Producer) Publish(ctx context.Context, topic string, evt *CatalogChangeEventPB) error {
	native, err := toAvroNative(evt)
	if err != nil {
		return err
	}

	bin, err := p.avroCodec.BinaryFromNative(nil, native)
	if err != nil {
		return err
	}

	var payload bytes.Buffer
	payload.WriteByte(0)
	if err := binary.Write(&payload, binary.BigEndian, uint32(p.schema.ID())); err != nil {
		return err
	}
	payload.Write(bin)

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            []byte(evt.EntityType + ":" + evt.TenantId),
		Value:          payload.Bytes(),
		Timestamp:      time.Now().UTC(),
	}

	delivery := make(chan kafka.Event, 1)
	if err := p.writer.Produce(msg, delivery); err != nil {
		return err
	}
	e := <-delivery
	if m, ok := e.(*kafka.Message); ok {
		if m.TopicPartition.Error != nil {
			return m.TopicPartition.Error
		}
	}
	return nil
}

func toAvroNative(evt *CatalogChangeEventPB) (map[string]any, error) {
	change := evt.ChangeType
	if change != "insert" && change != "update" && change != "delete" {
		return nil, fmt.Errorf("invalid changeType: %s", change)
	}

	ts, err := time.Parse(time.RFC3339Nano, evt.OccurredAt)
	if err != nil {
		return nil, fmt.Errorf("parse occurredAt: %w", err)
	}

	return map[string]any{
		"eventId":    evt.EventId,
		"entityType": evt.EntityType,
		"changeType": change,
		"tenantId":   evt.TenantId,
		"occurredAt": ts.UnixMilli(),
		"before":     toStringMap(evt.Before),
		"after":      toStringMap(evt.After),
		"source":     evt.Source,
	}, nil
}

func toStringMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	return m
}
