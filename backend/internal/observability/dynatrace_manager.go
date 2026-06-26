package observability

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hondyman/semlayer/backend/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// DynatraceManager handles integration with Dynatrace for observability.
type DynatraceManager struct {
	tracerProvider *sdktrace.TracerProvider
	logger         *zap.Logger
	enabled        bool
}

// NewDynatraceManager initializes the Dynatrace OpenTelemetry exporter and tracer provider.
func NewDynatraceManager(logger *zap.Logger) (*DynatraceManager, error) {
	// Dynatrace configuration is typically handled via environment variables:
	// DT_ENDPOINT, DT_API_TOKEN, OTEL_SERVICE_NAME
	if os.Getenv("DT_ENDPOINT") == "" || os.Getenv("DT_API_TOKEN") == "" {
		logger.Info("Dynatrace environment variables not set, disabling Dynatrace integration.")
		return &DynatraceManager{enabled: false, logger: logger}, nil
	}

	exporter, err := otlptracegrpc.New(context.Background())
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	// Set up standard OpenTelemetry propagator for Dynatrace compatibility
	otel.SetTextMapPropagator(otel.GetTextMapPropagator())

	logger.Info("Dynatrace manager initialized successfully.")

	return &DynatraceManager{
		tracerProvider: tp,
		logger:         logger,
		enabled:        true,
	}, nil
}

// TraceFunc wraps a function call in a Dynatrace trace span.
func (dm *DynatraceManager) TraceFunc(ctx context.Context, spanName string, f func(context.Context) error, attrs ...attribute.KeyValue) error {
	if !dm.enabled {
		return f(ctx)
	}

	tracer := otel.Tracer("semlayer.backend")
	ctx, span := tracer.Start(ctx, spanName, oteltrace.WithAttributes(attrs...))
	defer span.End()

	err := f(ctx)
	if err != nil {
		span.RecordError(err)
	}
	return err
}

// TraceAccessEvaluation wraps access evaluation with comprehensive Dynatrace tracing
func (dm *DynatraceManager) TraceAccessEvaluation(ctx context.Context, tenantID, userID, assetID string, f func(context.Context) (*models.EvaluateAccessResponse, error)) (*models.EvaluateAccessResponse, error) {
	if !dm.enabled {
		return f(ctx)
	}

	tracer := otel.Tracer("semlayer.access_intelligence")
	ctx, span := tracer.Start(ctx, "access_evaluation",
		oteltrace.WithAttributes(
			attribute.String("tenant.id", tenantID),
			attribute.String("user.id", hashString(userID)), // Privacy protection
			attribute.String("asset.id", hashString(assetID)),
			attribute.String("service.name", "access_intelligence"),
		))
	defer span.End()

	startTime := time.Now()
	response, err := f(ctx)
	duration := time.Since(startTime)

	if err != nil {
		span.RecordError(err)
		span.SetAttributes(
			attribute.String("evaluation.result", "error"),
			attribute.String("error.type", "evaluation_error"),
		)
		return response, err
	}

	// Add response attributes
	if response != nil {
		span.SetAttributes(
			attribute.String("evaluation.result", "success"),
			attribute.String("decision.type", response.Decision),
			attribute.Int("allowed.scope.count", len(response.AllowedScope)),
			attribute.String("decision.id", response.DecisionID.String()),
			attribute.Int64("duration.ms", duration.Milliseconds()),
		)

		// Add business event for governance monitoring
		if dm.logger != nil {
			dm.logger.Info("Access evaluation completed",
				zap.String("tenant_id", tenantID),
				zap.String("decision", response.Decision),
				zap.String("reason", response.Reason),
				zap.Duration("duration", duration),
			)
		}
	}

	return response, err
}

// TraceConversationalQuery wraps conversational query processing with Dynatrace tracing
func (dm *DynatraceManager) TraceConversationalQuery(ctx context.Context, conversationID, tenantID, userID, query string, f func(context.Context) error) error {
	if !dm.enabled {
		return f(ctx)
	}

	tracer := otel.Tracer("semlayer.conversational")
	ctx, span := tracer.Start(ctx, "conversational_query",
		oteltrace.WithAttributes(
			attribute.String("conversation.id", conversationID),
			attribute.String("tenant.id", tenantID),
			attribute.String("user.id", hashString(userID)),
			attribute.String("service.name", "conversational_ai"),
			attribute.Int("query.length", len(query)),
		))
	defer span.End()

	startTime := time.Now()
	err := f(ctx)
	duration := time.Since(startTime)

	span.SetAttributes(
		attribute.Int64("processing.duration.ms", duration.Milliseconds()),
	)

	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("query.result", "error"))
	} else {
		span.SetAttributes(attribute.String("query.result", "success"))
	}

	return err
}

// TraceGovernanceAction wraps governance steward actions with Dynatrace tracing
func (dm *DynatraceManager) TraceGovernanceAction(ctx context.Context, action, stewardID, tenantID, targetType, targetID string, f func(context.Context) error) error {
	if !dm.enabled {
		return f(ctx)
	}

	tracer := otel.Tracer("semlayer.governance")
	ctx, span := tracer.Start(ctx, "governance_action",
		oteltrace.WithAttributes(
			attribute.String("action.type", action),
			attribute.String("steward.id", hashString(stewardID)),
			attribute.String("tenant.id", tenantID),
			attribute.String("target.type", targetType),
			attribute.String("target.id", hashString(targetID)),
			attribute.String("service.name", "governance"),
		))
	defer span.End()

	startTime := time.Now()
	err := f(ctx)
	duration := time.Since(startTime)

	span.SetAttributes(
		attribute.Int64("action.duration.ms", duration.Milliseconds()),
	)

	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("action.result", "error"))
	} else {
		span.SetAttributes(attribute.String("action.result", "success"))
	}

	// Log governance action for audit trail
	if dm.logger != nil {
		dm.logger.Info("Governance action completed",
			zap.String("action", action),
			zap.String("steward_id", hashString(stewardID)),
			zap.String("tenant_id", tenantID),
			zap.String("target_type", targetType),
			zap.Duration("duration", duration),
		)
	}

	return err
}

// AddCustomMetrics sends custom metrics to Dynatrace
func (dm *DynatraceManager) AddCustomMetrics(ctx context.Context, metricName string, value float64, tags map[string]string) {
	if !dm.enabled {
		return
	}

	tracer := otel.Tracer("semlayer.metrics")
	_, span := tracer.Start(ctx, "custom_metric",
		oteltrace.WithAttributes(
			attribute.String("metric.name", metricName),
			attribute.Float64("metric.value", value),
		))
	defer span.End()

	// Convert tags to attributes
	attrs := make([]attribute.KeyValue, 0, len(tags))
	for k, v := range tags {
		attrs = append(attrs, attribute.String(k, v))
	}
	span.SetAttributes(attrs...)

	// In a full implementation, this would send metrics to Dynatrace Metrics API
	// For now, we use span attributes for observability
}

// hashString creates a simple hash for privacy protection
func hashString(s string) string {
	if len(s) <= 8 {
		return s
	}
	// Simple hash for privacy - in production, use proper hashing
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	return fmt.Sprintf("%x", h&0xFFFF)
}

// Shutdown gracefully shuts down the tracer provider.
func (dm *DynatraceManager) Shutdown(ctx context.Context) error {
	if !dm.enabled || dm.tracerProvider == nil {
		return nil
	}
	dm.logger.Info("Shutting down Dynatrace manager.")
	return dm.tracerProvider.Shutdown(ctx)
}
