package workflows

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// RebalanceInput defines the input for the rebalance workflow.
type RebalanceInput struct {
	TenantID    string
	PortfolioID string
}

// ApprovalSignal is the signal that the advisor sends to approve/reject the proposal.
type ApprovalSignal struct {
	Approved  bool
	AdvisorID string
	Rationale string
	Time      time.Time
}

// RebalanceWorkflow orchestrates the complete autonomous rebalancing process:
//  1. Check drift
//  2. Generate AI proposal
//  3. Check policy
//  4. Wait for advisor approval
//  5. Execute trades (saga pattern)
//  6. Persist UAR
func RebalanceWorkflow(ctx workflow.Context, input RebalanceInput) (map[string]any, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("🚀 Starting RebalanceWorkflow", "TenantID", input.TenantID, "PortfolioID", input.PortfolioID)

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	result := map[string]any{
		"tenant_id":    input.TenantID,
		"portfolio_id": input.PortfolioID,
		"status":       "started",
	}

	// -------------------------------------------------
	// 1️⃣ Check Drift
	logger.Info("▶️  Step 1: Checking drift")
	var driftReport map[string]any
	err := workflow.ExecuteActivity(ctx, "CheckDriftActivity", input.TenantID, input.PortfolioID).Get(ctx, &driftReport)
	if err != nil {
		logger.Error("❌ Drift check failed", "Error", err)
		result["status"] = "failed"
		result["error"] = err.Error()
		return result, err
	}

	hasDrift, _ := driftReport["has_drift"].(bool)
	if !hasDrift {
		logger.Info("ℹ️  No drift detected; nothing to do")
		result["status"] = "no_drift"
		return result, nil
	}

	logger.Info("✅ Drift detected", "DriftPct", driftReport["drift_pct"])
	result["drift_report"] = driftReport

	// -------------------------------------------------
	// 2️⃣ Generate AI Proposal
	logger.Info("▶️  Step 2: Generating AI proposal")
	var proposal map[string]any
	err = workflow.ExecuteActivity(ctx, "GenerateAIProposalActivity", input.TenantID, input.PortfolioID, driftReport).Get(ctx, &proposal)
	if err != nil {
		logger.Error("❌ AI proposal generation failed", "Error", err)
		result["status"] = "failed"
		result["error"] = err.Error()
		return result, err
	}

	logger.Info("✅ AI proposal generated", "ProposalID", proposal["id"])
	result["proposal"] = proposal

	// -------------------------------------------------
	// 3️⃣ Policy Check
	logger.Info("▶️  Step 3: Checking policy compliance")
	var policyResult map[string]any
	err = workflow.ExecuteActivity(ctx, "PolicyCheckActivity", input.TenantID, proposal).Get(ctx, &policyResult)
	if err != nil {
		logger.Error("❌ Policy check failed", "Error", err)
		result["status"] = "failed"
		result["error"] = err.Error()
		return result, err
	}

	policyOK, _ := policyResult["ok"].(bool)
	if !policyOK {
		logger.Warn("⚠️  Policy rejected the proposal", "Reasons", policyResult["reasons"])
		// Notify advisor but mark as blocked
		_ = workflow.ExecuteActivity(ctx, "NotifyAdvisorActivity", input.TenantID, proposal, "policy_blocked").Get(ctx, nil)
		result["status"] = "policy_blocked"
		result["policy_result"] = policyResult
		return result, nil
	}

	logger.Info("✅ Policy check passed")
	result["policy_result"] = policyResult

	// -------------------------------------------------
	// 4️⃣ Notify Advisor & Wait for Approval
	logger.Info("▶️  Step 4: Notifying advisor and waiting for approval")
	_ = workflow.ExecuteActivity(ctx, "NotifyAdvisorActivity", input.TenantID, proposal, "awaiting_approval").Get(ctx, nil)

	// Create approval signal channel
	approvalCh := workflow.GetSignalChannel(ctx, "AdvisorApproval")

	// Wait for approval with a 2-minute SLA timer
	approvalSelector := workflow.NewSelector(ctx)
	var approval ApprovalSignal
	var timedOut bool

	approvalSelector.AddReceive(approvalCh, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &approval)
	})

	approvalSelector.AddFuture(workflow.NewTimer(ctx, 2*time.Minute), func(f workflow.Future) {
		timedOut = true
	})

	approvalSelector.Select(ctx)

	if timedOut {
		logger.Warn("⏰ Approval timeout; escalating to supervisor")
		result["status"] = "timeout"
		result["escalation"] = "supervisor_notified"
		return result, nil
	}

	if !approval.Approved {
		logger.Info("❌ Advisor rejected the proposal", "Rationale", approval.Rationale)
		result["status"] = "rejected"
		result["rejection_reason"] = approval.Rationale
		return result, nil
	}

	logger.Info("✅ Advisor approved the proposal", "AdvisorID", approval.AdvisorID)
	result["approval"] = map[string]any{
		"advisor_id": approval.AdvisorID,
		"rationale":  approval.Rationale,
		"time":       approval.Time,
	}

	// -------------------------------------------------
	// 5️⃣ Execute Trades (Saga)
	logger.Info("▶️  Step 5: Executing trades")
	var executionResult map[string]any
	err = workflow.ExecuteActivity(ctx, "ExecuteTradeSagaActivity", input.TenantID, proposal).Get(ctx, &executionResult)
	if err != nil {
		logger.Warn("⚠️  Trade execution failed (saga compensated)", "Error", err)
		result["status"] = "execution_failed"
		result["execution_result"] = executionResult
		return result, err
	}

	logger.Info("✅ Trades executed successfully")
	result["execution_result"] = executionResult

	// -------------------------------------------------
	// 6️⃣ Persist UAR
	logger.Info("▶️  Step 6: Persisting UAR")
	err = workflow.ExecuteActivity(ctx, "PersistUARActivity", input.TenantID, result).Get(ctx, nil)
	if err != nil {
		logger.Warn("⚠️  UAR persistence failed (non-blocking)", "Error", err)
	}

	logger.Info("✅ Workflow completed successfully")
	result["status"] = "completed"
	return result, nil
}
