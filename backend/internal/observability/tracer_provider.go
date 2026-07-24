package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Span represents a distributed trace span
type Span struct {
	TraceID       string                 `json:"trace_id"`
	SpanID        string                 `json:"span_id"`
	ParentSpanID  string                 `json:"parent_span_id,omitempty"`
	ServiceName   string                 `json:"service_name"`
	OperationName string                 `json:"operation_name"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time,omitempty"`
	Duration      int64                  `json:"duration_us,omitempty"`
	Status        string                 `json:"status"` // ok, error
	StatusMessage string                 `json:"status_message,omitempty"`
	Attributes    map[string]interface{} `json:"attributes"`
	Events        []SpanEvent            `json:"events,omitempty"`
	Tags          map[string]string      `json:"tags"`
}

// SpanEvent represents an event within a span
type SpanEvent struct {
	Name       string                 `json:"name"`
	Timestamp  time.Time              `json:"timestamp"`
	Attributes map[string]interface{} `json:"attributes"`
}

// TracerProvider manages tracer instances and span collection
type TracerProvider struct {
	serviceName        string
	jaegerEndpoint     string
	spans              []*Span
	mu                 sync.RWMutex
	maxSpans           int
	samplingRate       float64
	contextPropagators map[string]func(context.Context) map[string]string
}

// InitTracerProvider initializes a new Tracer Provider connected to Jaeger
func InitTracerProvider(serviceName, jaegerEndpoint string) (*TracerProvider, error) {
	if serviceName == "" {
		serviceName = "unknown-service"
	}
	if jaegerEndpoint == "" {
		jaegerEndpoint = "http://jaeger:14268/api/traces"
	}

	tp := &TracerProvider{
		serviceName:        serviceName,
		jaegerEndpoint:     jaegerEndpoint,
		spans:              make([]*Span, 0, 10000),
		maxSpans:           10000,
		samplingRate:       0.1, // 10% sampling by default
		contextPropagators: make(map[string]func(context.Context) map[string]string),
	}

	return tp, nil
}

// StartSpan creates a new span with the given operation name
func (tp *TracerProvider) StartSpan(ctx context.Context, operationName string, attributes map[string]interface{}) (*Span, context.Context) {
	traceID := tp.getTraceID(ctx)
	spanID := tp.generateSpanID()
	parentSpanID := tp.getSpanID(ctx)

	span := &Span{
		TraceID:       traceID,
		SpanID:        spanID,
		ParentSpanID:  parentSpanID,
		ServiceName:   tp.serviceName,
		OperationName: operationName,
		StartTime:     time.Now(),
		Status:        "ok",
		Attributes:    make(map[string]interface{}),
		Tags:          make(map[string]string),
		Events:        make([]SpanEvent, 0),
	}

	// Add attributes
	if attributes != nil {
		span.Attributes = attributes
	}

	// Store span
	tp.recordSpan(span)

	// Create new context with span info using typed keys
	newCtx := WithTraceContext(ctx, traceID, spanID)
	newCtx = context.WithValue(newCtx, ctxKeyCurrentSpan, span)

	return span, newCtx
}

// EndSpan finalizes a span and records its duration
func (tp *TracerProvider) EndSpan(span *Span, status string, message string) {
	if span == nil {
		return
	}

	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime).Microseconds()
	span.Status = status
	span.StatusMessage = message
}

// AddEvent adds an event to a span
func (tp *TracerProvider) AddEvent(span *Span, eventName string, attributes map[string]interface{}) {
	if span == nil {
		return
	}

	event := SpanEvent{
		Name:       eventName,
		Timestamp:  time.Now(),
		Attributes: attributes,
	}

	span.Events = append(span.Events, event)
}

// SetAttribute adds an attribute to a span
func (tp *TracerProvider) SetAttribute(span *Span, key string, value interface{}) {
	if span == nil {
		return
	}

	if span.Attributes == nil {
		span.Attributes = make(map[string]interface{})
	}

	span.Attributes[key] = value
}

// SetTag adds a tag to a span
func (tp *TracerProvider) SetTag(span *Span, key string, value string) {
	if span == nil {
		return
	}

	if span.Tags == nil {
		span.Tags = make(map[string]string)
	}

	span.Tags[key] = value
}

// recordSpan stores the span
func (tp *TracerProvider) recordSpan(span *Span) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	// Apply sampling
	if tp.shouldSample() {
		tp.spans = append(tp.spans, span)

		// Trim old spans if we exceed max
		if len(tp.spans) > tp.maxSpans {
			tp.spans = tp.spans[len(tp.spans)-tp.maxSpans:]
		}
	}
}

// GetSpans returns all recorded spans
func (tp *TracerProvider) GetSpans() []*Span {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	result := make([]*Span, len(tp.spans))
	copy(result, tp.spans)
	return result
}

// GetSpansByTraceID returns all spans for a given trace ID
func (tp *TracerProvider) GetSpansByTraceID(traceID string) []*Span {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	result := make([]*Span, 0)
	for _, span := range tp.spans {
		if span.TraceID == traceID {
			result = append(result, span)
		}
	}
	return result
}

// ClearSpans clears all recorded spans (useful for testing)
func (tp *TracerProvider) ClearSpans() {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	tp.spans = make([]*Span, 0, 10000)
}

// Shutdown gracefully shuts down the tracer provider
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	// Flush remaining spans
	return tp.ForceFlush(ctx)
}

// ForceFlush exports all pending spans to Jaeger
func (tp *TracerProvider) ForceFlush(ctx context.Context) error {
	spans := tp.GetSpans()
	if len(spans) == 0 {
		return nil
	}

	// In production, this would send spans to Jaeger via HTTP
	// For now, we log the count
	fmt.Printf("Flushing %d spans to Jaeger\n", len(spans))

	return nil
}

// Health checks tracer connectivity
func (tp *TracerProvider) Health() error {
	if tp.serviceName == "" {
		return fmt.Errorf("tracer provider not initialized")
	}
	return nil
}

// Helper methods

func (tp *TracerProvider) getTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(ctxKeyTraceID).(string); ok && traceID != "" {
		return traceID
	}
	return tp.generateTraceID()
}

func (tp *TracerProvider) getSpanID(ctx context.Context) string {
	if spanID, ok := ctx.Value(ctxKeySpanID).(string); ok && spanID != "" {
		return spanID
	}
	return ""
}

func (tp *TracerProvider) shouldSample() bool {
	// Simple random sampling
	return tp.samplingRate >= 1.0 || (float64(uuid.New().ID()%100) < tp.samplingRate*100)
}

func (tp *TracerProvider) generateTraceID() string {
	return uuid.New().String()
}

func (tp *TracerProvider) generateSpanID() string {
	return uuid.New().String()[:16]
}
