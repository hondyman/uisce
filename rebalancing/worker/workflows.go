package main

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func RebalanceAlphaWorkflow(ctx workflow.Context, portfolioID string) error {
	logger := workflow.GetLogger(ctx)

	opts := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, opts)

	// Step 1: Fetch portfolio
	var portfolio Portfolio
	if err := workflow.ExecuteActivity(ctx, "FetchPortfolio", portfolioID).Get(ctx, &portfolio); err != nil {
		return err
	}

	// Step 2: Get real-time prices
	symbols := make([]string, len(portfolio.Holdings))
	for i, h := range portfolio.Holdings {
		symbols[i] = h.Symbol
	}
	var prices map[string]float64
	if err := workflow.ExecuteActivity(ctx, "FetchRealTimePrices", symbols).Get(ctx, &prices); err != nil {
		return err
	}

	// Update portfolio with live prices
	for i, h := range portfolio.Holdings {
		if price, ok := prices[h.Symbol]; ok {
			portfolio.Holdings[i].CurrentPrice = price
		}
	}

	// Step 3: Analyze drift with live data
	var drift float64
	if err := workflow.ExecuteActivity(ctx, "AnalyzeDrift", portfolio).Get(ctx, &drift); err != nil {
		return err
	}

	logger.Info("Drift analyzed with live data", "portfolioID", portfolioID, "drift", drift)

	// Early exit if within tolerance
	if drift < 5.0 {
		logger.Info("Drift within tolerance, skipping")
		return nil
	}

	// Step 4: ABAC check
	var allowed bool
	if err := workflow.ExecuteActivity(ctx, "ABACCheck", "rebalance", "portfolio", portfolioID).Get(ctx, &allowed); err != nil || !allowed {
		return fmt.Errorf("ABAC denied: %w", err)
	}

	// Step 5: AI rebalance
	var plan *RebalancePlan
	if err := workflow.ExecuteActivity(ctx, "AIRebalance", portfolio).Get(ctx, &plan); err != nil {
		return err
	}

	logger.Info("AI plan generated", "trades", len(plan.ProposedTrades))

	// Step 6: Validate
	if err := workflow.ExecuteActivity(ctx, "ValidatePlan", plan).Get(ctx, nil); err != nil {
		logger.Error("Validation failed", "error", err)
		workflow.ExecuteActivity(ctx, "NotifyStakeholders", plan, "validation_failed")
		return err
	}

	// Step 7: Insert plan to database
	if err := workflow.ExecuteActivity(ctx, "InsertRebalancePlan", plan).Get(ctx, nil); err != nil {
		return err
	}

	// Step 8: Execute trades
	var orderIDs []string
	if err := workflow.ExecuteActivity(ctx, "ExecuteTrades", plan).Get(ctx, &orderIDs); err != nil {
		logger.Error("Trade execution failed", "error", err)
		workflow.ExecuteActivity(ctx, "NotifyStakeholders", plan, "execution_failed")
		return err
	}

	logger.Info("Trades executed", "orders", len(orderIDs))

	// Step 9: Update portfolio state
	if err := workflow.ExecuteActivity(ctx, "UpdatePortfolioState", portfolioID, plan).Get(ctx, nil); err != nil {
		return err
	}

	// Step 10: Notify
	workflow.ExecuteActivity(ctx, "NotifyStakeholders", plan, "completed")

	// Step 11: Generate and save AI summary
	var summary string
	if err := workflow.ExecuteActivity(ctx, "GenerateRebalanceSummary", *plan).Get(ctx, &summary); err == nil {
		plan.Summary = summary // Update the plan object in the workflow
		// Get the plan ID from the database after it has been inserted
		// For simplicity, we assume the plan object has an ID after insertion.
		// A more robust way would be to return the ID from InsertRebalancePlan activity.
		if plan.ID != "" { // Assuming plan ID is available
			wfErr := workflow.ExecuteActivity(ctx, "UpdatePlanWithSummary", plan.ID, summary).Get(ctx, nil)
			if wfErr != nil {
				logger.Error("Failed to update plan with summary", "error", wfErr)
			}
		}
	} else {
		logger.Error("Failed to generate rebalance summary", "error", err)
	}

	logger.Info("Rebalance completed successfully")
	return nil
}
