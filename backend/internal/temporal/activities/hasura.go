//go:build ignore
// +build ignore

package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/hondyman/semlayer/internal/logger"
	"github.com/hondyman/semlayer/internal/metrics"
)

// Record posts a temporal_workflow record back to Hasura via GraphQL. This
// implementation includes structured logging, OpenTelemetry spans, and
// Prometheus metrics. The file is build-tagged so teams can adapt imports and
// enable it when ready.
func Record(ctx context.Context, data map[string]any) error {
	tracer := otel.Tracer("activity")
	ctx, span := tracer.Start(ctx, "hasura_record")
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	logger.L.Infow("hasura_insert_start", "data", data, "trace_id", traceID)

	start := time.Now()
	var lastErr error

	hasuraURL := os.Getenv("HASURA_GRAPHQL_URL")
	if hasuraURL == "" {
		hasuraURL = "http://hasura:8080/v1/graphql"
	}

	payload := map[string]any{
		"query":     `mutation($d:temporal_workflows_insert_input!){insert_temporal_workflows_one(object:$d){id}}`,
		"variables": map[string]any{"d": data},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		metrics.ObserveInsert(start, err)
		logger.L.Errorw("marshal_payload_failed", "error", err, "trace_id", traceID)
		span.RecordError(err)
		return err
	}

	client := &http.Client{Timeout: 15 * time.Second}
	attempts := 5
	backoff := 1 * time.Second
	for i := 0; i < attempts; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, hasuraURL, bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		if s := os.Getenv("HASURA_ADMIN_SECRET"); s != "" {
			req.Header.Set("x-hasura-admin-secret", s)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			logger.L.Warnw("hasura_request_failed", "error", err, "attempt", i+1, "trace_id", traceID)
		} else {
			defer resp.Body.Close()
			if resp.StatusCode >= 400 {
				body, _ := io.ReadAll(resp.Body)
				lastErr = fmt.Errorf("hasura error %d: %s", resp.StatusCode, string(body))
				logger.L.Warnw("hasura_response_error", "status", resp.StatusCode, "body", string(body), "attempt", i+1, "trace_id", traceID)
			} else {
				metrics.ObserveInsert(start, nil)
				logger.L.Infow("hasura_insert_success", "trace_id", traceID)
				return nil
			}
		}

		// backoff or cancel
		select {
		case <-time.After(backoff):
			backoff = backoff * 2
		case <-ctx.Done():
			metrics.ObserveInsert(start, ctx.Err())
			logger.L.Errorw("hasura_request_cancelled", "error", ctx.Err(), "trace_id", traceID)
			span.RecordError(ctx.Err())
			return ctx.Err()
		}
	}

	metrics.ObserveInsert(start, lastErr)
	logger.L.Errorw("hasura_insert_failed", "error", lastErr, "trace_id", traceID)
	span.RecordError(lastErr)
	return lastErr
}
