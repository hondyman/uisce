package reports

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	adminMaxListLimit = 500
)

type AdminExecution struct {
	ID, TenantID, TemplateID uuid.UUID
	ScheduleID               *uuid.UUID
	ReportKey, Status       string
	Parameters              []byte
	OutputURL               sql.NullString
	OutputSizeBytes         sql.NullInt64
	RowsProcessed           sql.NullInt64
	ExecutionTimeMS         sql.NullInt64
	ErrorMessage            sql.NullString
	WorkflowID             sql.NullString
	RunID                  sql.NullString
	RequestedBy            sql.NullString
	TriggeredBy             sql.NullString
	Metadata               []byte
	CreatedAt              time.Time
	CompletedAt            *time.Time
	IsPersonal             bool
	CreatedByID            sql.NullString
}

type AdminEvent struct {
	ID          uuid.UUID
	ExecutionID uuid.UUID
	TenantID   uuid.UUID
	Event      string
	FromStatus sql.NullString
	ToStatus   string
	ActorID    string
	Detail     []byte
	CreatedAt  time.Time
}

type AdminExecutionRepository struct {
	db *sql.DB
}

func NewAdminExecutionRepository(db *sql.DB) *AdminExecutionRepository {
	return &AdminExecutionRepository{db: db}
}

func (r *AdminExecutionRepository) ListExecutions(
	ctx context.Context,
	tenantID *uuid.UUID,
	status string,
	from, to *time.Time,
	cursor *Cursor,
	limit int,
) ([]AdminExecution, *Cursor, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > adminMaxListLimit {
		limit = adminMaxListLimit
	}

	var cursorCreatedAt *time.Time
	var cursorID *uuid.UUID
	if cursor != nil {
		cursorCreatedAt = &cursor.CreatedAt
		cursorID = &cursor.ID
	}

	query := `
		SELECT e.id, e.tenant_id, e.template_id, e.schedule_id, e.report_key, e.status,
		       e.parameters, e.output_url, e.output_size_bytes, e.rows_processed,
		       e.execution_time_ms, e.error_message, e.workflow_id, e.run_id,
		       e.requested_by, e.triggered_by, e.metadata, e.created_at, e.completed_at,
		       t.is_personal, t.created_by_id
		FROM public.report_executions e
		JOIN public.report_templates t ON t.id = e.template_id
		WHERE ($1::timestamptz IS NULL OR (e.created_at, e.id) < ($1, $2))
		  AND ($3::uuid IS NULL OR e.tenant_id = $3)
		  AND ($4::text IS NULL OR e.status = $4)
		  AND ($5::timestamptz IS NULL OR e.created_at >= $5)
		  AND ($6::timestamptz IS NULL OR e.created_at <= $6)
		ORDER BY e.created_at DESC, e.id DESC
		LIMIT $7
	`

	rows, err := r.db.QueryContext(ctx, query,
		cursorCreatedAt, cursorID,
		tenantID, nullString(status), from, to,
		limit,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var execs []AdminExecution
	for rows.Next() {
		var e AdminExecution
		err := rows.Scan(
			&e.ID, &e.TenantID, &e.TemplateID, &e.ScheduleID, &e.ReportKey, &e.Status,
			&e.Parameters, &e.OutputURL, &e.OutputSizeBytes, &e.RowsProcessed,
			&e.ExecutionTimeMS, &e.ErrorMessage, &e.WorkflowID, &e.RunID,
			&e.RequestedBy, &e.TriggeredBy, &e.Metadata, &e.CreatedAt, &e.CompletedAt,
			&e.IsPersonal, &e.CreatedByID,
		)
		if err != nil {
			return nil, nil, err
		}
		execs = append(execs, e)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var nextCursor *Cursor
	if len(execs) == limit {
		lastExec := execs[len(execs)-1]
		c := Cursor{CreatedAt: lastExec.CreatedAt, ID: lastExec.ID}
		nextCursor = &c
	}

	return execs, nextCursor, rows.Err()
}

func (r *AdminExecutionRepository) GetExecution(
	ctx context.Context,
	execID uuid.UUID,
) (*AdminExecution, error) {
	query := `
		SELECT e.id, e.tenant_id, e.template_id, e.schedule_id, e.report_key, e.status,
		       e.parameters, e.output_url, e.output_size_bytes, e.rows_processed,
		       e.execution_time_ms, e.error_message, e.workflow_id, e.run_id,
		       e.requested_by, e.triggered_by, e.metadata, e.created_at, e.completed_at,
		       t.is_personal, t.created_by_id
		FROM public.report_executions e
		JOIN public.report_templates t ON t.id = e.template_id
		WHERE e.id = $1
	`

	var e AdminExecution
	err := r.db.QueryRowContext(ctx, query, execID).Scan(
		&e.ID, &e.TenantID, &e.TemplateID, &e.ScheduleID, &e.ReportKey, &e.Status,
		&e.Parameters, &e.OutputURL, &e.OutputSizeBytes, &e.RowsProcessed,
		&e.ExecutionTimeMS, &e.ErrorMessage, &e.WorkflowID, &e.RunID,
		&e.RequestedBy, &e.TriggeredBy, &e.Metadata, &e.CreatedAt, &e.CompletedAt,
		&e.IsPersonal, &e.CreatedByID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *AdminExecutionRepository) ListEvents(
	ctx context.Context,
	tenantID *uuid.UUID,
	from, to *time.Time,
	cursor *Cursor,
	limit int,
) ([]AdminEvent, *Cursor, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > maxEventsLimit {
		limit = maxEventsLimit
	}

	var cursorCreatedAt *time.Time
	var cursorID *uuid.UUID
	if cursor != nil {
		cursorCreatedAt = &cursor.CreatedAt
		cursorID = &cursor.ID
	}

	query := `
		SELECT id, execution_id, tenant_id, event, from_status, to_status,
		       actor_id, detail, created_at
		FROM public.report_execution_events
		WHERE ($1::timestamptz IS NULL OR (created_at, id) < ($1, $2))
		  AND ($3::uuid IS NULL OR tenant_id = $3)
		  AND ($4::timestamptz IS NULL OR created_at >= $4)
		  AND ($5::timestamptz IS NULL OR created_at <= $5)
		ORDER BY created_at DESC, id DESC
		LIMIT $6
	`

	rows, err := r.db.QueryContext(ctx, query,
		cursorCreatedAt, cursorID,
		tenantID, from, to,
		limit,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var events []AdminEvent
	for rows.Next() {
		var ev AdminEvent
		err := rows.Scan(
			&ev.ID, &ev.ExecutionID, &ev.TenantID, &ev.Event,
			&ev.FromStatus, &ev.ToStatus, &ev.ActorID, &ev.Detail, &ev.CreatedAt,
		)
		if err != nil {
			return nil, nil, err
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var nextCursor *Cursor
	if len(events) == limit {
		lastEv := events[len(events)-1]
		c := Cursor{CreatedAt: lastEv.CreatedAt, ID: lastEv.ID}
		nextCursor = &c
	}

	return events, nextCursor, rows.Err()
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
