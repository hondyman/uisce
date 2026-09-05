package datapipeline

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// PipelineWorkflowInput is the payload passed into RunPipelineDAGWorkflow /
// ActivityExecutePipelineDAG for a durable pipeline run.
type PipelineWorkflowInput struct {
	TenantID     uuid.UUID          `json:"tenant_id"`
	Definition   PipelineDefinition `json:"definition"`
	InputRecords []PipelineRecord   `json:"input_records"`
	RunID        uuid.UUID          `json:"run_id,omitempty"`
	TriggerID    *uuid.UUID        `json:"trigger_id,omitempty"`
}

// PipelineActivities holds the dependencies (namely the PipelineEngine,
// which does real DB I/O) needed to run a pipeline DAG from inside a
// Temporal Activity. ExecuteRun performs real, non-deterministic I/O (DB
// reads/writes, in-process concurrency) so it is not workflow-code-safe as
// written — it must run inside an Activity, with RunPipelineDAGWorkflow as
// a thin wrapper workflow that just invokes that Activity. This mirrors the
// existing child-workflow/child-activity convention in
// pkg/workflows/child_pipeline_activity.go and subpipeline_activity.go
// (Activities struct holding dependencies, methods registered on the
// worker) rather than reshaping PipelineDAG into a
// workflows.WorkflowDefinition, since PipelineDAG's node/edge shape and
// its BO/catalog/rule-validator semantics don't map cleanly onto
// InterpreterWorkflow's ACTIVITY/BRANCH DSL.
type PipelineActivities struct {
	Engine *PipelineEngine
}

// NewPipelineActivities creates a new PipelineActivities bound to engine.
func NewPipelineActivities(engine *PipelineEngine) *PipelineActivities {
	return &PipelineActivities{Engine: engine}
}

// ActivityExecutePipelineDAG runs a pipeline DAG synchronously via
// PipelineEngine.ExecuteRun. Registered as a Temporal Activity on the
// deployed worker (cmd/worker) so pipeline runs started via
// PipelineEngine.ExecuteRunAsWorkflow get Temporal's durability (retries,
// history, visibility) around the same DAG-walking logic used for
// synchronous runs.
func (a *PipelineActivities) ActivityExecutePipelineDAG(ctx context.Context, input PipelineWorkflowInput) (*PipelineExecutionRun, error) {
	if a.Engine == nil {
		return nil, temporal.NewNonRetryableApplicationError("pipeline engine not configured", "ConfigError", nil)
	}
	runID := input.RunID
	if runID == uuid.Nil {
		runID = uuid.New()
	}
	return a.Engine.executeRunWithRunID(ctx, input.TenantID, runID, input.Definition, input.InputRecords, false, input.TriggerID)
}

// RunPipelineDAGWorkflow is the thin Temporal workflow wrapper that gives a
// pipeline run durable execution: it delegates all real work to
// ActivityExecutePipelineDAG so the DB I/O and in-process fan-out in
// PipelineEngine.ExecuteRun never runs directly on the workflow goroutine.
func RunPipelineDAGWorkflow(ctx workflow.Context, input PipelineWorkflowInput) (*PipelineExecutionRun, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1, // pipeline runs are not safely re-runnable by default (writes may not be idempotent)
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result PipelineExecutionRun
	// Referenced by registered activity name (see cmd/worker/main.go's
	// w.RegisterActivity(pipelineActivities.ActivityExecutePipelineDAG))
	// rather than a method value, since no *PipelineActivities instance is
	// available inside workflow code.
	err := workflow.ExecuteActivity(ctx, "ActivityExecutePipelineDAG", input).Get(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
