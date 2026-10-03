package workflows

import (
	"time"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// TenantLakehouseProvisioningWorkflowName is the registered name the API starts.
const TenantLakehouseProvisioningWorkflowName = "TenantLakehouseProvisioningWorkflow"

// LakehouseProvisionResult is what a run reports. It carries no secret.
type LakehouseProvisionResult struct {
	TenantID           string
	WarehouseID        string
	AlreadyProvisioned bool
}

// TenantLakehouseProvisioningWorkflow provisions a tenant's one Iceberg warehouse
// (ADR-032): KMS key, WORM bucket, bucket-scoped credential, Lakekeeper warehouse, then
// the registry row.
//
// There is no compensation, deliberately. The bucket holds write-once audit and a failed
// run must never delete it; every step is idempotent, so the next run resumes where this
// one stopped (forward recovery). A failure is written to the tenant's audit trail with
// the step and reason, so an admin can see why a tenant is still waiting.
func TenantLakehouseProvisioningWorkflow(ctx workflow.Context, in activities.LakehouseProvisionInput) (res *LakehouseProvisionResult, err error) {
	logger := workflow.GetLogger(ctx)
	acts := &activities.TenantLakehouseActivities{} // method references only
	result := &LakehouseProvisionResult{TenantID: in.TenantID}

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &sdktemporal.RetryPolicy{
			MaximumAttempts:    5,
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
		},
	})

	step := "LoadLakehouseSpec"
	defer func() {
		if err == nil && ctx.Err() == nil {
			return
		}
		reason := "cancelled"
		if err != nil {
			reason = err.Error()
		}
		// A disconnected context so the record is written even when the run was
		// cancelled. Best effort: it must never replace the original error.
		dc, _ := workflow.NewDisconnectedContext(ctx)
		dc = workflow.WithActivityOptions(dc, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
			RetryPolicy:         &sdktemporal.RetryPolicy{MaximumAttempts: 3, InitialInterval: 2 * time.Second},
		})
		if rerr := workflow.ExecuteActivity(dc, acts.RecordLakehouseFailure, in, step, reason).Get(dc, nil); rerr != nil {
			logger.Error("could not record the lakehouse provisioning failure", "step", step, "error", rerr)
		}
	}()

	var spec activities.LakehouseSpec
	if err = workflow.ExecuteActivity(ctx, acts.LoadLakehouseSpec, in).Get(ctx, &spec); err != nil {
		return result, err
	}
	if spec.AlreadyProvisioned {
		result.AlreadyProvisioned = true
		return result, nil
	}

	step = "EnsureLakehouseKey"
	var keyID string
	if err = workflow.ExecuteActivity(ctx, acts.EnsureLakehouseKey, in).Get(ctx, &keyID); err != nil {
		return result, err
	}

	step = "EnsureLakehouseBucket"
	if err = workflow.ExecuteActivity(ctx, acts.EnsureLakehouseBucket, in, keyID, spec.RetentionDays).Get(ctx, nil); err != nil {
		return result, err
	}

	step = "EnsureLakehouseCredential"
	if err = workflow.ExecuteActivity(ctx, acts.EnsureLakehouseCredential, in).Get(ctx, nil); err != nil {
		return result, err
	}

	step = "EnsureLakehouseWarehouse"
	var warehouseID string
	if err = workflow.ExecuteActivity(ctx, acts.EnsureLakehouseWarehouse, in).Get(ctx, &warehouseID); err != nil {
		return result, err
	}

	step = "MarkLakehouseProvisioned"
	if err = workflow.ExecuteActivity(ctx, acts.MarkLakehouseProvisioned, in, warehouseID, keyID).Get(ctx, nil); err != nil {
		return result, err
	}

	result.WarehouseID = warehouseID
	logger.Info("tenant lakehouse provisioned", "tenant", in.TenantID, "warehouse", warehouseID)
	return result, nil
}
