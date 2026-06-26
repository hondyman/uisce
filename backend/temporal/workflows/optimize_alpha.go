package workflows

import (
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/temporal/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// OptimizeAlpha is a workflow that optimizes a portfolio.
func OptimizeAlpha(ctx workflow.Context, portfolioID string) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
	})

	// 1. AI Optimize
	var opt map[string]any
	err := workflow.ExecuteActivity(ctx, activities.AIOptimize, portfolioID).Get(ctx, &opt)
	if err != nil {
		return err
	}

	// 2. ABAC + Temporal Policy
	var allowed bool
	err = workflow.ExecuteActivity(ctx, activities.ABACCheck, "optimize", "portfolio", portfolioID).Get(ctx, &allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("ABAC denied")
	}

	// 3. Execute
	err = workflow.ExecuteActivity(ctx, activities.ExecuteTrades, opt).Get(ctx, nil)
	if err != nil {
		return err
	}

	// 4. Update
	err = workflow.ExecuteActivity(ctx, activities.HasuraUpdate, map[string]any{
		"portfolio_id": portfolioID,
		"status":       "alpha_optimized",
		"sharpe":       opt["sharpe"],
		"risk":         opt["risk"],
	}).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
