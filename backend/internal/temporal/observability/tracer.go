package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// InitTracer sets up OpenTelemetry with Jaeger exporter
// Call this once at application startup, before starting Temporal workers.
//
// Example usage:
//
//	cleanup, err := observability.InitTracer("semlayer-backend", "localhost:14268")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cleanup(context.Background())
func InitTracer(serviceName string, jaegerEndpoint string) (func(context.Context) error, error) {
	// Create Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("environment", "production"), // Could be from env var
		)),
		// Always sample in dev. In production, use probabilistic sampling.
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set as global tracer provider
	otel.SetTracerProvider(tp)

	// Return cleanup function
	return tp.Shutdown, nil
}

// GetTracer returns the configured OTel tracer for the given component.
// Use this in your workflow/activity code to create spans.
//
// Example:
//
//	tracer := observability.GetTracer("nba-workflow")
//	ctx, span := tracer.Start(ctx, "ExtractTrainingData")
//	defer span.End()
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// InjectWorkflowAttributes adds standard workflow metadata to a span.
// This makes traces searchable by workflow ID, run ID, etc.
func InjectWorkflowAttributes(span trace.Span, workflowID, runID, workflowType string) {
	span.SetAttributes(
		attribute.String("temporal.workflow_id", workflowID),
		attribute.String("temporal.run_id", runID),
		attribute.String("temporal.workflow_type", workflowType),
	)
}

// InjectActivityAttributes adds activity-specific metadata.
func InjectActivityAttributes(span trace.Span, activityType string, attemptNumber int) {
	span.SetAttributes(
		attribute.String("temporal.activity_type", activityType),
		attribute.Int("temporal.attempt", attemptNumber),
	)
}

// InjectBusinessAttributes adds domain-specific metadata to make traces semantically meaningful.
// This is the "magic" that makes metadata-driven workflows observable.
//
// Example:
//
//	InjectBusinessAttributes(span, map[string]string{
//	    "asset_class": "private_equity",
//	    "entity_id": "pe-fund-123",
//	    "action": "compliance_check",
//	})
func InjectBusinessAttributes(span trace.Span, attrs map[string]string) {
	for key, value := range attrs {
		span.SetAttributes(attribute.String(fmt.Sprintf("app.%s", key), value))
	}
}
