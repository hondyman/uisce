package main

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func RiskAlphaWorkflow(ctx workflow.Context, portfolioID string) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
	})

	// 1. AI Risk Score
	var riskResult map[string]interface{}
	if err := workflow.ExecuteActivity(ctx, "AIRiskScore", portfolioID).Get(ctx, &riskResult); err != nil {
		return fmt.Errorf("failed to get AI risk score: %w", err)
	}

	// 2. ABAC + Temporal Policy
	var allowed bool
	if err := workflow.ExecuteActivity(ctx, "ABACCheck", "risk", "portfolio", portfolioID).Get(ctx, &allowed); err != nil || !allowed {
		return fmt.Errorf("ABAC denied for risk management: %w", err)
	}

	// 3. Execute Mitigation
	if err := workflow.ExecuteActivity(ctx, "ExecuteMitigation", riskResult).Get(ctx, nil); err != nil {
		return fmt.Errorf("failed to execute mitigation: %w", err)
	}

	// 4. Update Status in Hasura
	if err := workflow.ExecuteActivity(ctx, "UpdateRiskStatus", portfolioID, riskResult).Get(ctx, nil); err != nil {
		return fmt.Errorf("failed to update risk status: %w", err)
	}

	return nil
}
