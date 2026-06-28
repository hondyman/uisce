package main

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func AttributionAlphaWorkflow(ctx workflow.Context, portfolioID string) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
	})

	// 1. AI Attribution
	var attrResult map[string]interface{}
	if err := workflow.ExecuteActivity(ctx, "AIAttribution", portfolioID).Get(ctx, &attrResult); err != nil {
		return fmt.Errorf("failed to get AI attribution: %w", err)
	}

	// 2. ABAC Check
	var allowed bool
	if err := workflow.ExecuteActivity(ctx, "ABACCheck", "attribute", "portfolio", portfolioID).Get(ctx, &allowed); err != nil || !allowed {
		return fmt.Errorf("ABAC denied for attribution: %w", err)
	}

	// 3. Execute Attribution (Placeholder)
	if err := workflow.ExecuteActivity(ctx, "ExecuteAttribution", attrResult).Get(ctx, nil); err != nil {
		return fmt.Errorf("failed to execute attribution: %w", err)
	}

	// 4. Update Status in Hasura
	if err := workflow.ExecuteActivity(ctx, "UpdateAttributionStatus", portfolioID, attrResult).Get(ctx, nil); err != nil {
		return fmt.Errorf("failed to update attribution status: %w", err)
	}

	return nil
}
