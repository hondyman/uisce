package workflows

import (
	"time"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// TenantLakehouseRetentionWorkflowName is the registered name the API starts.
const TenantLakehouseRetentionWorkflowName = "TenantLakehouseRetentionWorkflow"

// LakehouseRetentionResult is what a reconcile reports.
type LakehouseRetentionResult struct {
	TenantID       string
	AppliedDays    int
	AlreadyCurrent bool
}

// TenantLakehouseRetentionWorkflow brings a provisioned tenant's bucket up to the audit
// retention the registry wants (ADR-032). It only ever raises: the registry cannot lower a set
// retention, and the bucket operation cannot lower an enforced one. It is idempotent, so a
// retry or a second run is safe, and a bucket that already enforces enough is left alone.
//
// It changes the bucket's DEFAULT retention, which applies to objects written from then on.
// Objects already in the bucket keep the retain-until they were stamped with.
//
// A failure is written to the tenant's audit trail as retention_sync_failed, with the step and
// the reason, so an admin can see that the bucket is still behind the registry and why.
func TenantLakehouseRetentionWorkflow(ctx workflow.Context, in activities.LakehouseProvisionInput) (res *LakehouseRetentionResult, err error) {
	logger := workflow.GetLogger(ctx)
	acts := &activities.TenantLakehouseActivities{} // method references only
	result := &LakehouseRetentionResult{TenantID: in.TenantID}

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &sdktemporal.RetryPolicy{
			MaximumAttempts:    5,
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
		},
	})

	step := "LoadRetentionTarget"
	defer func() {
		if err == nil && ctx.Err() == nil {
			return
		}
		reason := "cancelled"
		if err != nil {
			reason = err.Error()
		}
		dc, _ := workflow.NewDisconnectedContext(ctx)
		dc = workflow.WithActivityOptions(dc, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
			RetryPolicy:         &sdktemporal.RetryPolicy{MaximumAttempts: 3, InitialInterval: 2 * time.Second},
		})
		if rerr := workflow.ExecuteActivity(dc, acts.RecordRetentionSyncFailure, in, step, reason).Get(dc, nil); rerr != nil {
			logger.Error("could not record the retention sync failure", "step", step, "error", rerr)
		}
	}()

	var target activities.RetentionTarget
	if err = workflow.ExecuteActivity(ctx, acts.LoadRetentionTarget, in).Get(ctx, &target); err != nil {
		return result, err
	}
	if !target.Pending {
		result.AlreadyCurrent = true
		result.AppliedDays = target.DesiredDays
		return result, nil
	}

	step = "ExtendBucketRetention"
	var applied int
	if err = workflow.ExecuteActivity(ctx, acts.ExtendBucketRetention, in, target.DesiredDays).Get(ctx, &applied); err != nil {
		return result, err
	}

	step = "MarkRetentionApplied"
	if err = workflow.ExecuteActivity(ctx, acts.MarkRetentionApplied, in, applied).Get(ctx, nil); err != nil {
		return result, err
	}

	result.AppliedDays = applied
	logger.Info("tenant lakehouse retention applied", "tenant", in.TenantID, "days", applied)
	return result, nil
}
