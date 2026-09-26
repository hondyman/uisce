package schedule

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/tenant"
)

// Store persists schedules, runs, outputs and the audit trail (metadata DB).
// Every statement runs in a transaction carrying the tenant's RLS context
// and also filters on tenant_id.
type Store struct{ DB *sqlx.DB }

func (s *Store) tx(ctx context.Context, tenantID string, fn func(*sqlx.Tx) error) error {
	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if err := tenant.SetRLSContext(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("setting tenant context: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

type scheduleRow struct {
	ID           string         `db:"id"`
	TenantID     string         `db:"tenant_id"`
	Name         string         `db:"name"`
	Description  sql.NullString `db:"description"`
	TargetKind   string         `db:"target_kind"`
	TargetRef    string         `db:"target_ref"`
	TargetParams []byte         `db:"target_params"`
	Cron         string         `db:"cron"`
	TimeZone     string         `db:"time_zone"`
	CalendarCD   sql.NullString `db:"calendar_cd"`
	CalendarRule string         `db:"calendar_rule"`
	BusinessDay  sql.NullInt64  `db:"business_day"`
	StartAt      sql.NullTime   `db:"start_at"`
	EndAt        sql.NullTime   `db:"end_at"`
	Enabled      bool           `db:"enabled"`
	OwnerID      string         `db:"owner_id"`
	DatasourceID sql.NullString `db:"datasource_id"`
	Region       sql.NullString `db:"region"`
	Version      int            `db:"version"`
	CreatedBy    string         `db:"created_by"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedBy    sql.NullString `db:"updated_by"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

const scheduleCols = `id::text, tenant_id::text, name, description, target_kind, target_ref, target_params,
	cron, time_zone, calendar_cd, calendar_rule, business_day, start_at, end_at, enabled, owner_id,
	datasource_id, region, version, created_by, created_at, updated_by, updated_at`

func (r scheduleRow) schedule() *Schedule {
	s := &Schedule{
		ID: r.ID, TenantID: r.TenantID, Name: r.Name, Description: r.Description.String,
		Target: Target{Kind: r.TargetKind, Ref: r.TargetRef},
		Timing: Timing{Cron: r.Cron, TimeZone: r.TimeZone, Calendar: r.CalendarCD.String,
			CalendarRule: r.CalendarRule, BusinessDay: int(r.BusinessDay.Int64)},
		Enabled: r.Enabled, OwnerID: r.OwnerID, DatasourceID: r.DatasourceID.String, Region: r.Region.String,
		Version: r.Version, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedBy: r.UpdatedBy.String, UpdatedAt: r.UpdatedAt,
	}
	if r.StartAt.Valid {
		s.Timing.StartAt = &r.StartAt.Time
	}
	if r.EndAt.Valid {
		s.Timing.EndAt = &r.EndAt.Time
	}
	_ = json.Unmarshal(r.TargetParams, &s.Target.Params)
	return s
}

func nullStr(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func businessDayArg(t Timing) sql.NullInt64 {
	if t.rule() != RuleBusinessDayOfMonth {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(t.BusinessDay), Valid: true}
}

func audit(ctx context.Context, tx *sqlx.Tx, tenantID, scheduleID, action, actor string, before, after *Schedule) error {
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	if before == nil {
		b = nil
	}
	if after == nil {
		a = nil
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO public.schedule_audit (tenant_id, schedule_id, action, actor, before, after)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5::jsonb, $6::jsonb)`, tenantID, scheduleID, action, actor, nullJSON(b), nullJSON(a))
	return err
}

func nullJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}

func (s *Store) Get(ctx context.Context, tenantID, id string) (*Schedule, error) {
	var out *Schedule
	err := s.tx(ctx, tenantID, func(tx *sqlx.Tx) error {
		var err error
		out, err = getTx(ctx, tx, tenantID, id, false)
		return err
	})
	return out, err
}

func getTx(ctx context.Context, tx *sqlx.Tx, tenantID, id string, lock bool) (*Schedule, error) {
	q := `SELECT ` + scheduleCols + ` FROM public.schedules WHERE id::text = $1 AND tenant_id::text = $2 AND deleted_at IS NULL`
	if lock {
		q += ` FOR UPDATE`
	}
	var r scheduleRow
	if err := tx.GetContext(ctx, &r, q, id, tenantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, msgNotFound(id)
		}
		return nil, err
	}
	return r.schedule(), nil
}

// ListFilter narrows List.
type ListFilter struct {
	Kind string
	Ref  string
}

func (s *Store) List(ctx context.Context, tenantID string, f ListFilter) ([]*Schedule, error) {
	var rows []scheduleRow
	err := s.tx(ctx, tenantID, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &rows, `SELECT `+scheduleCols+` FROM public.schedules
			WHERE tenant_id::text = $1 AND deleted_at IS NULL AND ($2 = '' OR target_kind = $2) AND ($3 = '' OR target_ref = $3)
			ORDER BY name`, tenantID, f.Kind, f.Ref)
	})
	out := make([]*Schedule, len(rows))
	for i, r := range rows {
		out[i] = r.schedule()
	}
	return out, err
}

func (s *Store) Create(ctx context.Context, sc *Schedule, actor string) (*Schedule, error) {
	params, _ := json.Marshal(sc.Target.Params)
	var out *Schedule
	err := s.tx(ctx, sc.TenantID, func(tx *sqlx.Tx) error {
		var id string
		if err := tx.GetContext(ctx, &id, `INSERT INTO public.schedules
			(tenant_id, name, description, target_kind, target_ref, target_params, cron, time_zone, calendar_cd,
			 calendar_rule, business_day, start_at, end_at, enabled, owner_id, datasource_id, region, created_by)
			VALUES ($1::uuid, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
			RETURNING id::text`,
			sc.TenantID, sc.Name, nullStr(sc.Description), sc.Target.Kind, sc.Target.Ref, string(params),
			sc.Timing.Cron, sc.Timing.TimeZone, nullStr(sc.Timing.Calendar), sc.Timing.rule(), businessDayArg(sc.Timing),
			nullTime(sc.Timing.StartAt), nullTime(sc.Timing.EndAt), sc.Enabled, sc.OwnerID, nullStr(sc.DatasourceID),
			nullStr(sc.Region), actor); err != nil {
			return err
		}
		var err error
		if out, err = getTx(ctx, tx, sc.TenantID, id, false); err != nil {
			return err
		}
		return audit(ctx, tx, sc.TenantID, id, "create", actor, nil, out)
	})
	return out, err
}

// Update applies change to the stored schedule (locked) and records it.
// change returns an error to abort.
func (s *Store) Update(ctx context.Context, tenantID, id, action, actor string, change func(*Schedule) error) (before, after *Schedule, err error) {
	err = s.tx(ctx, tenantID, func(tx *sqlx.Tx) error {
		cur, err := getTx(ctx, tx, tenantID, id, true)
		if err != nil {
			return err
		}
		before = cur
		next := *cur
		next.Target.Params = cur.Target.Params
		if err := change(&next); err != nil {
			return err
		}
		params, _ := json.Marshal(next.Target.Params)
		if _, err := tx.ExecContext(ctx, `UPDATE public.schedules SET name = $3, description = $4, target_kind = $5,
			target_ref = $6, target_params = $7::jsonb, cron = $8, time_zone = $9, calendar_cd = $10, calendar_rule = $11,
			business_day = $12, start_at = $13, end_at = $14, enabled = $15, version = version + 1,
			updated_by = $16, updated_at = now()
			WHERE id::text = $1 AND tenant_id::text = $2`,
			id, tenantID, next.Name, nullStr(next.Description), next.Target.Kind, next.Target.Ref, string(params),
			next.Timing.Cron, next.Timing.TimeZone, nullStr(next.Timing.Calendar), next.Timing.rule(),
			businessDayArg(next.Timing), nullTime(next.Timing.StartAt), nullTime(next.Timing.EndAt), next.Enabled, actor); err != nil {
			return err
		}
		if after, err = getTx(ctx, tx, tenantID, id, false); err != nil {
			return err
		}
		return audit(ctx, tx, tenantID, id, action, actor, before, after)
	})
	return before, after, err
}

// Delete soft-deletes (runs keep their schedule) and records it.
func (s *Store) Delete(ctx context.Context, tenantID, id, actor string) (*Schedule, error) {
	var before *Schedule
	err := s.tx(ctx, tenantID, func(tx *sqlx.Tx) error {
		cur, err := getTx(ctx, tx, tenantID, id, true)
		if err != nil {
			return err
		}
		before = cur
		if _, err := tx.ExecContext(ctx, `UPDATE public.schedules SET deleted_at = now(), enabled = false, updated_by = $3, updated_at = now()
			WHERE id::text = $1 AND tenant_id::text = $2`, id, tenantID, actor); err != nil {
			return err
		}
		return audit(ctx, tx, tenantID, id, "delete", actor, cur, nil)
	})
	return before, err
}

// AuditEntry is one recorded change.
type AuditEntry struct {
	Action string          `db:"action" json:"action"`
	Actor  string          `db:"actor" json:"actor"`
	At     time.Time       `db:"at" json:"at"`
	Before json.RawMessage `db:"before" json:"before,omitempty"`
	After  json.RawMessage `db:"after" json:"after,omitempty"`
}

func (s *Store) Audit(ctx context.Context, tenantID, id string) ([]AuditEntry, error) {
	var out []AuditEntry
	err := s.tx(ctx, tenantID, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &out, `SELECT action, actor, at, COALESCE(before, 'null'::jsonb) AS before,
			COALESCE(after, 'null'::jsonb) AS after FROM public.schedule_audit
			WHERE tenant_id::text = $1 AND schedule_id::text = $2 ORDER BY at DESC LIMIT 200`, tenantID, id)
	})
	return out, err
}

// --- runs -----------------------------------------------------------------

// Run is one firing of a schedule.
type Run struct {
	ID               string          `db:"id" json:"id"`
	ScheduleID       string          `db:"schedule_id" json:"schedule_id"`
	ScheduleName     string          `db:"schedule_name" json:"schedule_name"`
	TargetKind       string          `db:"target_kind" json:"target_kind"`
	TargetRef        string          `db:"target_ref" json:"target_ref"`
	Trigger          string          `db:"trigger" json:"trigger"`
	TriggeredBy      sql.NullString  `db:"triggered_by" json:"-"`
	ScheduledFor     time.Time       `db:"scheduled_for" json:"scheduled_for"`
	StartedAt        sql.NullTime    `db:"started_at" json:"-"`
	FinishedAt       sql.NullTime    `db:"finished_at" json:"-"`
	Status           string          `db:"status" json:"status"`
	SkipReason       sql.NullString  `db:"skip_reason" json:"-"`
	CalendarDecision json.RawMessage `db:"calendar_decision" json:"calendar_decision,omitempty"`
	Outcome          json.RawMessage `db:"outcome" json:"outcome,omitempty"`
	ErrorCode        sql.NullString  `db:"error_code" json:"-"`
	ErrorParams      json.RawMessage `db:"error_params" json:"-"`
	HasOutput        bool            `db:"has_output" json:"has_output"`
	WorkflowID       sql.NullString  `db:"workflow_id" json:"-"`
}

// MarshalJSON flattens nullable columns. error_detail is never included.
func (r Run) MarshalJSON() ([]byte, error) {
	type alias Run
	var started, finished *time.Time
	if r.StartedAt.Valid {
		started = &r.StartedAt.Time
	}
	if r.FinishedAt.Valid {
		finished = &r.FinishedAt.Time
	}
	var params []string
	_ = json.Unmarshal(r.ErrorParams, &params)
	return json.Marshal(struct {
		alias
		TriggeredBy string     `json:"triggered_by,omitempty"`
		StartedAt   *time.Time `json:"started_at,omitempty"`
		FinishedAt  *time.Time `json:"finished_at,omitempty"`
		SkipReason  string     `json:"skip_reason,omitempty"`
		ErrorCode   string     `json:"error_code,omitempty"`
		ErrorParams []string   `json:"error_params,omitempty"`
	}{alias(r), r.TriggeredBy.String, started, finished, r.SkipReason.String, r.ErrorCode.String, params})
}

const runCols = `r.id::text, r.schedule_id::text, s.name AS schedule_name, r.target_kind, r.target_ref, r.trigger,
	r.triggered_by, r.scheduled_for, r.started_at, r.finished_at, r.status, r.skip_reason,
	COALESCE(r.calendar_decision, 'null'::jsonb) AS calendar_decision, COALESCE(r.outcome, 'null'::jsonb) AS outcome,
	r.error_code, COALESCE(r.error_params, 'null'::jsonb) AS error_params,
	EXISTS (SELECT 1 FROM public.schedule_run_outputs o WHERE o.run_id = r.id) AS has_output, r.workflow_id`

// RunFilter narrows Runs.
type RunFilter struct {
	ScheduleID string
	Status     string
	Limit      int
}

func (s *Store) Runs(ctx context.Context, tenantID string, f RunFilter) ([]Run, error) {
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var out []Run
	err := s.tx(ctx, tenantID, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &out, `SELECT `+runCols+` FROM public.schedule_runs r
			JOIN public.schedules s ON s.id = r.schedule_id
			WHERE r.tenant_id::text = $1 AND ($2 = '' OR r.schedule_id::text = $2) AND ($3 = '' OR r.status = $3)
			ORDER BY r.scheduled_for DESC LIMIT $4`, tenantID, f.ScheduleID, f.Status, limit)
	})
	if out == nil {
		out = []Run{}
	}
	return out, err
}

func (s *Store) GetRun(ctx context.Context, tenantID, runID string) (*Run, error) {
	var out Run
	err := s.tx(ctx, tenantID, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &out, `SELECT `+runCols+` FROM public.schedule_runs r
			JOIN public.schedules s ON s.id = r.schedule_id
			WHERE r.tenant_id::text = $1 AND r.id::text = $2`, tenantID, runID)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, msgRunNotFound(runID)
	}
	return &out, err
}

// RunStart records a run about to start (or a skipped firing).
type RunStart struct {
	TenantID, ScheduleID, Kind, Ref, Trigger, TriggeredBy string
	ScheduledFor                                          time.Time
	Status                                                string // running | skipped
	SkipReason                                            string
	Decision                                              *Decision
	WorkflowID, WorkflowRunID                             string
}

func (s *Store) StartRun(ctx context.Context, in RunStart) (string, error) {
	var id string
	dec, _ := json.Marshal(in.Decision)
	if in.Decision == nil {
		dec = nil
	}
	err := s.tx(ctx, in.TenantID, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &id, `INSERT INTO public.schedule_runs
			(tenant_id, schedule_id, target_kind, target_ref, trigger, triggered_by, scheduled_for, started_at, finished_at,
			 status, skip_reason, calendar_decision, workflow_id, workflow_run_id)
			VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, now(), CASE WHEN $8 = 'skipped' THEN now() END,
			        $8, $9, $10::jsonb, $11, $12)
			RETURNING id::text`, in.TenantID, in.ScheduleID, in.Kind, in.Ref, in.Trigger, nullStr(in.TriggeredBy),
			in.ScheduledFor, in.Status, nullStr(in.SkipReason), nullJSON(dec), nullStr(in.WorkflowID), nullStr(in.WorkflowRunID))
	})
	return id, err
}

// FinishRun records a run's result. code/params are the catalog error
// shown to users; detail is internal.
func (s *Store) FinishRun(ctx context.Context, tenantID, runID string, out *Outcome, code string, params []string, detail string) error {
	status := "succeeded"
	if code != "" {
		status = "failed"
	}
	var outcome, errParams any
	if out != nil {
		b, _ := json.Marshal(out)
		outcome = string(b)
	}
	if len(params) > 0 {
		b, _ := json.Marshal(params)
		errParams = string(b)
	}
	return s.tx(ctx, tenantID, func(tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx, `UPDATE public.schedule_runs SET status = $3, finished_at = now(),
			outcome = $4::jsonb, error_code = $5, error_params = $6::jsonb, error_detail = $7
			WHERE id::text = $1 AND tenant_id::text = $2`,
			runID, tenantID, status, outcome, nullStr(code), errParams, nullStr(detail)); err != nil {
			return err
		}
		if out != nil && out.Output != nil {
			_, err := tx.ExecContext(ctx, `INSERT INTO public.schedule_run_outputs (run_id, tenant_id, file_name, content_type, content, row_count)
				VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6) ON CONFLICT (run_id) DO NOTHING`,
				runID, tenantID, out.Output.FileName, out.Output.ContentType, out.Output.Content, out.Output.Rows)
			return err
		}
		return nil
	})
}

// RunOutput is a file a run produced.
type RunOutput struct {
	FileName    string `db:"file_name"`
	ContentType string `db:"content_type"`
	Content     []byte `db:"content"`
}

func (s *Store) Output(ctx context.Context, tenantID, runID string) (*RunOutput, error) {
	var out RunOutput
	err := s.tx(ctx, tenantID, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &out, `SELECT file_name, content_type, content FROM public.schedule_run_outputs
			WHERE tenant_id::text = $1 AND run_id::text = $2`, tenantID, runID)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, msgNoOutput(runID)
	}
	return &out, err
}
