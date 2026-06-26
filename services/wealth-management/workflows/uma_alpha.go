package workflows

import (
	"fmt"
	"time"

	"github.com/hondyman/semlayer/services/wealth-management/activities"
	"go.temporal.io/sdk/workflow"
)

// UMAAlpha executes the killer UMA rebalance workflow with AI tax harvesting and ABAC governance
func UMAAlpha(ctx workflow.Context, umaID string) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
	})

	// 1. AI Harvest
	var harvest map[string]any
	err := workflow.ExecuteActivity(ctx, activities.AITaxHarvest, umaID).Get(ctx, &harvest)
	if err != nil {
		return fmt.Errorf("AI tax harvest failed: %w", err)
	}

	// 2. ABAC + Temporal Policy Check
	var allowed bool
	err = workflow.ExecuteActivity(ctx, activities.ABACCheck, "rebalance", "uma", umaID).Get(ctx, &allowed)
	if err != nil {
		return fmt.Errorf("ABAC check failed: %w", err)
	}
	if !allowed {
		return fmt.Errorf("ABAC denied rebalance for UMA %s", umaID)
	}

	// 3. Execute Trades
	if err := workflow.ExecuteActivity(ctx, activities.ExecuteTrades, harvest).Get(ctx, nil); err != nil {
		return fmt.Errorf("trade execution failed: %w", err)
	}

	// 4. Update Hasura
	update := map[string]any{
		"uma_id":    umaID,
		"status":    "alpha_rebalanced",
		"tax_saved": harvest["saved"],
	}
	if err := workflow.ExecuteActivity(ctx, activities.HasuraUpdate, update).Get(ctx, nil); err != nil {
		return fmt.Errorf("Hasura update failed: %w", err)
	}

	return nil
}
