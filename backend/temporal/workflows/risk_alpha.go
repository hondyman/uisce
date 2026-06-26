package workflows

import (
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/temporal/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// RiskAlpha is a workflow that manages portfolio risk.
func RiskAlpha(ctx workflow.Context, portfolioID string) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
	})

	// 1. AI Risk Score
	var risk map[string]any
	err := workflow.ExecuteActivity(ctx, activities.AIRiskScore, portfolioID).Get(ctx, &risk)
	if err != nil {
		return err
	}

	// 2. ABAC + Temporal Policy
	var allowed bool
	err = workflow.ExecuteActivity(ctx, activities.ABACCheck, "risk", "portfolio", portfolioID).Get(ctx, &allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("ABAC denied")
	}

	// 3. Execute Mitigation
	err = workflow.ExecuteActivity(ctx, activities.ExecuteMitigation, risk).Get(ctx, nil)
	if err != nil {
		return err
	}

	// 4. Update
	err = workflow.ExecuteActivity(ctx, activities.HasuraUpdate, map[string]any{
		"portfolio_id": portfolioID,
		"status":       "risk_managed",
		"risk_score":   risk["score"],
		"mitigation":   risk["action"],
	}).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
