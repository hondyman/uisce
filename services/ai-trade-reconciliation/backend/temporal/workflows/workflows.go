package workflows

import (
	"encoding/json"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/hondyman/semlayer/services/ai-trade-reconciliation/backend/internal/ai"
	"github.com/hondyman/semlayer/services/ai-trade-reconciliation/backend/internal/models"
	"github.com/hondyman/semlayer/services/ai-trade-reconciliation/backend/temporal/activities"
	"github.com/google/uuid"
)

// AIReconciliationWorkflow orchestrates the daily trade reconciliation process
func AIReconciliationWorkflow(ctx workflow.Context) error {
	// Set up activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Activity 1: Fetch yesterday's trades
	var trades []models.Trade
	err := workflow.ExecuteActivity(ctx, activities.FetchYesterdaysTrades).Get(ctx, &trades)
	if err != nil {
		return err
	}

	// Activity 2: Fetch unprocessed confirmations
	var confirms []models.TradeConfirm
	err = workflow.ExecuteActivity(ctx, activities.FetchTradeConfirms).Get(ctx, &confirms)
	if err != nil {
		return err
	}

	// Activity 3: AI Reconciliation
	var reconResult *ai.ReconcileOutput
	err = workflow.ExecuteActivity(ctx, activities.AIReconcile, trades, confirms).Get(ctx, &reconResult)
	if err != nil {
		return err
	}

	// Activity 4: Save result
	var resultID uuid.UUID
	err = workflow.ExecuteActivity(ctx, activities.SaveReconciliationResult, reconResult, 1).Get(ctx, &resultID)
	if err != nil {
		return err
	}

	// Activity 5 & 6: Process discrepancies in parallel
	for _, discrepancy := range reconResult.Discrepancies {
		if discrepancy.Severity == "high" || discrepancy.Severity == "medium" {
			// Create task
			err = workflow.ExecuteActivity(ctx, activities.CreateReconciliationTask, resultID, discrepancy, discrepancy.Severity).Get(ctx, nil)
			if err != nil {
				// Log but don't fail workflow
				workflow.GetLogger(ctx).Error("Failed to create task", "error", err)
			}

			// Send notification
			err = workflow.ExecuteActivity(ctx, activities.NotifyDiscrepancy, discrepancy).Get(ctx, nil)
			if err != nil {
				workflow.GetLogger(ctx).Error("Failed to notify", "error", err)
			}
		} else {
			// Auto-resolve low-severity
			err = workflow.ExecuteActivity(ctx, activities.AutoResolveDiscrepancy, discrepancy).Get(ctx, nil)
			if err != nil {
				workflow.GetLogger(ctx).Error("Failed to auto-resolve", "error", err)
			}
		}
	}

	// Activity 7: Audit logging
	auditDetails, _ := json.Marshal(map[string]interface{}{
		"match_rate": reconResult.MatchRate,
		"matched":    len(reconResult.Matched),
		"unmatched":  len(reconResult.UnmatchedTrades) + len(reconResult.UnmatchedConfirms),
	})
	err = workflow.ExecuteActivity(ctx, activities.LogReconciliationAudit, resultID, "reconciliation_completed", auditDetails).Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Error("Failed to audit", "error", err)
	}

	return nil
}

// CronSchedule returns the cron schedule for daily 6 AM runs
func CronSchedule() string {
	return "0 6 * * *" // 6 AM every day
}
