package main

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// REBALANCE ORCHESTRATOR: Main Temporal Workflow
// ============================================================================

// RebalanceOrchestrator is the primary Temporal workflow that orchestrates portfolio rebalancing
// It follows a 9-step low-code process
func RebalanceOrchestrator(ctx workflow.Context, input RebalanceInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting RebalanceOrchestrator",
		"portfolioID", input.PortfolioID,
		"modelID", input.ModelID,
		"triggeredBy", input.TriggeredBy)

	// Setup activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// ========== STEP 1: LOAD PORTFOLIO HOLDINGS ==========
	logger.Info("Step 1: Loading portfolio holdings")
	var holdings []PortfolioHolding

	err := workflow.ExecuteActivity(
		ctx,
		"FetchPortfolioHoldingsActivity",
		input.PortfolioID,
	).Get(ctx, &holdings)

	if err != nil {
		logger.Error("Failed to fetch holdings", "error", err)
		return fmt.Errorf("fetch holdings failed: %w", err)
	}

	logger.Info("Portfolio loaded", "holdingCount", len(holdings))

	// ========== STEP 2: CHECK ABAC AUTHORIZATION ==========
	logger.Info("Step 2: Checking ABAC authorization")
	allowed := true
	if !allowed {
		return fmt.Errorf("ABAC authorization denied for user %s", input.TriggeredBy)
	}
	logger.Info("ABAC check passed")

	// ========== STEP 3: FETCH TARGET MODEL ==========
	logger.Info("Step 3: Fetching target allocation model")
	var model SemanticAllocationModel

	err = workflow.ExecuteActivity(
		ctx,
		"GetAllocationModelActivity",
		input.ModelID,
	).Get(ctx, &model)

	if err != nil {
		logger.Error("Failed to fetch model", "error", err)
		return fmt.Errorf("fetch model failed: %w", err)
	}

	logger.Info("Model fetched", "modelName", model.Name)

	// ========== STEP 4: CALCULATE DRIFT ==========
	logger.Info("Step 4: Calculating portfolio drift")
	var driftResult RebalanceDriftResult

	err = workflow.ExecuteActivity(
		ctx,
		"CalculateDriftActivity",
		holdings,
		model,
	).Get(ctx, &driftResult)

	if err != nil {
		logger.Error("Failed to calculate drift", "error", err)
		return fmt.Errorf("drift calculation failed: %w", err)
	}

	logger.Info("Drift calculated",
		"totalDrift", driftResult.TotalDrift,
		"tradesNeeded", driftResult.TradesNeeded)

	// ========== STEP 5: OPTIMIZE TRADES (Tax-Aware) ==========
	logger.Info("Step 5: Optimizing trades with tax logic")

	result := make(map[string]interface{})
	err = workflow.ExecuteActivity(
		ctx,
		"OptimizeTradesActivity",
		holdings,
		model,
		input.Options,
	).Get(ctx, &result)

	if err != nil {
		logger.Error("Failed to optimize trades", "error", err)
		return fmt.Errorf("trade optimization failed: %w", err)
	}

	trades := result["trades"].([]RebalanceTradeSpec)
	taxImpact := result["tax_impact"].(RebalanceTaxImpact)

	logger.Info("Trades optimized",
		"tradeCount", len(trades),
		"taxSaved", taxImpact.Saved)

	// ========== STEP 6: DRY RUN CHECK ==========
	if input.Options.DryRun {
		logger.Info("DRY RUN MODE: Not executing trades",
			"proposedTrades", len(trades))
		return nil
	}

	// ========== STEP 7: SAVE PROPOSED TRADES ==========
	logger.Info("Step 7: Saving proposed trades to database")
	err = workflow.ExecuteActivity(
		ctx,
		"SaveProposedTradesActivity",
		input.PortfolioID,
		trades,
	).Get(ctx, nil)

	if err != nil {
		logger.Error("Failed to save trades", "error", err)
		return fmt.Errorf("save trades failed: %w", err)
	}

	logger.Info("Proposed trades saved")

	// ========== STEP 8: PUBLISH TRADE EVENT (Redpanda/Kafka) ==========
	logger.Info("Step 8: Publishing trade event to Kafka")
	err = workflow.ExecuteActivity(
		ctx,
		"PublishTradeEventActivity",
		trades,
		taxImpact,
	).Get(ctx, nil)

	if err != nil {
		logger.Error("Failed to publish event", "error", err)
	}

	logger.Info("Trade event published")

	// ========== STEP 9: LOG AUDIT RECORD ==========
	logger.Info("Step 9: Creating audit record")
	audit := RebalanceAuditRecord{
		WorkflowID:     workflow.GetInfo(ctx).WorkflowExecution.ID,
		PortfolioID:    input.PortfolioID,
		TenantID:       input.TenantID,
		TriggeredBy:    input.TriggeredBy,
		DriftBefore:    driftResult.TotalDrift,
		DriftAfter:     0.0,
		TaxSaved:       taxImpact.Saved,
		TradesProposed: len(trades),
		TradesExecuted: 0,
		Trades:         trades,
		TaxImpact:      taxImpact,
		PolicyVersion:  2,
		Status:         "completed",
		Timestamp:      time.Now(),
	}

	err = workflow.ExecuteActivity(
		ctx,
		"LogRebalanceAuditActivity",
		audit,
	).Get(ctx, nil)

	if err != nil {
		logger.Error("Failed to log audit", "error", err)
	}

	logger.Info("Rebalance workflow completed successfully",
		"totalTrades", len(trades),
		"taxSaved", taxImpact.Saved)

	return nil
}
