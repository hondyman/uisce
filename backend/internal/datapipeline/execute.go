package datapipeline

import (
	"context"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// TaskQueue is where the API server's in-process worker executes loads (it
// owns the dependencies: rule service, enforced BO writes, staging DB).
const TaskQueue = "data_pipeline_queue"

// Execute runs a queued run end to end and records the outcome. The run's
// own spec snapshot is executed, never the pipeline's current version.
func (s *Store) Execute(ctx context.Context, deps Deps, tenantID, runID string) (*Summary, error) {
	spec, err := s.RunSpec(ctx, tenantID, runID)
	if err != nil {
		return nil, err
	}
	if err := s.MarkRunning(ctx, tenantID, runID); err != nil {
		return nil, err
	}
	rec := s.Recorder(tenantID, runID).(*dbRecorder)
	sum, runErr := Run(ctx, spec, &RunContext{RunID: runID, TenantID: tenantID}, deps, rec)
	done := context.WithoutCancel(ctx)
	_ = rec.Flush(done)
	if err := s.FinishRun(done, tenantID, runID, sum, runErr); err != nil && runErr == nil {
		runErr = err
	}
	return sum, runErr
}

// RunInput identifies a queued run.
type RunInput struct {
	TenantID string
	RunID    string
}

// Workflow executes one pipeline run as a single activity. Loads are not
// retried automatically: a create-mode BO load is not idempotent.
func Workflow(ctx workflow.Context, in RunInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 12 * time.Hour,
		HeartbeatTimeout:    2 * time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	})
	return workflow.ExecuteActivity(ctx, ActivityName, in).Get(ctx, nil)
}

// ActivityName is the registered name of Activities.Run.
const ActivityName = "DataPipelineRun"

// Activities hosts the run activity with its dependencies.
type Activities struct {
	Store *Store
	Deps  Deps
}

func (a *Activities) Run(ctx context.Context, in RunInput) error {
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		t := time.NewTicker(20 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				activity.RecordHeartbeat(ctx)
			}
		}
	}()
	_, err := a.Store.Execute(ctx, a.Deps, in.TenantID, in.RunID)
	return err
}
