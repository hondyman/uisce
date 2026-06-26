package observability

import "context"

// WithTraceContext returns a context with trace and span IDs set using typed keys.
func WithTraceContext(ctx context.Context, traceID, spanID string) context.Context {
	if traceID != "" {
		ctx = context.WithValue(ctx, ctxKeyTraceID, traceID)
	}
	if spanID != "" {
		ctx = context.WithValue(ctx, ctxKeySpanID, spanID)
	}
	return ctx
}

// AMQP related typed keys
const (
	ctxAmqpRoutingKey    ctxKey = "amqp.routing_key"
	ctxAmqpExchange      ctxKey = "amqp.exchange"
	ctxAmqpCorrelationID ctxKey = "amqp.correlation_id"
)
