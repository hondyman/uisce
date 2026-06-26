package workflows

import (
	"fmt"
	"time"

	"github.com/hondyman/semlayer/services/wealth-management/activities"
	"go.temporal.io/sdk/workflow"
)

// TaxHarvest executes the AI-powered tax optimization workflow for UMA accounts
func TaxHarvest(ctx workflow.Context, umaID string) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 60 * time.Second,
	})

	// 1. AI Tax Harvest Analysis
	var harvest map[string]any
	err := workflow.ExecuteActivity(ctx, activities.AITaxHarvest, umaID).Get(ctx, &harvest)
	if err != nil {
		return fmt.Errorf("AI tax harvest failed: %w", err)
	}

	// 2. ABAC + Temporal Policy Check
	var allowed bool
	err = workflow.ExecuteActivity(ctx, activities.ABACCheck, "harvest", "uma", umaID).Get(ctx, &allowed)
	if err != nil {
		return fmt.Errorf("ABAC check failed: %w", err)
	}
	if !allowed {
		return fmt.Errorf("ABAC denied tax harvest for UMA %s", umaID)
	}

	// 3. Execute Tax Harvest
	if err := workflow.ExecuteActivity(ctx, activities.ExecuteHarvest, harvest).Get(ctx, nil); err != nil {
		return fmt.Errorf("tax harvest execution failed: %w", err)
	}

	// 4. Update Hasura with tax optimization results
	update := map[string]any{
		"uma_id":        umaID,
		"status":        "tax_optimized",
		"tax_saved":     harvest["saved"],
		"lots_selected": harvest["lots"],
		"esg_score":     harvest["esg_score"],
	}
	if err := workflow.ExecuteActivity(ctx, activities.HasuraUpdate, update).Get(ctx, nil); err != nil {
		return fmt.Errorf("Hasura update failed: %w", err)
	}

	return nil
}
