package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"strings"

	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/services"
	kafka "github.com/segmentio/kafka-go"

	"github.com/prometheus/client_golang/prometheus"
)

// noopEventBus is a minimal EventBus implementation used in local/dev runs.
type noopEventBus struct{}

func (n *noopEventBus) Emit(ctx context.Context, event string, data interface{}) error {
	return nil
}

// notificationAdapter adapts the EngagementNotificationService to the
// NotificationService interface expected by the TriggerEngine.
type notificationAdapter struct {
	svc *services.EngagementNotificationService
}

func (a *notificationAdapter) Send(ctx context.Context, channel string, payload *NotificationPayload) error {
	userID := ""
	if len(payload.Recipients) > 0 {
		userID = payload.Recipients[0]
	}

	notification := &models.EngagementNotification{
		UserID:    userID,
		Title:     payload.Subject,
		Message:   payload.Body,
		Channels:  []string{channel},
		CreatedBy: "system",
	}

	if err := a.svc.CreateNotification(ctx, notification); err != nil {
		return err
	}
	return a.svc.SendNotification(ctx, notification.ID)
}

// AMQPEventBus is a compatibility shim that publishes events to Kafka/Redpanda.
// The name is retained for backwards compatibility with older code that expects an AMQP-based bus.
// Prefer using KafkaPublisher or a more explicitly-named KafkaEventBus in new code.
// If a legacy AMQP URL (amqp://) is provided, the bus will be disabled and Emit becomes a no-op.
type AMQPEventBus struct {
	writer   *kafka.Writer
	exchange string
}

// Metrics for AMQP operations
var (
	amqpConnectionAttempts int64
	amqpConnectionFailures int64
	amqpPublishedCount     int64
	amqpPublishFailures    int64
)

// Prometheus collectors
var (
	promAMQPPublishCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "amqp_publish_total", Help: "Total AMQP publish attempts by status"},
		[]string{"status"},
	)
	promAMQPPublishDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{Name: "amqp_publish_duration_ms", Help: "AMQP publish latency in ms", Buckets: prometheus.ExponentialBuckets(1, 2, 10)},
	)
	promAMQPConnectionAttempts = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "amqp_connection_attempts_total", Help: "Total AMQP connection attempts"},
	)
	promAMQPConnectionFailures = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "amqp_connection_failures_total", Help: "Total AMQP connection failures"},
	)
)

// GetAMQPMetrics returns a snapshot of AMQP metrics.
func GetAMQPMetrics() map[string]int64 {
	return map[string]int64{
		"connection_attempts": atomic.LoadInt64(&amqpConnectionAttempts),
		"connection_failures": atomic.LoadInt64(&amqpConnectionFailures),
		"published_count":     atomic.LoadInt64(&amqpPublishedCount),
		"publish_failures":    atomic.LoadInt64(&amqpPublishFailures),
	}
}

func init() {
	// Register prometheus collectors. If registration fails due to already
	// registered collectors, ignore the error to keep init idempotent in tests.
	_ = prometheus.Register(promAMQPPublishCount)
	_ = prometheus.Register(promAMQPPublishDuration)
	_ = prometheus.Register(promAMQPConnectionAttempts)
	_ = prometheus.Register(promAMQPConnectionFailures)
}

// NewAMQPEventBus creates a Kafka-backed EventBus. If a legacy AMQP URL is provided (amqp://...), the bus will be disabled.
func NewAMQPEventBus(amqpURL string, exchange string) (*AMQPEventBus, error) {
	if amqpURL == "" {
		amqpURL = os.Getenv("RABBITMQ_URL")
	}
	if amqpURL == "" {
		amqpURL = os.Getenv("AMQP_URL")
	}
	if amqpURL == "" {
		amqpURL = os.Getenv("KAFKA_BROKERS")
	}

	// If the value looks like an AMQP URL, disable and return a no-op backend (preserves backwards-compatibility)
	if strings.HasPrefix(amqpURL, "amqp://") {
		log.Printf("⚠️  Detected legacy AMQP URL %s - event bus disabled. Set KAFKA_BROKERS instead.", amqpURL)
		return &AMQPEventBus{writer: nil, exchange: exchange}, nil
	}

	brokers := strings.Split(amqpURL, ",")
	w := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}

	return &AMQPEventBus{writer: w, exchange: exchange}, nil
}

// Emit publishes the event payload to the configured topic using the event string as the message key.
func (a *AMQPEventBus) Emit(ctx context.Context, event string, data interface{}) error {
	if a == nil || a.writer == nil {
		return nil
	}
	body, err := json.Marshal(data)
	if err != nil {
		promAMQPPublishCount.WithLabelValues("error").Inc()
		atomic.AddInt64(&amqpPublishFailures, 1)
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	start := time.Now()
	msg := kafka.Message{Topic: a.exchange, Key: []byte(event), Value: body, Time: time.Now()}
	if err := a.writer.WriteMessages(ctx, msg); err != nil {
		durationMs := float64(time.Since(start).Milliseconds())
		promAMQPPublishDuration.Observe(durationMs)
		promAMQPPublishCount.WithLabelValues("error").Inc()
		atomic.AddInt64(&amqpPublishFailures, 1)
		return err
	}

	durationMs := float64(time.Since(start).Milliseconds())
	promAMQPPublishDuration.Observe(durationMs)
	promAMQPPublishCount.WithLabelValues("success").Inc()
	atomic.AddInt64(&amqpPublishedCount, 1)
	return nil
}

// Close cleans up resources.
func (a *AMQPEventBus) Close() error {
	if a == nil {
		return nil
	}
	if a.writer != nil {
		return a.writer.Close()
	}
	return nil
}
