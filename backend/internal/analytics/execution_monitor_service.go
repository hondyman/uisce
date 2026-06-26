package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type MonitorExecutionLog struct {
	ID            uuid.UUID       `db:"id" json:"id"`
	EventType     string          `db:"event_type" json:"event_type"`
	Status        string          `db:"status" json:"status"`
	Engine        string          `db:"engine" json:"engine"`
	Payload       json.RawMessage `db:"payload" json:"payload"`
	Result        json.RawMessage `db:"result" json:"result"`
	StartedAt     time.Time       `db:"started_at" json:"started_at"`
	CompletedAt   *time.Time      `db:"completed_at" json:"completed_at"`
	DurationMs    *int            `db:"duration_ms" json:"duration_ms"`
	UserID        *uuid.UUID      `db:"user_id" json:"user_id"`
	TenantID      *uuid.UUID      `db:"tenant_id" json:"tenant_id"`
	ErrorMessage  *string         `db:"error_message" json:"error_message"`
	CalculationID *uuid.UUID      `db:"calculation_id" json:"calculation_id"`
	WorkflowID    *string         `db:"workflow_id" json:"workflow_id"`
	RunID         *string         `db:"run_id" json:"run_id"`
	CreatedAt     time.Time       `db:"created_at" json:"created_at"`
}

type ExecutionMonitorService struct {
	db *sqlx.DB
}

func NewExecutionMonitorService(db *sqlx.DB) *ExecutionMonitorService {
	return &ExecutionMonitorService{db: db}
}

func (s *ExecutionMonitorService) LogStart(ctx context.Context, log MonitorExecutionLog) (uuid.UUID, error) {
	query := `
		INSERT INTO execution_logs (
			event_type, status, engine, payload, started_at, user_id, tenant_id, calculation_id, workflow_id, run_id
		) VALUES (
			:event_type, :status, :engine, :payload, :started_at, :user_id, :tenant_id, :calculation_id, :workflow_id, :run_id
		) RETURNING id
	`

	// Ensure defaults
	if log.StartedAt.IsZero() {
		log.StartedAt = time.Now()
	}
	if log.Status == "" {
		log.Status = "started"
	}

	rows, err := s.db.NamedQueryContext(ctx, query, log)
	if err != nil {
		return uuid.Nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return uuid.Nil, err
		}
		return id, nil
	}

	return uuid.Nil, nil
}

func (s *ExecutionMonitorService) LogCompletion(ctx context.Context, id uuid.UUID, result json.RawMessage) error {
	now := time.Now()
	query := `
		UPDATE execution_logs 
		SET status = 'completed', 
			result = $1, 
			completed_at = $2,
			duration_ms = EXTRACT(EPOCH FROM ($2 - started_at)) * 1000
		WHERE id = $3
	`
	_, err := s.db.ExecContext(ctx, query, result, now, id)
	return err
}

func (s *ExecutionMonitorService) LogFailure(ctx context.Context, id uuid.UUID, errorMessage string) error {
	now := time.Now()
	query := `
		UPDATE execution_logs 
		SET status = 'failed', 
			error_message = $1, 
			completed_at = $2,
			duration_ms = EXTRACT(EPOCH FROM ($2 - started_at)) * 1000
		WHERE id = $3
	`
	_, err := s.db.ExecContext(ctx, query, errorMessage, now, id)
	return err
}

func (s *ExecutionMonitorService) QueryLogs(ctx context.Context, limit int, offset int) ([]MonitorExecutionLog, error) {
	logs := []MonitorExecutionLog{}
	query := `
		SELECT * FROM execution_logs 
		ORDER BY started_at DESC 
		LIMIT $1 OFFSET $2
	`
	err := s.db.SelectContext(ctx, &logs, query, limit, offset)
	return logs, err
}

type ExecutionMetric struct {
	TenantID uuid.UUID
	TermID   uuid.UUID
	TermName string
	Duration time.Duration
	Status   string
	Engine   string
}

func (s *ExecutionMonitorService) RecordMetric(ctx context.Context, m ExecutionMetric) {
	// 1. Log to execution_logs for granular history
	log := MonitorExecutionLog{
		EventType:     "calculation",
		Status:        m.Status,
		Engine:        m.Engine,
		StartedAt:     time.Now().Add(-m.Duration),
		DurationMs:    ptrInt(int(m.Duration.Milliseconds())),
		TenantID:      &m.TenantID,
		CalculationID: &m.TermID,
	}
	s.LogStart(ctx, log)

	// 2. Update heatmap for Ops Cockpit
	s.updateHeatmap(ctx, m)
}

func (s *ExecutionMonitorService) updateHeatmap(ctx context.Context, m ExecutionMetric) {
	bucket := time.Now().Truncate(time.Minute)
	query := `
		INSERT INTO ops_latency_heatmap (bucket_time, dimension_type, dimension_value, p95_ms, request_count)
		VALUES ($1, 'tenant', $2, $3, 1)
		ON CONFLICT (bucket_time, dimension_type, dimension_value) 
		DO UPDATE SET 
			p95_ms = (ops_latency_heatmap.p95_ms * ops_latency_heatmap.request_count + EXCLUDED.p95_ms) / (ops_latency_heatmap.request_count + 1),
			request_count = ops_latency_heatmap.request_count + 1
	`
	s.db.ExecContext(ctx, query, bucket, m.TenantID.String(), m.Duration.Milliseconds())
}

func ptrInt(i int) *int { return &i }
func (s *ExecutionMonitorService) DetectAnomalies(ctx context.Context, tenantID uuid.UUID) error {
	// Simple logic: if p95 > 500ms in last 5 minutes, create an event
	var p95 float64
	query := `SELECT COALESCE(AVG(p95_ms), 0) FROM ops_latency_heatmap WHERE dimension_value = $1 AND bucket_time > now() - interval '5 minutes'`
	err := s.db.GetContext(ctx, &p95, query, tenantID.String())
	if err == nil && p95 > 500 {
		return s.createEvent(ctx, "latency_anomaly", tenantID, "High Latency Detected", fmt.Sprintf("Latency p95 is %.2fms", p95))
	}
	return err
}

func (s *ExecutionMonitorService) createEvent(ctx context.Context, eventType string, tenantID uuid.UUID, title, details string) error {
	query := `
		INSERT INTO ops_events (event_type, scope, tenant_id, severity, title, details, occurred_at)
		VALUES ($1, 'tenant', $2, 'warning', $3, jsonb_build_object('message', $4), now())
	`
	_, err := s.db.ExecContext(ctx, query, eventType, tenantID, title, details)
	return err
}

func (s *ExecutionMonitorService) GetIncidentTimeline(ctx context.Context, tenantID uuid.UUID) ([]map[string]interface{}, error) {
	var events []map[string]interface{}
	query := `
		SELECT event_type, severity, title, details, occurred_at 
		FROM ops_events 
		WHERE tenant_id = $1 
		ORDER BY occurred_at DESC 
		LIMIT 50
	`
	err := s.db.SelectContext(ctx, &events, query, tenantID)
	return events, err
}
