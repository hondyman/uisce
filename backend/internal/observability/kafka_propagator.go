package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

// KafkaTracePropagator handles trace context for Kafka messages (Redpanda)
// This replaces older AMQP-specific propagators and exposes similar helper methods
// for propagating trace IDs and instrumenting publish/consume operations.
type KafkaTracePropagator struct {
	tp *TracerProvider
}

// NewKafkaTracePropagator creates a new Kafka trace propagator
func NewKafkaTracePropagator(tp *TracerProvider) *KafkaTracePropagator {
	return &KafkaTracePropagator{tp: tp}
}

// InjectHeaders returns Kafka headers containing trace context
func (ktp *KafkaTracePropagator) InjectHeaders(ctx context.Context) []kafka.Header {
	traceID := ""
	if tid, ok := ctx.Value(ctxKeyTraceID).(string); ok {
		traceID = tid
	}

	spanID := ""
	if sid, ok := ctx.Value(ctxKeySpanID).(string); ok {
		spanID = sid
	}

	headers := []kafka.Header{}
	if traceID != "" {
		headers = append(headers, kafka.Header{Key: "X-Trace-ID", Value: []byte(traceID)})
		headers = append(headers, kafka.Header{Key: "X-B3-TraceId", Value: []byte(traceID)})
	}
	if spanID != "" {
		headers = append(headers, kafka.Header{Key: "X-Span-ID", Value: []byte(spanID)})
		headers = append(headers, kafka.Header{Key: "X-B3-SpanId", Value: []byte(spanID)})
	}

	return headers
}

// ExtractContext extracts trace context from a Kafka message
func (ktp *KafkaTracePropagator) ExtractContext(msg kafka.Message) context.Context {
	ctx := context.Background()

	for _, h := range msg.Headers {
		switch h.Key {
		case "X-Trace-ID", "X-B3-TraceId":
			if len(h.Value) > 0 {
				ctx = context.WithValue(ctx, ctxKeyTraceID, string(h.Value))
			}
		case "X-Span-ID", "X-B3-SpanId":
			if len(h.Value) > 0 {
				ctx = context.WithValue(ctx, ctxKeySpanID, string(h.Value))
			}
		}
	}

	// Add Kafka metadata
	ctx = context.WithValue(ctx, "kafka.topic", msg.Topic)
	ctx = context.WithValue(ctx, "kafka.partition", msg.Partition)
	ctx = context.WithValue(ctx, "kafka.offset", msg.Offset)

	return ctx
}

// StartMessageSpan starts a span for a Kafka message
func (ktp *KafkaTracePropagator) StartMessageSpan(ctx context.Context, topic, messageID string) (*Span, context.Context) {
	attributes := map[string]interface{}{
		"messaging.system":      "kafka",
		"messaging.destination": topic,
		"messaging.message_id":  messageID,
		"messaging.operation":   "consume",
	}

	return ktp.tp.StartSpan(ctx, fmt.Sprintf("kafka.consume.%s", messageID), attributes)
}

// TraceMessageHandler wraps a Kafka message handler with tracing
func (ktp *KafkaTracePropagator) TraceMessageHandler(topic, messageID string, handler func(context.Context, []byte) error) func(context.Context, kafka.Message) error {
	return func(ctx context.Context, msg kafka.Message) error {
		// Extract trace context from message
		msgCtx := ktp.ExtractContext(msg)

		// Start span using extracted context
		span, spanCtx := ktp.StartMessageSpan(msgCtx, topic, messageID)

		// Add message details
		ktp.tp.SetAttribute(span, "messaging.body_size", len(msg.Value))
		ktp.tp.SetAttribute(span, "messaging.partition", msg.Partition)
		ktp.tp.SetAttribute(span, "messaging.offset", msg.Offset)

		// Call handler with span context
		err := handler(spanCtx, msg.Value)

		// End span with error status if needed
		status := "ok"
		message := "Message processed successfully"
		if err != nil {
			status = "error"
			message = err.Error()
		}

		ktp.tp.EndSpan(span, status, message)

		if err != nil {
			ktp.tp.SetAttribute(span, "error", true)
			ktp.tp.SetAttribute(span, "error.kind", "exception")
			ktp.tp.AddEvent(span, "error", map[string]interface{}{
				"message": message,
			})
		}

		return err
	}
}

// TracePublishing wraps a publishing operation with tracing
func (ktp *KafkaTracePropagator) TracePublishing(ctx context.Context, topic, messageID string, publisher func(context.Context, kafka.Message) error, body []byte) error {
	// Start span for publishing
	attributes := map[string]interface{}{
		"messaging.system":      "kafka",
		"messaging.destination": topic,
		"messaging.message_id":  messageID,
		"messaging.operation":   "publish",
		"messaging.body_size":   len(body),
	}

	span, spanCtx := ktp.tp.StartSpan(ctx, fmt.Sprintf("kafka.publish.%s", messageID), attributes)

	// Prepare message with headers
	headers := ktp.InjectHeaders(spanCtx)
	msg := kafka.Message{
		Topic:   topic,
		Headers: headers,
		Value:   body,
		Time:    time.Now(),
	}

	// Publish message
	err := publisher(spanCtx, msg)

	// End span
	status := "ok"
	message := "Message published successfully"
	if err != nil {
		status = "error"
		message = err.Error()
	}

	ktp.tp.EndSpan(span, status, message)

	if err != nil {
		ktp.tp.SetAttribute(span, "error", true)
		ktp.tp.SetAttribute(span, "error.kind", "exception")
		ktp.tp.AddEvent(span, "error", map[string]interface{}{
			"message": message,
		})
	}

	return err
}

// UnmarshalMessageWithContext unmarshals JSON from message and adds trace context
func (ktp *KafkaTracePropagator) UnmarshalMessageWithContext(ctx context.Context, body []byte, v interface{}) error {
	err := json.Unmarshal(body, v)

	// Add parsing event to current span
	if span, ok := ctx.Value(ctxKeyCurrentSpan).(*Span); ok {
		if err != nil {
			ktp.tp.AddEvent(span, "json_parse_error", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			ktp.tp.AddEvent(span, "json_parsed", map[string]interface{}{
				"type": fmt.Sprintf("%T", v),
			})
		}
	}

	return err
}

// Deprecated: RabbitMQ-specific propagator removed. Use KafkaTracePropagator for Kafka/Redpanda message tracing and propagation.
