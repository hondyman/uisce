//go:build ignore

// Example Avro consumer: Kafka + Schema Registry (Confluent wire format) -> Protobuf-like struct.
// Excluded from builds to avoid module churn; copy into your service and wire real imports.
package events

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/linkedin/goavro/v2"
	"github.com/riferrei/srclient"
)

// CatalogChangeEventPB mirrors your generated Protobuf type.
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

// Consumer reads Avro messages and decodes into CatalogChangeEventPB.
type Consumer struct {
	rd         *kafka.Consumer
	registry   *srclient.SchemaRegistryClient
	codecCache map[int]*goavro.Codec
}

func NewConsumer(brokers, schemaRegistryURL, groupID string, topics []string) (*Consumer, error) {
	rd, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  brokers,
		"group.id":           groupID,
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false,
	})
	if err != nil {
		return nil, err
	}
	if err := rd.SubscribeTopics(topics, nil); err != nil {
		return nil, err
	}

	reg := srclient.CreateSchemaRegistryClient(schemaRegistryURL)

	return &Consumer{
		rd:         rd,
		registry:   reg,
		codecCache: make(map[int]*goavro.Codec),
	}, nil
}

func (c *Consumer) Close() {
	_ = c.rd.Close()
}

// Poll blocks until a message is received or context is done.
func (c *Consumer) Poll(ctx context.Context) (*CatalogChangeEventPB, error) {
	for {
		ev := c.rd.Poll(500)
		if ev == nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			continue
		}
		switch m := ev.(type) {
		case *kafka.Message:
			return c.decode(m.Value)
		case kafka.Error:
			return nil, m
		}
	}
}

func (c *Consumer) decode(payload []byte) (*CatalogChangeEventPB, error) {
	if len(payload) < 5 || payload[0] != 0 {
		return nil, fmt.Errorf("invalid confluent wire format")
	}
	schemaID := int(binary.BigEndian.Uint32(payload[1:5]))

	codec, ok := c.codecCache[schemaID]
	if !ok {
		schema, err := c.registry.GetSchema(schemaID)
		if err != nil {
			return nil, fmt.Errorf("schema %d: %w", schemaID, err)
		}
		codec, err = goavro.NewCodec(schema.Schema())
		if err != nil {
			return nil, fmt.Errorf("codec: %w", err)
		}
		c.codecCache[schemaID] = codec
	}

	native, _, err := codec.NativeFromBinary(payload[5:])
	if err != nil {
		return nil, fmt.Errorf("decode avro: %w", err)
	}
	rec := native.(map[string]any)

	toStr := func(k string) string {
		if v, ok := rec[k]; ok && v != nil {
			return v.(string)
		}
		return ""
	}
	toMap := func(k string) map[string]string {
		raw, ok := rec[k]
		if !ok || raw == nil {
			return nil
		}
		out := make(map[string]string)
		for key, val := range raw.(map[string]any) {
			out[key] = val.(string)
		}
		return out
	}

	ms, _ := rec["occurredAt"].(int64)
	occurred := time.Unix(0, ms*int64(time.Millisecond)).UTC().Format(time.RFC3339Nano)

	return &CatalogChangeEventPB{
		EventId:    toStr("eventId"),
		EntityType: toStr("entityType"),
		ChangeType: toStr("changeType"),
		TenantId:   toStr("tenantId"),
		OccurredAt: occurred,
		Before:     toMap("before"),
		After:      toMap("after"),
		Source:     toStr("source"),
	}, nil
}
