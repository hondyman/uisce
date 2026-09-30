package temporal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	enums "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

// WorkflowRunRecord represents a row in public.bp_workflow_run
type WorkflowRunRecord struct {
	ID            uuid.UUID       `db:"id" json:"id"`
	WorkflowID    string          `db:"workflow_id" json:"workflow_id"`
	RunID         string          `db:"run_id" json:"run_id"`
	TenantID      string          `db:"tenant_id" json:"tenant_id"`
	ProcessID     string          `db:"process_id" json:"process_id"`
	ProcessName   string          `db:"process_name" json:"process_name"`
	TriggerType   string          `db:"trigger_type" json:"trigger_type"` // 'manual', 'schedule', 'event'
	TriggerName   string          `db:"trigger_name" json:"trigger_name"`
	Entity        string          `db:"entity" json:"entity,omitempty"`
	EntityID      string          `db:"entity_id" json:"entity_id,omitempty"`
	Status        string          `db:"status" json:"status"` // 'RUNNING', 'COMPLETED', 'FAILED', 'TIMED_OUT', 'CANCELLED'
	InputPayload  json.RawMessage `db:"input_payload" json:"input_payload,omitempty"`
	OutputPayload json.RawMessage `db:"output_payload" json:"output_payload,omitempty"`
	ErrorMessage  sql.NullString  `db:"error_message" json:"-"`
	ErrorMsgStr   string          `db:"-" json:"error_message,omitempty"`
	StartedAt     time.Time       `db:"started_at" json:"started_at"`
	CompletedAt   *time.Time      `db:"completed_at" json:"completed_at,omitempty"`
	DurationMs    *int64          `db:"duration_ms" json:"duration_ms,omitempty"`
	CreatedAt     time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at" json:"updated_at"`
}

// WorkflowRunFilter provides query filters for searching workflow runs
type WorkflowRunFilter struct {
	TenantID  string
	Entity    string
	EntityID  string
	Status    string
	ProcessID string
	Limit     int
	Offset    int
}

// BPRunTracker manages persistence and reconciliation for bp_workflow_run
type BPRunTracker struct {
	db *sql.DB
	tc client.Client
}

// NewBPRunTracker creates a new tracker instance
func NewBPRunTracker(db *sql.DB, tc client.Client) *BPRunTracker {
	return &BPRunTracker{
		db: db,
		tc: tc,
	}
}

// RecordStart inserts a new RUNNING row into bp_workflow_run at the shared dispatch point
func (t *BPRunTracker) RecordStart(ctx context.Context, rec WorkflowRunRecord) error {
	if rec.TenantID == "" || rec.WorkflowID == "" || rec.RunID == "" {
		return errors.New("tenant_id, workflow_id, and run_id are required")
	}

	query := `
		INSERT INTO public.bp_workflow_run (
			workflow_id, run_id, tenant_id, process_id, process_name,
			trigger_type, trigger_name, entity, entity_id, status,
			input_payload, started_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, 'RUNNING',
			$10, NOW(), NOW(), NOW()
		)
		ON CONFLICT (run_id) DO UPDATE SET
			status = 'RUNNING',
			updated_at = NOW()
	`

	var inputBytes []byte
	if len(rec.InputPayload) > 0 {
		inputBytes = rec.InputPayload
	}

	_, err := t.db.ExecContext(ctx, query,
		rec.WorkflowID,
		rec.RunID,
		rec.TenantID,
		rec.ProcessID,
		rec.ProcessName,
		rec.TriggerType,
		rec.TriggerName,
		rec.Entity,
		rec.EntityID,
		inputBytes,
	)
	return err
}

// RecordTerminal updates a workflow run to its final terminal status
func (t *BPRunTracker) RecordTerminal(ctx context.Context, runID string, status string, outputPayload []byte, errMsg string) error {
	if runID == "" {
		return errors.New("run_id is required")
	}

	query := `
		UPDATE public.bp_workflow_run
		SET 
			status = $1,
			output_payload = CASE WHEN $2::jsonb IS NOT NULL THEN $2::jsonb ELSE output_payload END,
			error_message = $3,
			completed_at = NOW(),
			duration_ms = GREATEST(0, EXTRACT(EPOCH FROM (NOW() - started_at)) * 1000)::bigint,
			updated_at = NOW()
		WHERE run_id = $4
	`

	var outJSON *string
	if len(outputPayload) > 0 {
		s := string(outputPayload)
		outJSON = &s
	}

	var errVal *string
	if errMsg != "" {
		errVal = &errMsg
	}

	res, err := t.db.ExecContext(ctx, query, status, outJSON, errVal, runID)
	if err != nil {
		return err
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no workflow run found for run_id %s", runID)
	}
	return nil
}

