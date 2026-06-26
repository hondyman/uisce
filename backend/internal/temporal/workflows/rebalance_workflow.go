package workflows

import (
	"time"

	"github.com/hondyman/semlayer/backend/internal/optimizer"
	"github.com/hondyman/semlayer/backend/internal/temporal/activities"
	"go.temporal.io/sdk/workflow"
)

// RebalanceInput contains the workflow input parameters.
type RebalanceInput struct {
	TenantID    string  `json:"tenant_id"`
	PortfolioID string  `json:"portfolio_id"`
	AccountID   string  `json:"account_id"`
	AdvisorID   string  `json:"advisor_id"`
	Threshold   float64 `json:"threshold"`
	NumRuns     int     `json:"num_runs,omitempty"`
}

// RebalanceOutput contains the workflow result.
type RebalanceOutput struct {
	ProposalID string                             `json:"proposal_id"`
	Decision   string                             `json:"decision"`
	SagaResult *activities.ExecuteTradeSagaOutput `json:"saga_result,omitempty"`
	UARID      string                             `json:"uar_id"`
}

// RebalanceWorkflow orchestrates the tax-aware rebalancing flow.
func RebalanceWorkflow(ctx workflow.Context, input RebalanceInput) (*RebalanceOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting RebalanceWorkflow", "TenantID", input.TenantID, "PortfolioID", input.PortfolioID)

	// Activity options with appropriate timeouts
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Default values
	if input.NumRuns == 0 {
		input.NumRuns = 1000
	}
	if input.Threshold == 0 {
		input.Threshold = 0.03 // 3% drift threshold
	}

	var acts *activities.RebalanceActivities

	// Step 1: Check drift
	checkDriftInput := activities.CheckDriftInput{
		PortfolioID: input.PortfolioID,
		TenantID:    input.TenantID,
		Threshold:   input.Threshold,
	}
	var checkDriftOutput activities.CheckDriftOutput
	err := workflow.ExecuteActivity(ctx, acts.CheckDriftActivity, checkDriftInput).Get(ctx, &checkDriftOutput)
	if err != nil {
		return nil, err
	}

	if !checkDriftOutput.DriftDetected {
		logger.Info("No significant drift detected, workflow complete")
		return &RebalanceOutput{Decision: "no_drift"}, nil
	}

	// Step 2: Load lots
	loadLotsInput := activities.LoadLotsInput{
		PortfolioID: input.PortfolioID,
		TenantID:    input.TenantID,
	}
	var loadLotsOutput activities.LoadLotsOutput
	err = workflow.ExecuteActivity(ctx, acts.LoadLotsActivity, loadLotsInput).Get(ctx, &loadLotsOutput)
	if err != nil {
		return nil, err
	}

	// Step 3: Load prices
	securityIDs := make([]string, 0)
	for _, lot := range loadLotsOutput.Lots {
		securityIDs = append(securityIDs, lot.Symbol)
	}
	loadPricesInput := activities.LoadPricesInput{SecurityIDs: securityIDs}
	var loadPricesOutput activities.LoadPricesOutput
	err = workflow.ExecuteActivity(ctx, acts.LoadPricesActivity, loadPricesInput).Get(ctx, &loadPricesOutput)
	if err != nil {
		return nil, err
	}

	// Step 4: Load tax rules
	loadTaxRulesInput := activities.LoadTaxRulesInput{
		AccountID: input.AccountID,
		TenantID:  input.TenantID,
	}
	var loadTaxRulesOutput activities.LoadTaxRulesOutput
	err = workflow.ExecuteActivity(ctx, acts.LoadTaxRulesActivity, loadTaxRulesInput).Get(ctx, &loadTaxRulesOutput)
	if err != nil {
		return nil, err
	}

	// Step 5: Tax-aware optimize
	taxOptInput := activities.TaxAwareOptimizeInput{
		DriftReport: checkDriftOutput.DriftReport,
		Lots:        loadLotsOutput.Lots,
		Prices:      loadPricesOutput.Prices,
		TaxRules:    loadTaxRulesOutput.TaxRules,
	}
	var taxOptOutput activities.TaxAwareOptimizeOutput
	err = workflow.ExecuteActivity(ctx, acts.TaxAwareOptimizeActivity, taxOptInput).Get(ctx, &taxOptOutput)
	if err != nil {
		return nil, err
	}

	// Step 6: Monte Carlo simulation
	mcInput := activities.MonteCarloSimInput{
		Plan:     taxOptOutput.Plan,
		Lots:     loadLotsOutput.Lots,
		Prices:   loadPricesOutput.Prices,
		TaxRules: loadTaxRulesOutput.TaxRules,
		NumRuns:  input.NumRuns,
	}
	var mcOutput activities.MonteCarloSimOutput
	err = workflow.ExecuteActivity(ctx, acts.MonteCarloSimActivity, mcInput).Get(ctx, &mcOutput)
	if err != nil {
		return nil, err
	}

	// Update plan with Monte Carlo results
	plan := taxOptOutput.Plan
	plan.MonteCarlo = mcOutput.Summary

	// Step 7: Policy check
	policyInput := activities.PolicyCheckInput{
		Plan:        plan,
		PortfolioID: input.PortfolioID,
		TenantID:    input.TenantID,
	}
	var policyOutput activities.PolicyCheckOutput
	err = workflow.ExecuteActivity(ctx, acts.PolicyCheckActivity, policyInput).Get(ctx, &policyOutput)
	if err != nil {
		return nil, err
	}

	if !policyOutput.Passed {
		logger.Info("Policy check failed", "violations", len(policyOutput.Violations))
		return &RebalanceOutput{
			ProposalID: plan.ID,
			Decision:   "policy_blocked",
		}, nil
	}

	// Step 8: Notify advisor
	notifyInput := activities.NotifyAdvisorInput{
		ProposalID:  plan.ID,
		PortfolioID: input.PortfolioID,
		AdvisorID:   input.AdvisorID,
		Plan:        plan,
		MonteCarlo:  mcOutput.Summary,
		Violations:  policyOutput.Violations,
	}
	var notifyOutput activities.NotifyAdvisorOutput
	err = workflow.ExecuteActivity(ctx, acts.NotifyAdvisorActivity, notifyInput).Get(ctx, &notifyOutput)
	if err != nil {
		return nil, err
	}

	// Step 9: Wait for advisor approval signal
	var decision optimizer.AdvisorDecision
	signalChan := workflow.GetSignalChannel(ctx, "AdvisorApproval")

	selector := workflow.NewSelector(ctx)
	selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &decision)
	})

	// Wait up to 7 days for advisor decision
	timerFuture := workflow.NewTimer(ctx, 7*24*time.Hour)
	selector.AddFuture(timerFuture, func(f workflow.Future) {
		decision = optimizer.AdvisorDecision{
			Action:    "timeout",
			Rationale: "No response within 7 days",
			Timestamp: workflow.Now(ctx),
		}
	})

	selector.Select(ctx)
	logger.Info("Received advisor decision", "action", decision.Action)

	// Step 10: Execute trades if approved
	var sagaResult *activities.ExecuteTradeSagaOutput
	if decision.Action == "approve" {
		sagaInput := activities.ExecuteTradeSagaInput{
			ProposalID:  plan.ID,
			PortfolioID: input.PortfolioID,
			Plan:        plan,
			ApprovedBy:  decision.AdvisorID,
			ApprovedAt:  decision.Timestamp,
		}
		var sagaOutput activities.ExecuteTradeSagaOutput
		err = workflow.ExecuteActivity(ctx, acts.ExecuteTradeSagaActivity, sagaInput).Get(ctx, &sagaOutput)
		if err != nil {
			return nil, err
		}
		sagaResult = &sagaOutput
		logger.Info("Trade saga completed", "status", sagaOutput.Status)
	}

	// Step 11: Persist UAR
	uarInput := activities.PersistUARInput{
		ProposalID:  plan.ID,
		PortfolioID: input.PortfolioID,
		TenantID:    input.TenantID,
		Plan:        plan,
		MonteCarlo:  mcOutput.Summary,
		Violations:  policyOutput.Violations,
		Decision:    decision.Action,
		DecisionBy:  decision.AdvisorID,
		DecisionAt:  decision.Timestamp,
	}
	if sagaResult != nil {
		uarInput.Executions = sagaResult.Executions
	}

	var uarOutput activities.PersistUAROutput
	err = workflow.ExecuteActivity(ctx, acts.PersistUARActivity, uarInput).Get(ctx, &uarOutput)
	if err != nil {
		logger.Error("Failed to persist UAR", "error", err)
		// Don't fail workflow for UAR persistence failure
	}

	return &RebalanceOutput{
		ProposalID: plan.ID,
		Decision:   decision.Action,
		SagaResult: sagaResult,
		UARID:      uarOutput.UARID,
	}, nil
}
