package api

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	temporalclient "go.temporal.io/sdk/client"

	"github.com/hondyman/uisce/backend/internal/schedule"
	"github.com/hondyman/uisce/backend/internal/tenant"
	pkgworkflows "github.com/hondyman/uisce/backend/pkg/workflows"
)

// workflowTaskQueue is where RunStoredWorkflow is registered (cmd/worker).
const workflowTaskQueue = "bp_queue"

// workflowRunner schedules Studio / stored workflow definitions (kind "workflow").
// Ref is workflow_definitions.id. Optional params become InitialData for the run.
type workflowRunner struct {
	db *sqlx.DB
	tc temporalclient.Client
}

func (r *workflowRunner) Kind() string  { return "workflow" }
func (r *workflowRunner) Label() string { return "Workflow" }

func (r *workflowRunner) Check(ctx context.Context, tenantID, userID, ref string, _ map[string]any) error {
	if strings.TrimSpace(ref) == "" {
		return schedule.MsgTargetNotFound(r.Label(), ref)
	}
	var n int
	err := r.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &n, `
			SELECT COUNT(*) FROM public.workflow_definitions
			WHERE id::text = $1 AND tenant_id::text = $2 AND status IN ('active', 'draft')`,
			ref, tenantID)
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return schedule.MsgTargetNotFound(r.Label(), ref)
	}
	return nil
}

func (r *workflowRunner) Targets(ctx context.Context, tenantID, _ string) ([]schedule.TargetInfo, error) {
	var rows []struct {
		ID     string         `db:"id"`
		Name   string         `db:"name"`
		Status string         `db:"status"`
		Desc   sql.NullString `db:"description"`
	}
	err := r.inTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &rows, `
			SELECT id::text, name, status, description
			FROM public.workflow_definitions
			WHERE tenant_id::text = $1 AND status IN ('active', 'draft')
			ORDER BY name`, tenantID)
	})
	if err != nil {
		return nil, err
	}
	out := make([]schedule.TargetInfo, 0, len(rows))
	for _, row := range rows {
		ti := schedule.TargetInfo{Ref: row.ID, Name: row.Name, Description: row.Status}
		if row.Desc.Valid {
			ti.Description = row.Desc.String
		}
		out = append(out, ti)
	}
	return out, nil
}

func (r *workflowRunner) Run(ctx context.Context, rc schedule.RunContext) (*schedule.Outcome, error) {
	if r.tc == nil {
		return nil, fmt.Errorf("temporal client unavailable for workflow schedule")
	}
	if err := r.Check(ctx, rc.TenantID, rc.OwnerID, rc.Ref, rc.Params); err != nil {
		return nil, err
	}
	initial := map[string]any{}
	for k, v := range rc.Params {
		initial[k] = v
	}
	initial["tenant_id"] = rc.TenantID
	initial["schedule_id"] = rc.ScheduleID
	initial["schedule_run_id"] = rc.RunID

	opts := temporalclient.StartWorkflowOptions{
		ID:        fmt.Sprintf("schedule-workflow-%s-%s", rc.Ref, rc.RunID),
		TaskQueue: workflowTaskQueue,
	}
	we, err := r.tc.ExecuteWorkflow(ctx, opts, pkgworkflows.RunStoredWorkflow, pkgworkflows.InterpreterInput{
		WorkflowID:  rc.Ref,
		InitialData: initial,
	})
	if err != nil {
		return nil, fmt.Errorf("starting stored workflow %s: %w", rc.Ref, err)
	}
	out := &schedule.Outcome{
		Summary: "Workflow " + rc.Ref + " started",
		Refs: map[string]any{
			"temporal_workflow_id": we.GetID(),
			"temporal_run_id":      we.GetRunID(),
			"workflow_definition_id": rc.Ref,
		},
	}
	return out, nil
}

func (r *workflowRunner) inTenant(ctx context.Context, tenantID string, fn func(*sqlx.Tx) error) error {
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