// ListRuns queries runs using composite tenant indexes
func (t *BPRunTracker) ListRuns(ctx context.Context, filter WorkflowRunFilter) ([]WorkflowRunRecord, error) {
	if filter.TenantID == "" {
		return nil, errors.New("tenant_id is required")
	}

	baseQuery := `
		SELECT 
			id, workflow_id, run_id, tenant_id, process_id, process_name,
			trigger_type, trigger_name, entity, entity_id, status,
			input_payload, output_payload, error_message,
			started_at, completed_at, duration_ms, created_at, updated_at
		FROM public.bp_workflow_run
		WHERE tenant_id = $1
	`
	args := []interface{}{filter.TenantID}
	argIdx := 2

	if filter.Entity != "" {
		baseQuery += fmt.Sprintf(" AND entity = $%d", argIdx)
		args = append(args, filter.Entity)
		argIdx++
	}

	if filter.EntityID != "" {
		baseQuery += fmt.Sprintf(" AND entity_id = $%d", argIdx)
		args = append(args, filter.EntityID)
		argIdx++
	}

	if filter.Status != "" {
		baseQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.ProcessID != "" {
		baseQuery += fmt.Sprintf(" AND process_id = $%d", argIdx)
		args = append(args, filter.ProcessID)
		argIdx++
	}

	baseQuery += " ORDER BY started_at DESC"

	limit := 50
	if filter.Limit > 0 && filter.Limit <= 200 {
		limit = filter.Limit
	}
	baseQuery += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, limit)
	argIdx++

	if filter.Offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
	}

	dbx := sqlx.NewDb(t.db, "postgres")
	var records []WorkflowRunRecord
	err := dbx.SelectContext(ctx, &records, baseQuery, args...)
	if err != nil {
		return nil, err
	}

	for i := range records {
		if records[i].ErrorMessage.Valid {
			records[i].ErrorMsgStr = records[i].ErrorMessage.String
		}
	}

	return records, nil
}

// GetRun gets a single run record by ID or run_id
func (t *BPRunTracker) GetRun(ctx context.Context, tenantID, identifier string) (*WorkflowRunRecord, error) {
	if tenantID == "" || identifier == "" {
		return nil, errors.New("tenant_id and identifier are required")
	}

	query := `
		SELECT 
			id, workflow_id, run_id, tenant_id, process_id, process_name,
			trigger_type, trigger_name, entity, entity_id, status,
			input_payload, output_payload, error_message,
			started_at, completed_at, duration_ms, created_at, updated_at
		FROM public.bp_workflow_run
		WHERE tenant_id = $1 AND (run_id = $2 OR workflow_id = $2 OR id::text = $2)
		LIMIT 1
	`

	dbx := sqlx.NewDb(t.db, "postgres")
	var record WorkflowRunRecord
	err := dbx.GetContext(ctx, &record, query, tenantID, identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if record.ErrorMessage.Valid {
		record.ErrorMsgStr = record.ErrorMessage.String
	}

	return &record, nil
}

// ReconcileStaleRuns checks runs in RUNNING state older than minAge against Temporal's Visibility API
// and auto-heals any runs that finished without reporting a terminal status
func (t *BPRunTracker) ReconcileStaleRuns(ctx context.Context, tenantID string, minAge time.Duration) (int, error) {
	if t.tc == nil {
		return 0, errors.New("temporal client not configured for reconciliation")
	}

	query := `
		SELECT workflow_id, run_id
		FROM public.bp_workflow_run
		WHERE status = 'RUNNING' 
		  AND ($1 = '' OR tenant_id = $1)
		  AND started_at < $2
		LIMIT 100
	`

	cutoff := time.Now().Add(-minAge)
	rows, err := t.db.QueryContext(ctx, query, tenantID, cutoff)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type staleRun struct {
		workflowID string
		runID      string
	}
	var stale []staleRun
	for rows.Next() {
		var s staleRun
		if err := rows.Scan(&s.workflowID, &s.runID); err == nil {
			stale = append(stale, s)
		}
	}
	_ = rows.Close()

	reconciled := 0
	for _, sr := range stale {
		desc, err := t.tc.DescribeWorkflowExecution(ctx, sr.workflowID, sr.runID)
		if err != nil {
			continue
		}

		info := desc.GetWorkflowExecutionInfo()
		if info == nil {
			continue
		}

		var terminalStatus string
		switch info.GetStatus() {
		case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
			terminalStatus = "COMPLETED"
		case enums.WORKFLOW_EXECUTION_STATUS_FAILED:
			terminalStatus = "FAILED"
		case enums.WORKFLOW_EXECUTION_STATUS_CANCELED:
			terminalStatus = "CANCELLED"
		case enums.WORKFLOW_EXECUTION_STATUS_TERMINATED:
			terminalStatus = "TERMINATED"
		case enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
			terminalStatus = "TIMED_OUT"
		default:
			continue // Still running
		}

		if terminalStatus != "" {
			_ = t.RecordTerminal(ctx, sr.runID, terminalStatus, nil, "Reconciled from Temporal Visibility API")
			reconciled++
		}
	}

	return reconciled, nil
}
