package workflows

import (
	"time"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// PlatformLakehouseProvisioningWorkflowName is the registered name an operator starts. It is started on purpose, with
// an explicit retention, and never at worker boot: the retention cannot be shortened afterwards.
const PlatformLakehouseProvisioningWorkflowName = "PlatformLakehouseProvisioningWorkflow"

// PlatformLakehouseProvisionResult is what a run reports. It carries no secret.
type PlatformLakehouseProvisionResult struct {
	WarehouseID        string
	AlreadyProvisioned bool
}

// PlatformLakehouseProvisioningWorkflow provisions the platform warehouse ivy-control (ADR-049): KMS key, WORM
// bucket, bucket-scoped credential, Lakekeeper warehouse, then the registry row. Like a tenant's, there is no
// compensation: the bucket holds write-once audit and a failed run must never delete it. Every step is idempotent,
// so the next run resumes where this one stopped.
func PlatformLakehouseProvisioningWorkflow(ctx workflow.Context, in activities.PlatformLakehouseInput) (*PlatformLakehouseProvisionResult, error) {
	logger := workflow.GetLogger(ctx)
	acts := &activities.PlatformLakehouseActivities{} // method references only
	result := &PlatformLakehouseProvisionResult{}

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &sdktemporal.RetryPolicy{
			MaximumAttempts:    5,
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
		},
	})

	var spec activities.PlatformLakehouseSpec
	if err := workflow.ExecuteActivity(ctx, acts.ConfigurePlatformLakehouse, in).Get(ctx, &spec); err != nil {
		return result, err
	}
	if spec.AlreadyProvisioned {
		result.AlreadyProvisioned = true
		return result, nil
	}

	var keyID string
	if err := workflow.ExecuteActivity(ctx, acts.EnsurePlatformKey).Get(ctx, &keyID); err != nil {
		return result, err
	}
	if err := workflow.ExecuteActivity(ctx, acts.EnsurePlatformBucket, keyID, spec.RetentionDays).Get(ctx, nil); err != nil {
		return result, err
	}
	if err := workflow.ExecuteActivity(ctx, acts.EnsurePlatformCredential).Get(ctx, nil); err != nil {
		return result, err
	}
	var warehouseID string
	if err := workflow.ExecuteActivity(ctx, acts.EnsurePlatformWarehouse).Get(ctx, &warehouseID); err != nil {
		return result, err
	}
	if err := workflow.ExecuteActivity(ctx, acts.MarkPlatformProvisioned, warehouseID, keyID, spec.RetentionDays).Get(ctx, nil); err != nil {
		return result, err
	}

	result.WarehouseID = warehouseID
	logger.Info("platform lakehouse provisioned", "warehouse", warehouseID, "by", in.ActorID)
	return result, nil
}
