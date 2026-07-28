package temporal

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func TenantOnboardingWorkflow(ctx workflow.Context, p TenantProvisionParams) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    3,
			BackoffCoefficient: 2.0,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var act *TenantActivities
	var step1Done, step2Done bool

	// Define compensating actions via deferred execution
	defer func() {
		if ctx.Err() != nil {
			disconnectedCtx, _ := workflow.NewDisconnectedContext(ctx)

			if step2Done {
				_ = workflow.ExecuteActivity(disconnectedCtx, act.RollbackMinIOPrefix, p).Get(disconnectedCtx, nil)
			}
			if step1Done {
				_ = workflow.ExecuteActivity(disconnectedCtx, act.RollbackPostgresTenant, p).Get(disconnectedCtx, nil)
			}
		}
	}()

	// Step 1: PostgreSQL Tenant Record
	err := workflow.ExecuteActivity(ctx, act.CreatePostgresTenant, p).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("saga failed at step 1 (postgres): %w", err)
	}
	step1Done = true

	// Step 2: MinIO Storage Prefix Initialization
	err = workflow.ExecuteActivity(ctx, act.InitializeMinIOPrefix, p).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("saga failed at step 2 (minio): %w", err)
	}
	step2Done = true

	// Step 3: Apache Polaris Catalog Provisioning
	err = workflow.ExecuteActivity(ctx, act.ProvisionPolarisCatalog, p).Get(ctx, nil)
	if err != nil {
		_ = workflow.ExecuteActivity(ctx, act.RollbackMinIOPrefix, p).Get(ctx, nil)
		_ = workflow.ExecuteActivity(ctx, act.RollbackPostgresTenant, p).Get(ctx, nil)
		return fmt.Errorf("saga failed at step 3 (polaris): %w", err)
	}

	return nil
}
