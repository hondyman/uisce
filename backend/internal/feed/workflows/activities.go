package workflows

import (
	"context"

	"github.com/hondyman/semlayer/backend/internal/feed/approvals"
	feedaudit "github.com/hondyman/semlayer/backend/internal/feed/audit"
)

// CreateApprovalInput defines the input for creating an approval
type CreateApprovalInput struct {
	TenantID      string
	ClientID      string
	ActionType    string
	ActionDetails map[string]interface{}
	WorkflowID    string
	RunID         string
}

var approvalService *approvals.Service
var auditService feedaudit.AuditRecorder

// SetApprovalService sets the approval service for activities
func SetApprovalService(svc *approvals.Service) {
	approvalService = svc
}

// SetAuditService sets the audit service for activities
func SetAuditService(svc feedaudit.AuditRecorder) {
	auditService = svc
}

// ValidateTrade validates the trade request
func ValidateTrade(ctx context.Context, input TradeWorkflowInput) (bool, error) {
	// Record audit event
	if auditService != nil {
		_ = auditService.Record(
			input.ClientID+"-"+input.ActionType,
			"trade_validated",
			"system",
			"validate_trade",
			input.ClientID,
			map[string]interface{}{
				"client_id":   input.ClientID,
				"action_type": input.ActionType,
				"tenant_id":   input.TenantID,
			},
		)
	}

	// Placeholder validation logic
	return true, nil
}

// CreateApproval creates an approval request
func CreateApproval(ctx context.Context, input CreateApprovalInput) (string, error) {
	if approvalService == nil {
		return "", nil
	}

	req, err := approvalService.CreateApprovalRequest(
		input.TenantID,
		input.ClientID,
		input.ActionType,
		input.ActionDetails,
		input.WorkflowID,
		input.RunID,
	)
	if err != nil {
		return "", err
	}

	// Record audit event
	if auditService != nil {
		_ = auditService.Record(
			req.WorkflowID,
			"approval_created",
			"system",
			"create_approval",
			input.ClientID,
			map[string]interface{}{
				"approval_id": req.ID,
				"client_id":   input.ClientID,
				"action_type": input.ActionType,
				"workflow_id": input.WorkflowID,
			},
		)
	}

	return req.ID, nil
}

// ExecuteTrade executes the trade
func ExecuteTrade(ctx context.Context, input TradeWorkflowInput) (bool, error) {
	// Record audit event
	if auditService != nil {
		_ = auditService.Record(
			input.ClientID+"-"+input.ActionType,
			"trade_executed",
			"system",
			"execute_trade",
			input.ClientID,
			map[string]interface{}{
				"client_id":      input.ClientID,
				"action_type":    input.ActionType,
				"action_details": input.ActionDetails,
			},
		)
	}

	// Placeholder execution logic
	return true, nil
}

// NotifyTradeComplete sends notification about trade completion
func NotifyTradeComplete(ctx context.Context, input TradeWorkflowInput) error {
	// Placeholder notification logic
	return nil
}
