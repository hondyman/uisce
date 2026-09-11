package reports

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/db"
)

var ErrInvalidCursor = errors.New("invalid cursor")

const (
	cursorVersion    = 1
	maxEventsLimit  = 1000
	defaultListLimit = 50
	maxListLimit     = 200
)

type Execution struct {
	ID, TenantID, TemplateID uuid.UUID
	ScheduleID               *uuid.UUID
	ReportKey, Status       string
	Parameters               []byte
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

type Event struct {
	ID          uuid.UUID
	ExecutionID uuid.UUID
	Event       string
	FromStatus  sql.NullString
	ToStatus    string
	ActorID     string
	Detail      []byte
	CreatedAt   time.Time
}

type Cursor struct {
	Version   int       `json:"v"`
	CreatedAt time.Time `json:"created_at"`
	ID        uuid.UUID `json:"id"`
}

func EncodeCursor(c Cursor) (string, error) {
	c.Version = cursorVersion
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func DecodeCursor(s string) (Cursor, error) {
	if s == "" {
		return Cursor{}, ErrInvalidCursor
	}
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	var c Cursor
	if err := json.Unmarshal(b, &c); err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	if c.Version != cursorVersion {
		return Cursor{}, ErrInvalidCursor
	}
	if c.CreatedAt.IsZero() || c.ID == uuid.Nil {
		return Cursor{}, ErrInvalidCursor
	}
	return c, nil
}

type ExecutionRepository struct {
	db *sql.DB
}

func NewExecutionRepository(db *sql.DB) *ExecutionRepository {
	return &ExecutionRepository{db: db}
}

func (r *ExecutionRepository) ListExecutions(
	ctx context.Context,
	callerTenantID uuid.UUID,
	userID string,
	isAdmin bool,
	cursor *Cursor,
	limit int,
) ([]Execution, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	var createdAt *time.Time
	var cursorID *uuid.UUID
	if cursor != nil {
		createdAt = &cursor.CreatedAt
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
		  AND (e.tenant_id = $3 OR e.triggered_by = $4)
		  AND (
		      t.is_personal = false
		      OR (t.created_by_id IS NOT NULL AND t.created_by_id = $4)
		      OR $5 = true
		  )
		ORDER BY e.created_at DESC, e.id DESC
		LIMIT $6
	`

	rows, err := r.db.QueryContext(ctx, query, createdAt, cursorID, callerTenantID, userID, isAdmin, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var execs []Execution
	for rows.Next() {
		var e Execution
		err := rows.Scan(
			&e.ID, &e.TenantID, &e.TemplateID, &e.ScheduleID, &e.ReportKey, &e.Status,
			&e.Parameters, &e.OutputURL, &e.OutputSizeBytes, &e.RowsProcessed,
			&e.ExecutionTimeMS, &e.ErrorMessage, &e.WorkflowID, &e.RunID,
			&e.RequestedBy, &e.TriggeredBy, &e.Metadata, &e.CreatedAt, &e.CompletedAt,
			&e.IsPersonal, &e.CreatedByID,
		)
		if err != nil {
			return nil, err
		}
		execs = append(execs, e)
	}
	return execs, rows.Err()
}

func (r *ExecutionRepository) GetExecution(
	ctx context.Context,
	execID uuid.UUID,
	callerTenantID uuid.UUID,
	userID string,
	isAdmin bool,
) (*Execution, error) {
	query := `
		SELECT e.id, e.tenant_id, e.template_id, e.schedule_id, e.report_key, e.status,
		       e.parameters, e.output_url, e.output_size_bytes, e.rows_processed,
		       e.execution_time_ms, e.error_message, e.workflow_id, e.run_id,
		       e.requested_by, e.triggered_by, e.metadata, e.created_at, e.completed_at,
		       t.is_personal, t.created_by_id
		FROM public.report_executions e
		JOIN public.report_templates t ON t.id = e.template_id
		WHERE e.id = $1
		  AND (e.tenant_id = $2 OR e.triggered_by = $3)
		  AND (
		      t.is_personal = false
		      OR (t.created_by_id IS NOT NULL AND t.created_by_id = $3)
		      OR $4 = true
		  )
	`

	var e Execution
	err := r.db.QueryRowContext(ctx, query, execID, callerTenantID, userID, isAdmin).Scan(
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

func (r *ExecutionRepository) ListExecutionEvents(
	ctx context.Context,
	execID uuid.UUID,
	callerTenantID uuid.UUID,
	userID string,
	isAdmin bool,
	cursor *Cursor,
	limit int,
) ([]Event, bool, error) {
	if limit <= 0 {
		limit = maxEventsLimit
	}
	if limit > maxEventsLimit {
		limit = maxEventsLimit
	}

	var executionTenantID uuid.UUID
	resolveQuery := `
		SELECT e.tenant_id
		FROM public.report_executions e
		JOIN public.report_templates t ON t.id = e.template_id
		WHERE e.id = $1
		  AND (e.tenant_id = $2 OR e.triggered_by = $3)
		  AND (
		      t.is_personal = false
		      OR (t.created_by_id IS NOT NULL AND t.created_by_id = $3)
		      OR $4 = true
		  )
	`
	err := r.db.QueryRowContext(ctx, resolveQuery, execID, callerTenantID, userID, isAdmin).Scan(&executionTenantID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, ErrNotFound
	}
	if err != nil {
		return nil, false, err
	}

	var createdAt *time.Time
	var cursorID *uuid.UUID
	if cursor != nil {
		createdAt = &cursor.CreatedAt
		cursorID = &cursor.ID
	}

	var events []Event
	err = db.WithTenantTransaction(ctx, r.db, executionTenantID.String(), func(tx *sql.Tx) error {
		query := `
			SELECT id, execution_id, event, from_status, to_status, actor_id, detail, created_at
			FROM public.report_execution_events
			WHERE execution_id = $1
			  AND ($2::timestamptz IS NULL OR (created_at, id) > ($2, $3))
			ORDER BY created_at ASC, id ASC
			LIMIT $4
		`
		rows, err := tx.QueryContext(ctx, query, execID, createdAt, cursorID, limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var ev Event
			if err := rows.Scan(&ev.ID, &ev.ExecutionID, &ev.Event, &ev.FromStatus, &ev.ToStatus, &ev.ActorID, &ev.Detail, &ev.CreatedAt); err != nil {
				return err
			}
			events = append(events, ev)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, false, err
	}

	truncated := len(events) == limit
	return events, truncated, nil
}

func (r *ExecutionRepository) ListScheduleExecutions(
	ctx context.Context,
	scheduleID uuid.UUID,
	callerTenantID uuid.UUID,
	userID string,
	isAdmin bool,
	cursor *Cursor,
	limit int,
) ([]Execution, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	var createdAt *time.Time
	var cursorID *uuid.UUID
	if cursor != nil {
		createdAt = &cursor.CreatedAt
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
		JOIN public.report_schedules s ON s.id = e.schedule_id
		WHERE s.id = $1
		  AND s.tenant_id = $2
		  AND s.deleted_at IS NULL
		  AND ($3::timestamptz IS NULL OR (e.created_at, e.id) < ($3, $4))
		  AND (e.tenant_id = $2 OR e.triggered_by = $5)
		  AND (
		      t.is_personal = false
		      OR (t.created_by_id IS NOT NULL AND t.created_by_id = $5)
		      OR $6 = true
		  )
		ORDER BY e.created_at DESC, e.id DESC
		LIMIT $7
	`

	rows, err := r.db.QueryContext(ctx, query, scheduleID, callerTenantID, createdAt, cursorID, userID, isAdmin, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var execs []Execution
	for rows.Next() {
		var e Execution
		err := rows.Scan(
			&e.ID, &e.TenantID, &e.TemplateID, &e.ScheduleID, &e.ReportKey, &e.Status,
			&e.Parameters, &e.OutputURL, &e.OutputSizeBytes, &e.RowsProcessed,
			&e.ExecutionTimeMS, &e.ErrorMessage, &e.WorkflowID, &e.RunID,
			&e.RequestedBy, &e.TriggeredBy, &e.Metadata, &e.CreatedAt, &e.CompletedAt,
			&e.IsPersonal, &e.CreatedByID,
		)
		if err != nil {
			return nil, err
		}
		execs = append(execs, e)
	}
	return execs, rows.Err()
}
