package workflows

import (
	"time"

	"github.com/hondyman/uisce/backend/internal/mdm"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// DownstreamGoldSyncWorkflow coordinates resilient distribution to all target bindings
func DownstreamGoldSyncWorkflow(ctx workflow.Context, req mdm.DownstreamSyncRequest) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting DownstreamGoldSyncWorkflow", "tenant_id", req.TenantID, "entity_sid", req.EntitySID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    5,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Discover all active downstream bindings for this Business Object
	var targets []mdm.BindingTargetDescriptor
	err := workflow.ExecuteActivity(ctx, "ResolveTargetBindingsActivity", req.TenantID, req.BOID).Get(ctx, &targets)
	if err != nil {
		logger.Error("Failed resolving target bindings", "error", err)
		return err
	}

	// Step 2: Fan-out parallel transformations and delivery to each target binding
	futures := make([]workflow.Future, 0, len(targets))
	for _, target := range targets {
		f := workflow.ExecuteActivity(ctx, "TransformAndDispatchActivity", req.TenantID, req.BOID, target, req.EntitySID, req.GoldAttributes)
		futures = append(futures, f)
	}

	// Step 3: Await all child dispatch tasks and collect receipts
	for i, f := range futures {
		var syncResult struct {
			TargetName string `json:"targetName"`
			Status     string `json:"status"`
			Checksum   string `json:"checksum"`
		}
		if err := f.Get(ctx, &syncResult); err != nil {
			logger.Warn("Partial sync failure on target", "target", targets[i].TargetName, "error", err)
		} else {
			logger.Info("Target sync successful", "target", syncResult.TargetName, "checksum", syncResult.Checksum)
		}
	}

	return nil
}
