package datapipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type RunRepository struct {
	db *sqlx.DB
}

func NewRunRepository(db *sqlx.DB) *RunRepository {
	return &RunRepository{db: db}
}

func (r *RunRepository) CreateRun(ctx context.Context, run *PipelineExecutionRun, dagJSON json.RawMessage) error {
	if r.db == nil {
		return nil
	}
	stepOrder, _ := json.Marshal(run.StepOrder)
	errorDetails, _ := json.Marshal(run.ErrorDetails)
	query := `
		INSERT INTO public.data_pipeline_runs
		  (id, tenant_id, pipeline_id, trigger_id, status, start_time, step_order, error_details, dag_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.ExecContext(ctx, query,
		run.RunID, run.TenantID, run.PipelineID, run.TriggerID, run.Status, run.StartTime,
		stepOrder, errorDetails, dagJSON,
	)
	if err != nil {
		return fmt.Errorf("CreateRun: %w", err)
	}
	return nil
}

func (r *RunRepository) UpsertStepTelemetry(ctx context.Context, runID uuid.UUID, nodeID string, metrics StepMetrics, orderIndex int) error {
	if r.db == nil {
		return nil
	}
	query := `
		INSERT INTO public.data_pipeline_step_telemetry
		  (run_id, node_id, node_label, node_type, records_in, records_out, records_error,
		   bytes_processed, duration_ms, rows_per_sec, status, error_message, step_order_index)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (run_id, node_id) DO UPDATE SET
		  records_in = EXCLUDED.records_in,
		  records_out = EXCLUDED.records_out,
		  records_error = EXCLUDED.records_error,
		  duration_ms = EXCLUDED.duration_ms,
		  rows_per_sec = EXCLUDED.rows_per_sec,
		  status = EXCLUDED.status,
		  error_message = EXCLUDED.error_message,
		  step_order_index = EXCLUDED.step_order_index`
	_, err := r.db.ExecContext(ctx, query,
		runID, nodeID, metrics.NodeLabel, metrics.NodeType,
		metrics.RecordsIn, metrics.RecordsOut, metrics.RecordsError,
		metrics.BytesProcessed, metrics.Duration.Milliseconds(),
		metrics.RowsPerSec, metrics.Status, metrics.ErrorMessage, orderIndex,
	)
	if err != nil {
		return fmt.Errorf("UpsertStepTelemetry: %w", err)
	}
	return nil
}

func (r *RunRepository) UpdateRunCompletion(ctx context.Context, run *PipelineExecutionRun) error {
	if r.db == nil {
		return nil
	}
	query := `
		UPDATE public.data_pipeline_runs SET
		  status = $2,
		  end_time = $3,
		  total_records_in = $4,
		  total_records_out = $5,
		  total_errors = $6,
		  peak_throughput_rows_sec = $7,
		  step_order = $8,
		  error_details = $9,
		  updated_at = NOW()
		WHERE id = $1`
	stepOrder, _ := json.Marshal(run.StepOrder)
	errorDetails, _ := json.Marshal(run.ErrorDetails)
	_, err := r.db.ExecContext(ctx, query,
		run.RunID, run.Status, run.EndTime,
		run.TotalRecordsIn, run.TotalRecordsOut, run.TotalErrors,
		run.PeakThroughput, stepOrder, errorDetails,
	)
	if err != nil {
		return fmt.Errorf("UpdateRunCompletion: %w", err)
	}
	return nil
}

func (r *RunRepository) GetRun(ctx context.Context, runID uuid.UUID) (*PipelineExecutionRun, error) {
	if r.db == nil {
		return nil, fmt.Errorf("GetRun: db is nil")
	}
	query := `
		SELECT id, tenant_id, pipeline_id, status, start_time, end_time,
		       total_records_in, total_records_out, total_errors,
		       peak_throughput_rows_sec, step_order, error_details
		FROM public.data_pipeline_runs
		WHERE id = $1`
	var run PipelineExecutionRun
	var stepOrder, errorDetails []byte
	err := r.db.QueryRowContext(ctx, query, runID).Scan(
		&run.RunID, &run.TenantID, &run.PipelineID, &run.Status, &run.StartTime, &run.EndTime,
		&run.TotalRecordsIn, &run.TotalRecordsOut, &run.TotalErrors,
		&run.PeakThroughput, &stepOrder, &errorDetails,
	)
	if err != nil {
		return nil, fmt.Errorf("GetRun: %w", err)
	}
	if stepOrder != nil {
		json.Unmarshal(stepOrder, &run.StepOrder)
	}
	if errorDetails != nil {
		json.Unmarshal(errorDetails, &run.ErrorDetails)
	}
	run.StepTelemetry = make(map[string]StepMetrics)

	stepQuery := `
		SELECT node_id, node_label, node_type, records_in, records_out, records_error,
		       bytes_processed, duration_ms, rows_per_sec, status, error_message
		FROM public.data_pipeline_step_telemetry
		WHERE run_id = $1 ORDER BY step_order_index`
	rows, err := r.db.QueryContext(ctx, stepQuery, runID)
	if err != nil {
		return nil, fmt.Errorf("GetRun steps: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var m StepMetrics
		var nodeID string
		var durMs int64
		if err := rows.Scan(&nodeID, &m.NodeLabel, &m.NodeType, &m.RecordsIn, &m.RecordsOut,
			&m.RecordsError, &m.BytesProcessed, &durMs, &m.RowsPerSec, &m.Status, &m.ErrorMessage); err != nil {
			return nil, fmt.Errorf("scan step: %w", err)
		}
		m.NodeID = nodeID
		m.Duration = time.Duration(durMs) * time.Millisecond
		run.StepTelemetry[nodeID] = m
	}
	return &run, nil
}

func (r *RunRepository) ListRuns(ctx context.Context, tenantID, pipelineID uuid.UUID, triggerID *uuid.UUID, limit int) ([]PipelineExecutionRun, error) {
	if r.db == nil {
		return nil, fmt.Errorf("ListRuns: db is nil")
	}
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, tenant_id, pipeline_id, COALESCE(trigger_id, '00000000-0000-0000-0000-000000000000'::uuid), status, start_time, end_time,
		       total_records_in, total_records_out, total_errors,
		       peak_throughput_rows_sec, step_order, error_details
		FROM public.data_pipeline_runs
		WHERE tenant_id = $1
		  AND ($2 = '00000000-0000-0000-0000-000000000000'::uuid OR pipeline_id = $2)
		  AND ($3 = '00000000-0000-0000-0000-000000000000'::uuid OR trigger_id = $3)
		ORDER BY start_time DESC
		LIMIT $4`
	rows, err := r.db.QueryContext(ctx, query, tenantID, pipelineID, triggerID, limit)
	if err != nil {
		return nil, fmt.Errorf("ListRuns: %w", err)
	}
	defer rows.Close()
	var runs []PipelineExecutionRun
	for rows.Next() {
		var run PipelineExecutionRun
		var stepOrder, errorDetails []byte
		var triggerIDVal uuid.UUID
		if err := rows.Scan(&run.RunID, &run.TenantID, &run.PipelineID, &triggerIDVal, &run.Status, &run.StartTime, &run.EndTime,
			&run.TotalRecordsIn, &run.TotalRecordsOut, &run.TotalErrors,
			&run.PeakThroughput, &stepOrder, &errorDetails); err != nil {
			return nil, fmt.Errorf("scan run: %w", err)
		}
		if triggerIDVal != uuid.Nil {
			run.TriggerID = &triggerIDVal
		}
		if stepOrder != nil {
			json.Unmarshal(stepOrder, &run.StepOrder)
		}
		if errorDetails != nil {
			json.Unmarshal(errorDetails, &run.ErrorDetails)
		}
		run.StepTelemetry = make(map[string]StepMetrics)
		runs = append(runs, run)
	}
	return runs, nil
}
