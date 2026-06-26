package workflow

import (
	"time"

	"go.temporal.io/sdk/workflow"
	"github.com/hondyman/semlayer/backend/internal/rebalancer/engine"
)

// Signal types
const (
	SignalDeposit     = "SignalDeposit"
	SignalWithdrawal  = "SignalWithdrawal"
	SignalDrift       = "SignalDrift"
	SignalUpdateModel = "SignalUpdateModel"
	SignalAdvisorDecision = "SignalAdvisorDecision"
)

// Query types
const (
	QueryGetState = "QueryGetState"
)

// WorkflowState represents the current state of the portfolio workflow
type WorkflowState struct {
	Status        string    `json:"status"`
	LastRebalance time.Time `json:"last_rebalance"`
	CashBalance   float64   `json:"cash_balance"`
	Drift         float64   `json:"drift"`
	PendingPlan   *engine.Plan `json:"pending_plan,omitempty"`
}

// PortfolioLifecycleWorkflow manages the infinite lifecycle of an investment account
func PortfolioLifecycleWorkflow(ctx workflow.Context, portfolioID string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting PortfolioLifecycleWorkflow", "PortfolioID", portfolioID)

	state := WorkflowState{
		Status: "MONITORING",
	}

	err := workflow.SetQueryHandler(ctx, QueryGetState, func() (WorkflowState, error) {
		return state, nil
	})
	if err != nil {
		return err
	}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 10,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	selector := workflow.NewSelector(ctx)

	depositChan := workflow.GetSignalChannel(ctx, SignalDeposit)
	driftChan := workflow.GetSignalChannel(ctx, SignalDrift)
	decisionChan := workflow.GetSignalChannel(ctx, SignalAdvisorDecision)

	// Handle Deposit
	selector.AddReceive(depositChan, func(c workflow.ReceiveChannel, more bool) {
		var amount float64
		c.Receive(ctx, &amount)
		logger.Info("Received deposit signal", "Amount", amount)
		state.CashBalance += amount
		state.Status = "REBALANCING_DUE_TO_DEPOSIT"
		runRebalanceFlow(ctx, portfolioID, "DEPOSIT", &state)
		state.Status = "MONITORING"
	})

	// Handle Drift
	selector.AddReceive(driftChan, func(c workflow.ReceiveChannel, more bool) {
		var drift float64
		c.Receive(ctx, &drift)
		logger.Info("Received drift signal", "Drift", drift)
		state.Drift = drift
		if drift > 0.05 {
			state.Status = "REBALANCING_DUE_TO_DRIFT"
			runRebalanceFlow(ctx, portfolioID, "DRIFT", &state)
			state.Status = "MONITORING"
		}
	})
	
	// Handle Advisor Decision (if we are waiting for one)
	// Note: In a real infinite loop, we might handle this differently, 
	// but here we just register the channel for completeness.
	selector.AddReceive(decisionChan, func(c workflow.ReceiveChannel, more bool) {
		var decision string
		c.Receive(ctx, &decision)
		logger.Info("Received advisor decision", "Decision", decision)
		// Process decision...
	})

	// Heartbeat
	timerFuture := workflow.NewTimer(ctx, 15*time.Minute)
	selector.AddFuture(timerFuture, func(f workflow.Future) {
		// Heartbeat logic
	})

	for {
		selector.Select(ctx)
		
		info := workflow.GetInfo(ctx)
		if info.GetCurrentHistoryLength() > 10000 {
			return workflow.NewContinueAsNewError(ctx, PortfolioLifecycleWorkflow, portfolioID)
		}
	}
}

func runRebalanceFlow(ctx workflow.Context, portfolioID string, reason string, state *WorkflowState) {
	logger := workflow.GetLogger(ctx)
	
	// 1. Optimize & Generate Plan
	var plan engine.Plan
	err := workflow.ExecuteActivity(ctx, "TaxAwareOptimizeActivity", "tenant_default", portfolioID).Get(ctx, &plan)
	if err != nil {
		logger.Error("Optimization failed", "Error", err)
		return
	}

	// 2. Run Monte Carlo Simulation
	var planWithMC engine.Plan
	err = workflow.ExecuteActivity(ctx, "MonteCarloSimActivity", plan, 1000).Get(ctx, &planWithMC)
	if err != nil {
		logger.Error("Monte Carlo failed", "Error", err)
		return
	}
	
	state.PendingPlan = &planWithMC

	// 3. Notify Advisor (Push to GenUI)
	err = workflow.ExecuteActivity(ctx, "NotifyAdvisorActivity", "tenant_default", planWithMC).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to notify advisor", "Error", err)
		return
	}
	
	// 4. Wait for Signal (Simplified: In reality, we'd block here or use a dedicated state)
	// For this loop, we assume the signal handler above picks it up or we use a nested selector.
	// To keep it simple for this file, we just log that we are waiting.
	logger.Info("Waiting for Advisor Approval...")
}
