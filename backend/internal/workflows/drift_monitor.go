package workflows

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// DriftMonitorInput is the input for the drift monitor agent
type DriftMonitorInput struct {
	TenantID    string  `json:"tenant_id"`
	PortfolioID string  `json:"portfolio_id"`
	ThresholdBP int     `json:"threshold_bp"` // Basis points (e.g., 50 = 0.5%)
	AutoRebalance bool  `json:"auto_rebalance"`
}

// DriftMonitorResult is the output of the agent
type DriftMonitorResult struct {
	DriftDetected bool     `json:"drift_detected"`
	DriftBP       int      `json:"drift_bp"`
	ActionTaken   string   `json:"action_taken"` // NONE, ALERT, REBALANCE
	RebalanceID   string   `json:"rebalance_id,omitempty"`
}

// DriftMonitorWorkflow implements the portfolio drift monitoring agent
func DriftMonitorWorkflow(ctx workflow.Context, input DriftMonitorInput) (*DriftMonitorResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting DriftMonitor agent", "portfolio", input.PortfolioID)

	// 1. Log Start
	if err := RecordComplianceEvent(ctx, "INPUT", "Drift Monitor Started", input); err != nil {
		return nil, err
	}

	// 2. Get Portfolio Data (Activity)
	var portfolioData map[string]any
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 1 * time.Minute,
	})
	if err := workflow.ExecuteActivity(ctx, "GetPortfolioData", input.PortfolioID).Get(ctx, &portfolioData); err != nil {
		return nil, err
	}

	// 3. Calculate Drift (Activity)
	var driftBP int
	if err := workflow.ExecuteActivity(ctx, "CalculateDrift", portfolioData).Get(ctx, &driftBP); err != nil {
		return nil, err
	}

	// 4. Evaluate Threshold
	result := &DriftMonitorResult{
		DriftDetected: driftBP > input.ThresholdBP,
		DriftBP:       driftBP,
		ActionTaken:   "NONE",
	}

	if result.DriftDetected {
		// Log decision
		RecordComplianceEvent(ctx, "DECISION", "Drift Threshold Exceeded", map[string]any{
			"drift_bp":     driftBP,
			"threshold_bp": input.ThresholdBP,
		})

		if input.AutoRebalance {
			// Trigger Rebalance
			result.ActionTaken = "REBALANCE"
			// In a real app, this would call a child workflow or activity
			result.RebalanceID = "reb_" + workflow.GetInfo(ctx).WorkflowExecution.RunID
			
			RecordComplianceEvent(ctx, "ACTION", "Auto-Rebalance Triggered", map[string]any{
				"rebalance_id": result.RebalanceID,
			})
		} else {
			// Send Alert
			result.ActionTaken = "ALERT"
			workflow.ExecuteActivity(ctx, "SendAlert", input.PortfolioID, "Drift Detected")
			
			RecordComplianceEvent(ctx, "ACTION", "Advisor Alert Sent", nil)
		}
	}

	// 5. Log Completion
	RecordComplianceEvent(ctx, "OUTPUT", "Drift Monitor Completed", result)

	return result, nil
}
