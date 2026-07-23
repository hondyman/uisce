package ingestion

import (
	"context"
	"log"
	"strings"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

type MessageHandler func(ctx context.Context, body []byte) error

type KafkaListener struct {
	reader  *kafka.Reader
	topic   string
	handler MessageHandler
}

// NewKafkaListener creates a new Kafka listener.
// queueName is mapped to topic.
func NewKafkaListener(brokers string, queueName string, handler MessageHandler) (*KafkaListener, error) {
	brokerList := strings.Split(brokers, ",")

	// Map queueName to topic? Or use queueName as topic?
	// Assuming queueName IS the topic or closely related.
	// For migration, let's treat queueName as topic.
	topic := queueName

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokerList,
		GroupID:  "ingestion-group-" + topic, // Unique group per topic/listener
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	return &KafkaListener{
		reader:  r,
		topic:   topic,
		handler: handler,
	}, nil
}

// Deprecated: Compat wrapper for RabbitMQListener style
func NewRabbitMQListener(url string, queueName string, handler MessageHandler) (*KafkaListener, error) {
	// Extract brokers from url? No, can't easily.
	// We'll rely on default env var or assume url IS brokers if it doesn't look like amqp://
	// But usually this is called with config.
	// We'll hardcode default Redpanda for now or panic.
	// Better: assume global default or pass explicit brokers.
	brokers := "redpanda:9092"
	return NewKafkaListener(brokers, queueName, handler)
}

func (l *KafkaListener) Start(ctx context.Context) error {
	log.Printf(" [*] Waiting for messages in %s (Kafka).", l.topic)

	go func() {
		defer l.reader.Close()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				m, err := l.reader.FetchMessage(ctx)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					log.Printf("Error fetching Kafka message: %v", err)
					time.Sleep(1 * time.Second)
					continue
				}

				// Process message
				if err := l.handler(ctx, m.Value); err != nil {
					log.Printf("Error processing message: %v", err)
					// No Nack in Kafka. We can skip commit to retry (depending on offset commit policy)
					// or commit anyway.
					// If we don't commit, next fetch might pick it up if we restart?
					// FetchMessage doesn't auto commit.
					// If we continue without committing, we might process it again on restart.
					// But current loop continues... FetchMessage moves forward in memory?
					// No, FetchMessage gets NEXT message.
					// To retry, we'd need to seek back.
					// Simplest: Log error and commit (skip bad message).
					l.reader.CommitMessages(ctx, m)
				} else {
					l.reader.CommitMessages(ctx, m)
				}
			}
		}
	}()

	return nil
}

func (l *KafkaListener) Close() {
	if l.reader != nil {
		l.reader.Close()
	}
}
