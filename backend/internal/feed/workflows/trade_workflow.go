package workflows

import (
	"time"

	"go.temporal.io/sdk/workflow"

	"github.com/hondyman/semlayer/backend/internal/feed/approvals"
)

// TradeWorkflowInput defines the input for the trade workflow
type TradeWorkflowInput struct {
	TenantID      string
	ClientID      string
	ActionType    string
	ActionDetails map[string]interface{}
}

// TradeWorkflowResult contains the workflow output
type TradeWorkflowResult struct {
	Success       bool
	ApprovalID    string
	ExecutedAt    time.Time
	ErrorMessage  string
}

// TradeWorkflow executes a trade with approval gate
func TradeWorkflow(ctx workflow.Context, input TradeWorkflowInput) (*TradeWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("TradeWorkflow started", "tenantID", input.TenantID, "clientID", input.ClientID, "actionType", input.ActionType)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// 1. Validate the trade request
	var validationResult bool
	err := workflow.ExecuteActivity(ctx, ValidateTrade, input).Get(ctx, &validationResult)
	if err != nil {
		return &TradeWorkflowResult{Success: false, ErrorMessage: err.Error()}, err
	}
	if !validationResult {
		return &TradeWorkflowResult{Success: false, ErrorMessage: "trade validation failed"}, nil
	}

	// 2. Create approval request
	var approvalID string
	workflowInfo := workflow.GetInfo(ctx)
	createApprovalInput := CreateApprovalInput{
		TenantID:      input.TenantID,
		ClientID:      input.ClientID,
		ActionType:    input.ActionType,
		ActionDetails: input.ActionDetails,
		WorkflowID:    workflowInfo.WorkflowExecution.ID,
		RunID:         workflowInfo.WorkflowExecution.RunID,
	}
	err = workflow.ExecuteActivity(ctx, CreateApproval, createApprovalInput).Get(ctx, &approvalID)
	if err != nil {
		return &TradeWorkflowResult{Success: false, ErrorMessage: err.Error()}, err
	}

	// 3. Wait for approval signal (with timeout)
	var decision approvals.ApprovalDecision
	signalChan := workflow.GetSignalChannel(ctx, "approval_decision")
	
	selector := workflow.NewSelector(ctx)
	selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &decision)
	})

	// Wait up to 24 hours for approval
	selector.AddFuture(workflow.NewTimer(ctx, 24*time.Hour), func(f workflow.Future) {
		logger.Warn("Approval timeout reached")
		decision = approvals.ApprovalDecision{Approved: false, Comment: "Timeout - no approval received"}
	})

	selector.Select(ctx)

	// 4. Process approval decision
	if !decision.Approved {
		logger.Info("Trade rejected", "approvalID", approvalID, "comment", decision.Comment)
		return &TradeWorkflowResult{
			Success:    false,
			ApprovalID: approvalID,
			ErrorMessage: "trade rejected: " + decision.Comment,
		}, nil
	}

	// 5. Execute the trade
	var executionSuccess bool
	err = workflow.ExecuteActivity(ctx, ExecuteTrade, input).Get(ctx, &executionSuccess)
	if err != nil {
		return &TradeWorkflowResult{
			Success:      false,
			ApprovalID:   approvalID,
			ErrorMessage: err.Error(),
		}, err
	}

	// 6. Send notification
	_ = workflow.ExecuteActivity(ctx, NotifyTradeComplete, input).Get(ctx, nil)

	return &TradeWorkflowResult{
		Success:    executionSuccess,
		ApprovalID: approvalID,
		ExecutedAt: workflow.Now(ctx),
	}, nil
}
