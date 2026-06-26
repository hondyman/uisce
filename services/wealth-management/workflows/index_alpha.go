package workflows

import (
	"fmt"
	"time"

	"github.com/hondyman/semlayer/services/wealth-management/activities"
	"go.temporal.io/sdk/workflow"
)

// IndexAlpha executes the AI-powered direct indexing optimization workflow
func IndexAlpha(ctx workflow.Context, indexID string) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
	})

	// 1. AI Index Optimization Analysis
	var opt map[string]any
	err := workflow.ExecuteActivity(ctx, activities.AIIndexOptimize, indexID).Get(ctx, &opt)
	if err != nil {
		return fmt.Errorf("AI index optimization failed: %w", err)
	}

	// 2. ABAC + Temporal Policy Check
	var allowed bool
	err = workflow.ExecuteActivity(ctx, activities.ABACCheck, "optimize", "index", indexID).Get(ctx, &allowed)
	if err != nil {
		return fmt.Errorf("ABAC check failed: %w", err)
	}
	if !allowed {
		return fmt.Errorf("ABAC denied index optimization for %s", indexID)
	}

	// 3. Execute Index Optimization Trades
	if err := workflow.ExecuteActivity(ctx, activities.ExecuteTrades, opt).Get(ctx, nil); err != nil {
		return fmt.Errorf("index optimization execution failed: %w", err)
	}

	// 4. Update Hasura with optimization results
	update := map[string]any{
		"index_id":  indexID,
		"status":    "alpha_optimized",
		"drift":     opt["drift"],
		"tax_saved": opt["saved"],
		"esg_score": opt["esg_score"],
		"holdings":  opt["holdings"],
	}
	if err := workflow.ExecuteActivity(ctx, activities.HasuraUpdate, update).Get(ctx, nil); err != nil {
		return fmt.Errorf("Hasura update failed: %w", err)
	}

	return nil
}
