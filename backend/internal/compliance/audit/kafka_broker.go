package audit

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaBroker implements MessageBroker using segmentio/kafka-go with strict acks=all
type KafkaBroker struct {
	writers map[string]*kafka.Writer
	brokers []string
}

// NewKafkaBroker creates a new Kafka/Redpanda broker client with RequiredAcks: kafka.RequireAll (acks=all)
func NewKafkaBroker(brokers []string) *KafkaBroker {
	if len(brokers) == 0 {
		brokers = []string{"localhost:9092"}
	}
	return &KafkaBroker{
		writers: make(map[string]*kafka.Writer),
		brokers: brokers,
	}
}

// Publish writes a message to Redpanda with strict acks=all (RequireAll)
func (b *KafkaBroker) Publish(ctx context.Context, topic string, key string, payload []byte) error {
	w, exists := b.writers[topic]
	if !exists {
		transport := &kafka.Transport{
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(address)
				if err == nil && (host == "uisce-redpanda" || host == "redpanda") && len(b.brokers) > 0 {
					brokerHost, _, _ := net.SplitHostPort(b.brokers[0])
					if brokerHost != "" {
						address = net.JoinHostPort(brokerHost, port)
					}
				}
				dialer := &net.Dialer{Timeout: 5 * time.Second}
				return dialer.DialContext(ctx, network, address)
			},
		}

		w = &kafka.Writer{
			Addr:                   kafka.TCP(b.brokers...),
			Topic:                  topic,
			Balancer:               &kafka.Hash{},
			RequiredAcks:           kafka.RequireAll, // acks=all: Leader + all in-sync replicas must acknowledge
			MaxAttempts:            5,
			BatchTimeout:           10 * time.Millisecond,
			Async:                  false, // Synchronous hot path produce
			AllowAutoTopicCreation: true,
			Transport:              transport,
		}
		b.writers[topic] = w
	}

	msg := kafka.Message{
		Key:   []byte(key),
		Value: payload,
		Time:  time.Now().UTC(),
	}

	return w.WriteMessages(ctx, msg)
}

// Subscribe listens to a topic and invokes handler for each message
func (b *KafkaBroker) Subscribe(ctx context.Context, topic string, groupID string, handler func(key string, payload []byte) error) error {
	dialer := &kafka.Dialer{
		Timeout: 5 * time.Second,
		DualStack: true,
	}
	if len(b.brokers) > 0 {
		brokerHost, _, _ := net.SplitHostPort(b.brokers[0])
		if brokerHost != "" {
			dialer.Resolver = &customResolver{brokerHost: brokerHost}
		}
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        b.brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: 1 * time.Second,
		StartOffset:    kafka.FirstOffset,
		Dialer:         dialer,
	})
	defer r.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := r.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return fmt.Errorf("fetch kafka message: %w", err)
			}

			if err := handler(string(msg.Key), msg.Value); err != nil {
				return fmt.Errorf("process message: %w", err)
			}

			if err := r.CommitMessages(ctx, msg); err != nil {
				return fmt.Errorf("commit message: %w", err)
			}
		}
	}
}

type customResolver struct {
	brokerHost string
}

func (c *customResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	if host == "uisce-redpanda" || host == "redpanda" {
		return []string{c.brokerHost}, nil
	}
	return net.DefaultResolver.LookupHost(ctx, host)
}

// Close closes all active writers
func (b *KafkaBroker) Close() error {
	for _, w := range b.writers {
		_ = w.Close()
	}
	return nil
}
