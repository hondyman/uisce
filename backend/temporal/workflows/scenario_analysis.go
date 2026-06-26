package workflows

import (
	"time"

	"github.com/hondyman/semlayer/backend/temporal/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ScenarioAnalysis is a workflow that analyzes a portfolio under a given scenario.
func ScenarioAnalysis(ctx workflow.Context, portfolioID string, scenario string) (map[string]any, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
	})

	var result map[string]any
	err := workflow.ExecuteActivity(ctx, activities.AIScenario, portfolioID, scenario).Get(ctx, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
