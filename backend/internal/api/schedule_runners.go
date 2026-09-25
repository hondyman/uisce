package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/querybuilder"
	"github.com/hondyman/uisce/backend/internal/reports"
	"github.com/hondyman/uisce/backend/internal/schedule"
	"github.com/hondyman/uisce/backend/internal/tenant"
)

// --- reports ----------------------------------------------------------------

// reportRunner runs a report template through the report executor and
// waits for the report to finish, so the run history says whether the
// report actually succeeded - not merely that it was started.
type reportRunner struct {
	service  *reports.ReportService
	executor reports.ReportExecutor
	db       *sqlx.DB
	poll     time.Duration
	timeout  time.Duration
}

func newReportRunner(svc *reports.ReportService, exec reports.ReportExecutor, db *sqlx.DB) *reportRunner {
	return &reportRunner{service: svc, executor: exec, db: db, poll: 5 * time.Second, timeout: 15 * time.Minute}
}

func (r *reportRunner) Kind() string  { return "report" }
func (r *reportRunner) Label() string { return "Report" }

// template returns the report if userID may run it: a personal report only
// for its owner.
func (r *reportRunner) template(ctx context.Context, tenantID, userID, ref string) (*reports.ReportTemplate, error) {
	id, err := uuid.Parse(ref)
	tid, terr := uuid.Parse(tenantID)
	if err != nil || terr != nil {
		return nil, schedule.MsgTargetNotFound(r.Label(), ref)
	}
	t, err := r.service.GetTemplate(ctx, id, tid)
	if err != nil || t == nil || !t.IsActive {
		return nil, schedule.MsgTargetNotFound(r.Label(), ref)
	}
	if t.IsPersonal && (t.CreatedByID == nil || *t.CreatedByID != userID) {
		return nil, schedule.MsgTargetNotFound(r.Label(), ref)
	}
	return t, nil
}

func (r *reportRunner) Check(ctx context.Context, tenantID, userID, ref string, _ map[string]any) error {
	if r.executor == nil {
		return schedule.MsgTargetNotFound(r.Label(), ref)
	}
	_, err := r.template(ctx, tenantID, userID, ref)
	return err
}

func (r *reportRunner) Targets(ctx context.Context, tenantID, userID string) ([]schedule.TargetInfo, error) {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, err
	}
	list, err := r.service.ListTemplatesScoped(ctx, tid, userID)
	if err != nil {
		return nil, err
	}
	out := make([]schedule.TargetInfo, 0, len(list))
	for _, t := range list {
		if t.IsActive {
			out = append(out, schedule.TargetInfo{Ref: t.ID.String(), Name: t.TemplateName, Description: t.Description})
		}
	}
	return out, nil
}

func (r *reportRunner) Run(ctx context.Context, rc schedule.RunContext) (*schedule.Outcome, error) {
	t, err := r.template(ctx, rc.TenantID, rc.OwnerID, rc.Ref)
	if err != nil {
		return nil, err
	}
	params := map[string]interface{}{}
	for k, v := range rc.Params {
		params[k] = v
	}
	// Not "schedule_id": report_executions.schedule_id is a foreign key to
	// the old report schedules.
	params["core_schedule_id"] = rc.ScheduleID
	params["core_schedule_run_id"] = rc.RunID
	params["triggered_by"] = "schedule:" + rc.ScheduleID
	res, err := r.executor.ExecuteReport(ctx, t, params)
	if err != nil {
		return nil, fmt.Errorf("starting report %s: %w", rc.Ref, err)
	}
	out := &schedule.Outcome{Refs: map[string]any{"report_execution_id": res.ExecutionID.String()}}
	deadline := time.Now().Add(r.timeout)
	for {
		var st struct {
			Status string         `db:"status"`
			Rows   sql.NullInt64  `db:"rows_processed"`
			URL    sql.NullString `db:"output_url"`
		}
		err := r.inTenant(ctx, rc.TenantID, func(tx *sqlx.Tx) error {
			return tx.GetContext(ctx, &st, `SELECT status, rows_processed, output_url FROM public.report_executions
				WHERE id = $1 AND tenant_id = $2::uuid`, res.ExecutionID, rc.TenantID)
		})
		if err != nil {
			return out, fmt.Errorf("reading report execution %s: %w", res.ExecutionID, err)
		}
		switch strings.ToLower(st.Status) {
		case "completed", "succeeded", "success":
			out.Rows = int(st.Rows.Int64)
			out.Summary = "Report " + t.TemplateName + " completed"
			if st.URL.Valid {
				out.Refs["output_url"] = st.URL.String
			}
			return out, nil
		case "failed", "error", "cancelled", "canceled":
			return out, fmt.Errorf("report execution %s ended %s", res.ExecutionID, st.Status)
		}
		if time.Now().After(deadline) {
			return out, fmt.Errorf("report execution %s still %s after %s", res.ExecutionID, st.Status, r.timeout)
		}
		if rc.Heartbeat != nil {
			rc.Heartbeat(st.Status)
		}
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		case <-time.After(r.poll):
		}
	}
}

