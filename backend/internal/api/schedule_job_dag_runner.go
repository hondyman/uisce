package api

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	temporalclient "go.temporal.io/sdk/client"

	"github.com/hondyman/uisce/backend/internal/schedule"
	si "github.com/hondyman/uisce/backend/internal/scheduler_intelligence"
)

// jobDAGRunner schedules S2 scheduled_dags on the one scheduler (kind "job_dag").
// Ref is scheduled_dags.id. Creates a DAG run and starts Temporal DAGExecutionWorkflow
// when an execution adapter is wired.
type jobDAGRunner struct {
	si   *si.Service
	exec *si.ExecutionAdapter
}

func newJobDAGRunner(svc *si.Service, tc temporalclient.Client, repo *si.Repository) *jobDAGRunner {
	r := &jobDAGRunner{si: svc}
	if tc != nil && repo != nil {
		r.exec = si.NewExecutionAdapter(tc, repo, slog.Default())
	}
	return r
}

func (r *jobDAGRunner) Kind() string  { return "job_dag" }
func (r *jobDAGRunner) Label() string { return "Job DAG" }

func (r *jobDAGRunner) Check(ctx context.Context, tenantID, _ string, ref string, _ map[string]any) error {
	id, err := uuid.Parse(ref)
	if err != nil {
		return schedule.MsgTargetNotFound(r.Label(), ref)
	}
	dag, err := r.si.GetDAG(ctx, id)
	if err != nil || dag == nil {
		return schedule.MsgTargetNotFound(r.Label(), ref)
	}
	if dag.TenantID != nil && dag.TenantID.String() != tenantID {
		return schedule.MsgTargetNotFound(r.Label(), ref)
	}
	if !dag.IsActive {
		return schedule.MsgTargetNotFound(r.Label(), ref)
	}
	return nil
}

func (r *jobDAGRunner) Targets(ctx context.Context, tenantID, _ string) ([]schedule.TargetInfo, error) {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, err
	}
	dags, err := r.si.ListDAGs(ctx, tid, true)
	if err != nil {
		return nil, err
	}
	out := make([]schedule.TargetInfo, 0, len(dags))
	for _, d := range dags {
		ti := schedule.TargetInfo{Ref: d.ID.String(), Name: d.Name, Description: d.Description}
		out = append(out, ti)
	}
	return out, nil
}

func (r *jobDAGRunner) Run(ctx context.Context, rc schedule.RunContext) (*schedule.Outcome, error) {
	if err := r.Check(ctx, rc.TenantID, rc.OwnerID, rc.Ref, rc.Params); err != nil {
		return nil, err
	}
	dagID, err := uuid.Parse(rc.Ref)
	if err != nil {
		return nil, schedule.MsgTargetNotFound(r.Label(), rc.Ref)
	}
	var triggeredBy *uuid.UUID
	if oid, err := uuid.Parse(rc.OwnerID); err == nil {
		triggeredBy = &oid
	}
	run, err := r.si.TriggerDAG(ctx, dagID, triggeredBy)
	if err != nil {
		return nil, fmt.Errorf("triggering job DAG %s: %w", rc.Ref, err)
	}
	out := &schedule.Outcome{
		Summary: "Job DAG " + rc.Ref + " run " + run.ID.String(),
		Refs: map[string]any{
			"dag_run_id": run.ID.String(),
			"dag_id":     dagID.String(),
		},
	}
	if r.exec != nil {
		dag, err := r.si.GetDAG(ctx, dagID)
		if err != nil {
			return out, fmt.Errorf("loading DAG %s after trigger: %w", rc.Ref, err)
		}
		if err := r.exec.ExecuteDAG(ctx, dag, run); err != nil {
			return out, fmt.Errorf("starting DAG workflow %s: %w", rc.Ref, err)
		}
		if run.TemporalWorkflowID != "" {
			out.Refs["temporal_workflow_id"] = run.TemporalWorkflowID
			out.Refs["temporal_run_id"] = run.TemporalRunID
		}
	}
	return out, nil
}
