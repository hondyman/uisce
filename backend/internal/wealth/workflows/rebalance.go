package workflows

import (
	"time"

	"github.com/hondyman/semlayer/backend/pkg/optimizer"
	"go.temporal.io/sdk/workflow"
)

// RebalanceWorkflowInput defines the input for the rebalancing workflow
type RebalanceWorkflowInput struct {
	PortfolioID string
	TenantID    string
}

// RebalanceWorkflowResult defines the output of the rebalancing workflow
type RebalanceWorkflowResult struct {
	PlanID     string
	Status     string
	TradesExecuted int
}

// RebalanceWorkflow orchestrates the autonomous rebalancing process
func RebalanceWorkflow(ctx workflow.Context, input RebalanceWorkflowInput) (*RebalanceWorkflowResult, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	logger := workflow.GetLogger(ctx)
	logger.Info("Starting RebalanceWorkflow", "PortfolioID", input.PortfolioID)

	// Step 1: Fetch Portfolio Data & Market Prices
	var inputs optimizer.Inputs
	err := workflow.ExecuteActivity(ctx, "FetchRebalanceInputsActivity", input.PortfolioID, input.TenantID).Get(ctx, &inputs)
	if err != nil {
		return nil, err
	}

	// Step 2: Run Optimizer
	var plan optimizer.Plan
	err = workflow.ExecuteActivity(ctx, "RunOptimizerActivity", inputs).Get(ctx, &plan)
	if err != nil {
		return nil, err
	}

	// Step 3: Check Autonomy Thresholds
	var isAutonomous bool
	err = workflow.ExecuteActivity(ctx, "CheckAutonomyActivity", plan, inputs).Get(ctx, &isAutonomous)
	if err != nil {
		return nil, err
	}

	if !isAutonomous {
		// Step 4a: Request Approval
		logger.Info("Plan requires approval", "PlanID", plan.ID)
		
		// Signal channel for approval
		approvalChan := workflow.GetSignalChannel(ctx, "AdvisorApproval")
		
		// Wait for signal or timeout (e.g., 24 hours)
		var signalVal map[string]interface{}
		selector := workflow.NewSelector(ctx)
		selector.AddReceive(approvalChan, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, &signalVal)
		})
		
		// Wait for 24 hours
		timerFuture := workflow.NewTimer(ctx, 24*time.Hour)
		selector.AddFuture(timerFuture, func(f workflow.Future) {
			// Timeout occurred
		})

		selector.Select(ctx)

		if signalVal == nil {
			logger.Info("Approval timed out", "PlanID", plan.ID)
			return &RebalanceWorkflowResult{PlanID: plan.ID, Status: "TIMED_OUT"}, nil
		}

		approved, ok := signalVal["approved"].(bool)
		if !ok || !approved {
			logger.Info("Plan rejected", "PlanID", plan.ID)
			return &RebalanceWorkflowResult{PlanID: plan.ID, Status: "REJECTED"}, nil
		}
	}

	// Step 5: Execute Trades (Saga)
	var execResult string
	err = workflow.ExecuteActivity(ctx, "ExecuteTradesActivity", plan, input.TenantID).Get(ctx, &execResult)
	if err != nil {
		return nil, err
	}

	return &RebalanceWorkflowResult{
		PlanID:         plan.ID,
		Status:         "COMPLETED",
		TradesExecuted: len(plan.Trades),
	}, nil
}
