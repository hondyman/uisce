package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/hondyman/semlayer/backend/internal/logging"
)

// Webhook handles Hasura event webhooks for the temporal_workflows table.
// It accepts Hasura insert events and forwards them to an EventBus under the
// topic `temporal.workflow`. The handler uses structured logging and
// OpenTelemetry tracing and performs a small retry loop with backoff.
func Webhook(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("backend.httpapi.webhook")
	ctx, span := tracer.Start(r.Context(), "temporal_webhook")
	defer span.End()

	var payload struct {
		Table struct{ Name string }                       `json:"table"`
		Event struct{ Data struct{ New map[string]any } } `json:"event"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		traceID := span.SpanContext().TraceID().String()
		logging.GetLogger().Sugar().Errorw("invalid_payload", "error", err, "trace_id", traceID)
		span.RecordError(err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload"})
		return
	}

	if payload.Table.Name != "temporal_workflows" {
		logging.GetLogger().Sugar().Debugw("ignored_table", "table", payload.Table.Name)
		w.WriteHeader(http.StatusOK)
		return
	}

	em := r.Context().Value("eventBus")
	if em == nil {
		traceID := span.SpanContext().TraceID().String()
		logging.GetLogger().Sugar().Errorw("no_event_bus", "trace_id", traceID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "no event bus configured"})
		return
	}

	type emitter interface {
		Emit(ctx context.Context, topic string, payload any) error
	}

	bus, ok := em.(emitter)
	if !ok {
		traceID := span.SpanContext().TraceID().String()
		logging.GetLogger().Sugar().Errorw("invalid_event_bus", "trace_id", traceID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid event bus"})
		return
	}

	// propagate the request context into the emit, with a reasonable timeout
	emitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var lastErr error
	var spanKind trace.SpanKind = trace.SpanKindServer
	for attempt := 0; attempt < 3; attempt++ {
		_, child := tracer.Start(emitCtx, "emit_to_eventbus", trace.WithSpanKind(spanKind))
		err := bus.Emit(emitCtx, "temporal.workflow", payload.Event.Data.New)
		child.End()
		if err == nil {
			traceID := span.SpanContext().TraceID().String()
			logging.GetLogger().Sugar().Infow("emitted", "workflow_id", payload.Event.Data.New["workflow_id"], "attempt", attempt+1, "trace_id", traceID)
			w.WriteHeader(http.StatusOK)
			return
		}
		lastErr = err
		traceID := span.SpanContext().TraceID().String()
		logging.GetLogger().Sugar().Warnw("emit_retry", "error", err, "attempt", attempt+1, "trace_id", traceID)

		// simple linear backoff
		select {
		case <-time.After(time.Duration(attempt+1) * 300 * time.Millisecond):
		case <-emitCtx.Done():
			logging.GetLogger().Sugar().Errorw("emit_timeout", "error", emitCtx.Err())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusGatewayTimeout)
			json.NewEncoder(w).Encode(map[string]string{"error": "emit timeout", "detail": emitCtx.Err().Error()})
			return
		}
	}

	logging.GetLogger().Sugar().Errorw("emit_failed", "error", lastErr)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{"error": "failed to emit event", "detail": fmt.Sprintf("%v", lastErr)})
}
