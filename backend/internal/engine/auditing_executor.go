package engine

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/hondyman/uisce/backend/internal/domain"
)

// AuditEvent represents the payload published to Redpanda / Kafka topic bo_execution_telemetry
type AuditEvent struct {
	TenantID    string `json:"tenant_id"`
	UserID      string `json:"user_id"`
	BOID        string `json:"bo_id"`
	Engine      string `json:"engine"`
	DurationMs  int64  `json:"duration_ms"`
	ExplainPlan string `json:"explain_plan"`
	ExecutedSQL string `json:"executed_sql"`
}

// AuditingExecutor wraps a QueryExecutor to capture telemetry and emit audit events non-blockingly
type AuditingExecutor struct {
	next        domain.QueryExecutor
	eventStream domain.EventPublisher
}

// NewAuditingExecutor wraps a QueryExecutor with non-blocking telemetry publishing
func NewAuditingExecutor(next domain.QueryExecutor, publisher domain.EventPublisher) *AuditingExecutor {
	return &AuditingExecutor{
		next:        next,
		eventStream: publisher,
	}
}

// Execute triggers pre-flight explain, executes the target query, and emits telemetry asynchronously
func (a *AuditingExecutor) Execute(ctx context.Context, req domain.ExecutionRequest) (*sql.Rows, error) {
	startTime := time.Now()

	// 1. Pre-flight EXPLAIN plan capture
	plan, explainErr := a.next.Explain(ctx, req)
	if explainErr != nil {
		plan = "EXPLAIN_FAILED: " + explainErr.Error()
	}

	// 2. Execute target query
	rows, err := a.next.Execute(ctx, req)
	duration := time.Since(startTime).Milliseconds()

	// 3. Emit Telemetry Asynchronously
	tenantID, _ := ctx.Value("tenant_id").(string)
	userID, _ := ctx.Value("user_id").(string)

	event := AuditEvent{
		TenantID:    tenantID,
		UserID:      userID,
		BOID:        req.BOID,
		Engine:      req.TargetEngine,
		DurationMs:  duration,
		ExplainPlan: plan,
		ExecutedSQL: req.GeneratedSQL,
	}

	if a.eventStream != nil {
		eventBytes, marshalErr := json.Marshal(event)
		if marshalErr == nil {
			go func() {
				_ = a.eventStream.Publish("bo_execution_telemetry", tenantID, eventBytes)
			}()
		}
	}

	return rows, err
}

// Explain delegates directly to the underlying QueryExecutor
func (a *AuditingExecutor) Explain(ctx context.Context, req domain.ExecutionRequest) (string, error) {
	return a.next.Explain(ctx, req)
}
