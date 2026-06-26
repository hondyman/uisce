package workflows

import (
	"fmt"
	"time"

	"github.com/hondyman/semlayer/services/wealth-management/activities"
	"go.temporal.io/sdk/workflow"
)

// AttributionAlpha executes the killer performance attribution workflow with AI and ABAC governance
func AttributionAlpha(ctx workflow.Context, portfolioID string) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})

	// 1. AI Attribution
	var attr map[string]any
	err := workflow.ExecuteActivity(ctx, activities.AIAttribution, portfolioID).Get(ctx, &attr)
	if err != nil {
		return fmt.Errorf("AI attribution failed: %w", err)
	}

	// 2. ABAC + Temporal Policy Check
	var allowed bool
	err = workflow.ExecuteActivity(ctx, activities.ABACCheck, "attribute", "portfolio", portfolioID).Get(ctx, &allowed)
	if err != nil {
		return fmt.Errorf("ABAC check failed: %w", err)
	}
	if !allowed {
		return fmt.Errorf("ABAC denied attribution for portfolio %s", portfolioID)
	}

	// 3. Execute Attribution
	if err := workflow.ExecuteActivity(ctx, activities.ExecuteAttribution, attr).Get(ctx, nil); err != nil {
		return fmt.Errorf("attribution execution failed: %w", err)
	}

	// 4. Update Hasura
	update := map[string]any{
		"portfolio_id": portfolioID,
		"status":       "alpha_attributed",
		"alpha":        attr["alpha"],
		"sector":       attr["sector"],
	}
	if err := workflow.ExecuteActivity(ctx, activities.HasuraUpdate, update).Get(ctx, nil); err != nil {
		return fmt.Errorf("Hasura update failed: %w", err)
	}

	return nil
}