func (r *reportRunner) inTenant(ctx context.Context, tenantID string, fn func(*sqlx.Tx) error) error {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if err := tenant.SetRLSContext(ctx, tx, tenantID); err != nil {
		return err
	}
	return fn(tx)
}

// --- saved queries ----------------------------------------------------------

// maxScheduledRows bounds a scheduled saved query's stored result.
const maxScheduledRows = 100000

// savedQueryRunner runs a saved query as the schedule's owner and keeps the
// result as a CSV with the run.
type savedQueryRunner struct {
	h *querybuilder.SavedQueryHandler
}

func (r *savedQueryRunner) Kind() string  { return "saved_query" }
func (r *savedQueryRunner) Label() string { return "Saved query" }

func (r *savedQueryRunner) Check(ctx context.Context, tenantID, userID, ref string, _ map[string]any) error {
	if _, err := r.h.FindSavedQuery(ctx, tenantID, userID, ref); err != nil {
		if errors.Is(err, querybuilder.ErrSavedQueryNotFound) {
			return schedule.MsgTargetNotFound(r.Label(), ref)
		}
		return err
	}
	return nil
}

func (r *savedQueryRunner) Targets(ctx context.Context, tenantID, userID string) ([]schedule.TargetInfo, error) {
	list, err := r.h.ListRunnableSavedQueries(ctx, tenantID, userID)
	out := make([]schedule.TargetInfo, 0, len(list))
	for _, q := range list {
		out = append(out, schedule.TargetInfo{Ref: q.ID, Name: q.Name, Description: q.Description})
	}
	return out, err
}

func (r *savedQueryRunner) Run(ctx context.Context, rc schedule.RunContext) (*schedule.Outcome, error) {
	params := map[string][]string{}
	for k, v := range rc.Params {
		params[k] = []string{fmt.Sprint(v)}
	}
	resp, sq, err := r.h.RunSaved(ctx, rc.TenantID, rc.OwnerID, rc.DatasourceID, rc.Region, rc.Ref, params, maxScheduledRows)
	if errors.Is(err, querybuilder.ErrSavedQueryNotFound) {
		return nil, schedule.MsgTargetNotFound(r.Label(), rc.Ref)
	}
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	header := make([]string, len(resp.Columns))
	for i, c := range resp.Columns {
		header[i] = c.Name
	}
	_ = w.Write(header)
	for _, row := range resp.Rows {
		rec := make([]string, len(header))
		for i, name := range header {
			if v := row[name]; v != nil {
				rec[i] = fmt.Sprint(v)
			}
		}
		_ = w.Write(rec)
	}
	w.Flush()
	name := safeFileName(sq.Name) + "-" + rc.ScheduledFor.UTC().Format("20060102-1504") + ".csv"
	return &schedule.Outcome{
		Rows:    resp.RowCount,
		Summary: fmt.Sprintf("%s: %d rows", sq.Name, resp.RowCount),
		Output:  &schedule.Output{FileName: name, ContentType: "text/csv", Content: buf.Bytes(), Rows: resp.RowCount},
	}, nil
}

func safeFileName(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	if b.Len() == 0 {
		return "saved-query"
	}
	return b.String()
}
