package reporting

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type BatchTelemetrySummary struct {
	BatchID          uuid.UUID              `json:"batchId"`
	ScheduleID       uuid.UUID              `json:"scheduleId"`
	Status           string                 `json:"status"`
	TotalClients     int                    `json:"totalClients"`
	SuccessfulCount  int                    `json:"successfulCount"`
	FailedCount      int                    `json:"failedCount"`
	ThroughputPerSec float64                `json:"throughputPerSec"`
	P50LatencyMs     int                    `json:"p50LatencyMs"`
	P95LatencyMs     int                    `json:"p95LatencyMs"`
	P99LatencyMs     int                    `json:"p99LatencyMs"`
	FailedSlices     []FailedArtifactDetail `json:"failedSlices"`
}

type FailedArtifactDetail struct {
	ArtifactID  uuid.UUID `json:"artifactId"`
	ClientID    string    `json:"clientId"`
	ErrorReason string    `json:"errorReason"`
	RetryCount  int       `json:"retryCount"`
}

type TelemetryDLQService struct {
	db *sqlx.DB
}

func NewTelemetryDLQService(db *sqlx.DB) *TelemetryDLQService {
	return &TelemetryDLQService{db: db}
}

// GetBatchTelemetry computes live progress and percentiles across client slices
func (s *TelemetryDLQService) GetBatchTelemetry(ctx context.Context, tenantID, batchID uuid.UUID) (*BatchTelemetrySummary, error) {
	var batch struct {
		ScheduleID uuid.UUID `db:"schedule_id"`
		Status     string    `db:"status"`
		StartedAt  time.Time `db:"started_at"`
	}
	err := s.db.GetContext(ctx, &batch, `
		SELECT schedule_id, status, started_at 
		FROM public.report_burst_batches 
		WHERE id = $1 AND tenant_id = $2
	`, batchID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("batch not found: %w", err)
	}

	var artifacts []struct {
		ID               uuid.UUID `db:"id"`
		ClientID         string    `db:"client_id"`
		Status           string    `db:"status"`
		RenderDurationMs int       `db:"render_duration_ms"`
		ErrorReason      string    `db:"error_reason"`
		RetryCount       int       `db:"retry_count"`
	}
	err = s.db.SelectContext(ctx, &artifacts, `
		SELECT id, client_id, status, COALESCE(render_duration_ms, 0) AS render_duration_ms, 
		       COALESCE(error_reason, '') AS error_reason, COALESCE(retry_count, 0) AS retry_count
		FROM public.report_burst_artifacts
		WHERE batch_id = $1 AND tenant_id = $2
	`, batchID, tenantID)
	if err != nil {
		return nil, err
	}

	total := len(artifacts)
	success := 0
	failed := 0
	latencies := make([]int, 0, total)
	failedSlices := make([]FailedArtifactDetail, 0)

	for _, a := range artifacts {
		if a.Status == "READY" {
			success++
			latencies = append(latencies, a.RenderDurationMs)
		} else if a.Status == "FAILED" {
			failed++
			failedSlices = append(failedSlices, FailedArtifactDetail{
				ArtifactID:  a.ID,
				ClientID:    a.ClientID,
				ErrorReason: a.ErrorReason,
				RetryCount:  a.RetryCount,
			})
		}
	}

	sort.Ints(latencies)
	p50, p95, p99 := 0, 0, 0
	if len(latencies) > 0 {
		p50 = latencies[int(float64(len(latencies))*0.50)]
		p95 = latencies[int(float64(len(latencies))*0.95)]
		p99 = latencies[int(float64(len(latencies))*0.99)]
	}

	elapsedSec := time.Since(batch.StartedAt).Seconds()
	throughput := 0.0
	if elapsedSec > 0 {
		throughput = float64(success) / elapsedSec
	}

	return &BatchTelemetrySummary{
		BatchID:          batchID,
		ScheduleID:       batch.ScheduleID,
		Status:           batch.Status,
		TotalClients:     total,
		SuccessfulCount:  success,
		FailedCount:      failed,
		ThroughputPerSec: throughput,
		P50LatencyMs:     p50,
		P95LatencyMs:     p95,
		P99LatencyMs:     p99,
		FailedSlices:     failedSlices,
	}, nil
}

// RetryFailedSlices creates a targeted child retry batch for failed DLQ artifacts
func (s *TelemetryDLQService) RetryFailedSlices(ctx context.Context, tenantID, batchID uuid.UUID) (*uuid.UUID, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. Fetch Failed Client Artifacts
	var failedClients []string
	err = tx.SelectContext(ctx, &failedClients, `
		SELECT client_id 
		FROM public.report_burst_artifacts 
		WHERE batch_id = $1 AND tenant_id = $2 AND status = 'FAILED'
	`, batchID, tenantID)
	if err != nil || len(failedClients) == 0 {
		return nil, fmt.Errorf("no failed slices found for retry")
	}

	// 2. Spawn Child Retry Batch
	newBatchID := uuid.New()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO public.report_burst_batches (id, tenant_id, schedule_id, effective_date, status, retry_batch_id)
		SELECT $1, tenant_id, schedule_id, effective_date, 'RUNNING', $2
		FROM public.report_burst_batches WHERE id = $2 AND tenant_id = $3
	`, newBatchID, batchID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed creating retry batch: %w", err)
	}

	return &newBatchID, tx.Commit()
}
