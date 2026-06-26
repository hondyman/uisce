package recon

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ReconBreak represents a discrepancy found by the matcher
type ReconBreak struct {
	ID               string  `json:"id"`
	TenantID         string  `json:"tenant_id"`
	AccountID        string  `json:"account_id"`
	AssetID          string  `json:"asset_id"`
	InternalQuantity float64 `json:"internal_quantity"`
	ExternalQuantity float64 `json:"external_quantity"`
}

// ReconExceptionWorkflow manages the lifecycle of a reconciliation break
func ReconExceptionWorkflow(ctx workflow.Context, breakDetails ReconBreak) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting ReconExceptionWorkflow", "BreakID", breakDetails.ID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumInterval: time.Second * 10,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Persist Break to DB
	err := workflow.ExecuteActivity(ctx, CreateReconExceptionActivity, breakDetails).Get(ctx, nil)
	if err != nil {
		return err
	}

	// Step 2: Create Human Task for Investigation
	taskID := fmt.Sprintf("task-%s", breakDetails.ID)
	err = workflow.ExecuteActivity(ctx, CreateHumanTaskActivity, taskID, "Investigate Recon Break").Get(ctx, nil)
	if err != nil {
		return err
	}

	// Step 3: Wait for Resolution Signal
	// The human operator investigates and signals "Resolved" or "Ignored"
	var resolution string
	selector := workflow.NewSelector(ctx)
	selector.AddReceive(workflow.GetSignalChannel(ctx, "ResolveBreak"), func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &resolution)
	})

	// Wait indefinitely for resolution (or add a timeout/SLA)
	logger.Info("Waiting for human resolution...")
	selector.Select(ctx)

	logger.Info("Break Resolved", "Resolution", resolution)

	// Step 4: Update Status
	err = workflow.ExecuteActivity(ctx, UpdateReconExceptionStatusActivity, breakDetails.ID, resolution).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}

// --- Activities ---

func CreateReconExceptionActivity(ctx context.Context, b ReconBreak) error {
	fmt.Printf("Creating Recon Exception: %+v\n", b)
	// INSERT INTO recon_exceptions ...
	return nil
}

func CreateHumanTaskActivity(ctx context.Context, taskID, description string) error {
	fmt.Printf("Creating Human Task: %s - %s\n", taskID, description)
	// INSERT INTO human_tasks ...
	return nil
}

func UpdateReconExceptionStatusActivity(ctx context.Context, id, status string) error {
	fmt.Printf("Updating Recon Exception %s to %s\n", id, status)
	// UPDATE recon_exceptions SET status = $2 WHERE id = $1
	return nil
}
