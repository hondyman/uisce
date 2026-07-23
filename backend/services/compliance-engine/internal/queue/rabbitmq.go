package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

// KafkaClient wraps Kafka writer and reader
type KafkaClient struct {
	writer  *kafka.Writer
	brokers []string
}

// NewKafkaClient creates a new Kafka client
func NewKafkaClient(brokers string) (*KafkaClient, error) {
	brokerList := strings.Split(brokers, ",")
	w := &kafka.Writer{
		Addr:     kafka.TCP(brokerList...),
		Balancer: &kafka.LeastBytes{},
	}

	return &KafkaClient{
		writer:  w,
		brokers: brokerList,
	}, nil
}

// DeclareTopology is a no-op for Kafka/Redpanda (auto-create topics)
// or could explicitly create topics if needed.
func (kc *KafkaClient) DeclareTopology() error {
	// Optional: verify connection or topics exist
	return nil
}

// PublishEvent publishes an event to a Kafka topic based on routing key prefix logic
// In AMQP: exchange="compliance.exchange", routingKey="trade.created.foo"
// In Kafka: we map routing key patterns to Topics.
func (kc *KafkaClient) PublishEvent(ctx context.Context, routingKey string, payload interface{}) error {
	topic := "compliance.events" // Default fallback

	if strings.HasPrefix(routingKey, "trade.created") {
		topic = "compliance.trades"
	} else if strings.HasPrefix(routingKey, "audit") {
		topic = "compliance.audit"
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(routingKey),
		Value: body,
		Time:  time.Now(),
	}

	return kc.writer.WriteMessages(ctx, msg)
}

// Consume starts consuming messages from a specific topic mapping
// queueName logic from AMQP needs mapping:
// "q.compliance.post_trade" -> topic "compliance.trades"
// "q.audit.starrocks" -> topic "compliance.audit"
func (kc *KafkaClient) Consume(queueName string, consumerID string) (*kafka.Reader, error) {
	topic := ""
	switch queueName {
	case "q.compliance.post_trade":
		topic = "compliance.trades"
	case "q.audit.starrocks":
		topic = "compliance.audit"
	default:
		return nil, fmt.Errorf("unknown queue mapping for %s", queueName)
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  kc.brokers,
		GroupID:  consumerID, // e.g. "compliance-post-trade-worker"
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	return r, nil
}

// Close closes the writer
func (kc *KafkaClient) Close() error {
	if kc.writer != nil {
		return kc.writer.Close()
	}
	return nil
}

// IsConnected checks if writer is instantiated
func (kc *KafkaClient) IsConnected() bool {
	return kc.writer != nil
}
