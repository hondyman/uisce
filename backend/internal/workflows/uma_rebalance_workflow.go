package workflows

import (
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// UMA REBALANCE WORKFLOW
// Orchestrates the complete UMA rebalancing process with ABAC, tax simulation,
// and execution tracking
// ============================================================================

// UMARebalanceWorkflowState tracks the current state of the workflow
type UMARebalanceWorkflowState struct {
	RequestID        string
	UMAAccountID     string
	CurrentPhase     string
	ABACApproved     bool
	PlanID           string
	ApprovalStatus   string
	ExecutionDetails map[string]interface{}
	Errors           []string
	LastUpdated      time.Time
}

// UMARebalanceWorkflow orchestrates the complete rebalancing process
func UMARebalanceWorkflow(ctx workflow.Context, input models.UMARebalanceWorkflowInput) (map[string]interface{}, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("🎬 Starting UMARebalanceWorkflow", "RequestID", input.RequestID, "UMAAccountID", input.UMAAccountID)

	// Set activity options
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Result tracking
	result := map[string]interface{}{
		"request_id":      input.RequestID,
		"uma_account_id":  input.UMAAccountID,
		"workflow_status": "pending",
		"status":          "pending",
		"completed_at":    nil,
	}

	// ========================================================================
	// PHASE 1: ABAC CHECK
	// ========================================================================
	logger.Info("▶️  Phase 1: ABAC Authorization Check")
	var abacApproved bool
	err := workflow.ExecuteActivity(ctx, (*UMAActivities).ABACCheckActivity, input).Get(ctx, &abacApproved)
	if err != nil {
		logger.Error("❌ ABAC check failed", "Error", err)
		result["workflow_status"] = "failed"
		result["status"] = "failed"
		result["error"] = fmt.Sprintf("ABAC check failed: %v", err)
		return result, fmt.Errorf("ABAC authorization denied: %w", err)
	}

	if !abacApproved {
		logger.Error("❌ ABAC authorization denied")
		result["workflow_status"] = "denied"
		result["status"] = "denied"
		result["error"] = "ABAC authorization denied"
		return result, fmt.Errorf("user not authorized to rebalance UMA %s", input.UMAAccountID)
	}

	logger.Info("✅ ABAC authorization approved")

	// ========================================================================
	// PHASE 2: LOAD UMA DATA
	// ========================================================================
	logger.Info("▶️  Phase 2: Load UMA Account Data")
	var uma *models.UMAAccount
	var sleeves []*models.UMASleeve
	var holdings []*models.UMAHolding

	// Execute LoadUMADataActivity and capture the returned map containing uma, sleeves, holdings
	var loadRes map[string]interface{}
	err = workflow.ExecuteActivity(ctx, (*UMAActivities).LoadUMADataActivity, input.UMAAccountID, input.TenantID).Get(ctx, &loadRes)
	if err != nil {
		logger.Error("❌ Failed to load UMA data", "Error", err)
		result["workflow_status"] = "failed"
		result["status"] = "failed"
		result["error"] = fmt.Sprintf("Failed to load UMA data: %v", err)
		return result, fmt.Errorf("load UMA data failed: %w", err)
	}

	// Extract typed values from the returned map
	if v, ok := loadRes["uma"]; ok {
		switch val := v.(type) {
		case *models.UMAAccount:
			uma = val
		case models.UMAAccount:
			uma = &val
		}
	}
	if v, ok := loadRes["sleeves"]; ok {
		if s, ok := v.([]*models.UMASleeve); ok {
			sleeves = s
		}
	}
	if v, ok := loadRes["holdings"]; ok {
		if h, ok := v.([]*models.UMAHolding); ok {
			holdings = h
		}
	}

	logger.Info("✅ UMA data loaded", "Sleeves", len(sleeves), "Holdings", len(holdings))
	result["uma_account"] = uma
	result["sleeve_count"] = len(sleeves)

	// ========================================================================
	// PHASE 3: EVALUATE RULES
	// ========================================================================
	logger.Info("▶️  Phase 3: Evaluate Business Rules")
	var ruleViolations []map[string]interface{}
	err = workflow.ExecuteActivity(ctx, (*UMAActivities).EvaluateRulesActivity, uma, sleeves, holdings).Get(ctx, &ruleViolations)
	if err != nil {
		logger.Error("❌ Rule evaluation failed", "Error", err)
		result["workflow_status"] = "failed"
		result["status"] = "failed"
		result["error"] = fmt.Sprintf("Rule evaluation failed: %v", err)
		return result, fmt.Errorf("rule evaluation failed: %w", err)
	}

	logger.Info("✅ Rules evaluated", "Violations", len(ruleViolations))
	result["rule_violations"] = ruleViolations

	// ========================================================================
	// PHASE 4: DETECT DRIFT & GENERATE TRADES
	// ========================================================================
	logger.Info("▶️  Phase 4: Detect Drift and Generate Trades")
	var plan *models.UMARebalancePlan
	err = workflow.ExecuteActivity(ctx, (*UMAActivities).GenerateRebalancePlanActivity, input.UMAAccountID, input.TenantID, sleeves, holdings).Get(ctx, &plan)
	if err != nil {
		logger.Error("❌ Failed to generate rebalance plan", "Error", err)
		result["workflow_status"] = "failed"
		result["status"] = "failed"
		result["error"] = fmt.Sprintf("Failed to generate plan: %v", err)
		return result, fmt.Errorf("generate plan failed: %w", err)
	}

	if plan == nil || len(plan.Trades) == 0 {
		logger.Info("ℹ️  No rebalancing needed; account is in balance")
		result["workflow_status"] = "completed"
		result["status"] = "completed"
		result["action"] = "no_rebalancing_needed"
		return result, nil
	}

	logger.Info("✅ Rebalance plan generated", "Trades", len(plan.Trades), "TotalCost", plan.TotalCost)
	result["plan_id"] = plan.ID
	result["trade_count"] = len(plan.Trades)
	result["total_cost"] = plan.TotalCost
	result["total_tax_impact"] = plan.TotalTaxImpact

	// ========================================================================
	// PHASE 5: TAX SIMULATION (if enabled)
	// ========================================================================
	logger.Info("▶️  Phase 5: Tax Harvest Simulation")
	var taxSimulation map[string]interface{}
	err = workflow.ExecuteActivity(ctx, (*UMAActivities).TaxHarvestSimulationActivity, plan, sleeves, holdings).Get(ctx, &taxSimulation)
	if err != nil {
		logger.Warn("⚠️  Tax simulation failed (non-blocking)", "Error", err)
		taxSimulation = map[string]interface{}{"error": err.Error()}
	}

	logger.Info("✅ Tax simulation completed")
	result["tax_simulation"] = taxSimulation

	// ========================================================================
	// PHASE 6: APPROVAL CHECK
	// ========================================================================
	logger.Info("▶️  Phase 6: Approval Check")
	var approvalRequired bool
	err = workflow.ExecuteActivity(ctx, (*UMAActivities).CheckApprovalRequiredActivity, uma, plan).Get(ctx, &approvalRequired)
	if err != nil {
		logger.Error("❌ Approval check failed", "Error", err)
		approvalRequired = true // Default to requiring approval on error
	}

	if approvalRequired {
		logger.Info("⏳ Plan requires approval; waiting for signal...")
		result["approval_status"] = "pending"

		// Create approval signal channel
		approvalCh := workflow.GetSignalChannel(ctx, "uma_rebalance_approval")

		var approval map[string]interface{}
		approvalCh.Receive(ctx, &approval)

		// Check approval status
		if approved, ok := approval["approved"].(bool); !ok || !approved {
			logger.Error("❌ Plan rejected by approver")
			result["workflow_status"] = "rejected"
			result["status"] = "rejected"
			result["approval_status"] = "rejected"
			result["rejection_reason"] = approval["reason"]
			return result, fmt.Errorf("plan rejected")
		}

		logger.Info("✅ Plan approved")
		result["approval_status"] = "approved"
	} else {
		logger.Info("✅ Plan auto-approved (no approval required)")
		result["approval_status"] = "auto_approved"
	}

	// ========================================================================
	// PHASE 7: EXECUTION
	// ========================================================================
	logger.Info("▶️  Phase 7: Execute Trades")
	result["execution_status"] = "executing"

	var executionResult map[string]interface{}
	err = workflow.ExecuteActivity(ctx, (*UMAActivities).ExecuteTradesActivity, plan).Get(ctx, &executionResult)
	if err != nil {
		logger.Error("❌ Trade execution failed", "Error", err)
		result["workflow_status"] = "failed"
		result["status"] = "failed"
		result["execution_status"] = "failed"
		result["error"] = fmt.Sprintf("Trade execution failed: %v", err)
		return result, fmt.Errorf("trade execution failed: %w", err)
	}

	logger.Info("✅ Trades executed")
	result["execution_details"] = executionResult
	result["execution_status"] = "completed"

	// ========================================================================
	// PHASE 8: UPDATE HASURA
	// ========================================================================
	logger.Info("▶️  Phase 8: Update Hasura")
	err = workflow.ExecuteActivity(ctx, (*UMAActivities).UpdateHasuraActivity, input.TenantID, plan, executionResult).Get(ctx, nil)
	if err != nil {
		logger.Warn("⚠️  Hasura update failed (non-blocking)", "Error", err)
	}

	logger.Info("✅ Hasura updated")

	// ========================================================================
	// PHASE 9: EMIT COMPLETION EVENT
	// ========================================================================
	logger.Info("▶️  Phase 9: Emit Completion Event")
	err = workflow.ExecuteActivity(ctx, (*UMAActivities).EmitRebalanceCompletedEventActivity, input, plan, executionResult).Get(ctx, nil)
	if err != nil {
		logger.Warn("⚠️  Event emission failed (non-blocking)", "Error", err)
	}

	logger.Info("✅ Completion event emitted")

	// ========================================================================
	// FINAL RESULT
	// ========================================================================
	result["workflow_status"] = "completed"
	result["status"] = "completed"
	result["completed_at"] = time.Now()
	logger.Info("✅ UMARebalanceWorkflow completed successfully")

	return result, nil
}

// ============================================================================
// SIGNAL HANDLERS
// ============================================================================

// UMARebalanceApprovalSignal handles approval/rejection signals
type UMARebalanceApprovalSignal struct {
	Approved   string // "approved" or "rejected"
	ApprovedBy string
	Reason     string
	Timestamp  time.Time
}

// ============================================================================
// QUERY HANDLERS (for monitoring/observability)
// ============================================================================

// QueryUMARebalanceStatus returns the current status of a rebalance workflow
func QueryUMARebalanceStatus(ctx workflow.Context) (UMARebalanceWorkflowState, error) {
	// This would be implemented using Temporal's query capabilities
	// For now, return placeholder
	return UMARebalanceWorkflowState{
		CurrentPhase: "executing",
	}, nil
}
