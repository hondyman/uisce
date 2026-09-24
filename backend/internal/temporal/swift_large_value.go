package temporal

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

type SWIFTLargeValueInput struct {
	TenantID         uuid.UUID `json:"tenant_id"`
	CustodianID      uuid.UUID `json:"custodian_id"`
	TransactionRef   string    `json:"transaction_ref"`
	UETR             string    `json:"uetr,omitempty"`
	SettlementAmount string    `json:"settlement_amount"`
	Currency         string    `json:"currency"`
	SLAHours         int       `json:"sla_hours"`
}

type LargeValueApprovalSignal struct {
	Decision   string `json:"decision"`
	ApproverID string `json:"approver_id"`
	Notes      string `json:"notes,omitempty"`
}

func SWIFTLargeValueApprovalWorkflow(ctx workflow.Context, input SWIFTLargeValueInput) (string, error) {
	logger := workflow.GetLogger(ctx)
	
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	}
	actx := workflow.WithActivityOptions(ctx, ao)
	
	err := workflow.ExecuteActivity(actx, CreateLargeValueApprovalTaskActivity, input).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to create large value approval task", "error", err)
		return "", err
	}
	
	slaHours := input.SLAHours
	if slaHours <= 0 {
		slaHours = 4
	}
	
	approvalSig := workflow.GetSignalChannel(ctx, "LargeValueApproval")
	slaTimer := workflow.NewTimer(ctx, time.Duration(slaHours) * time.Hour)
	
	selector := workflow.NewSelector(ctx)
	
	var decision string
	var approverID string
	var notes string
	var slaBreached bool
	
	selector.AddReceive(approvalSig, func(c workflow.ReceiveChannel, more bool) {
		var sig LargeValueApprovalSignal
		c.Receive(ctx, &sig)
		decision = sig.Decision
		approverID = sig.ApproverID
		notes = sig.Notes
	})
	
	selector.AddFuture(slaTimer, func(f workflow.Future) {
		slaBreached = true
	})
	
	selector.Select(ctx)
	
	if slaBreached {
		logger.Info("Large value approval SLA breached")
		_ = workflow.ExecuteActivity(actx, EscalateSLABreachActivity, input).Get(ctx, nil)
		return "escalated", nil
	}
	
	err = workflow.ExecuteActivity(actx, RecordLargeValueDecisionActivity, input.TransactionRef, decision, approverID, notes).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to record decision", "error", err)
	}
	
	return decision, nil
}

func CreateLargeValueApprovalTaskActivity(ctx context.Context, input SWIFTLargeValueInput) error {
	activity.GetLogger(ctx).Info("CreateLargeValueApprovalTaskActivity", "ref", input.TransactionRef, "amount", input.SettlementAmount)
	return nil
}

func RecordLargeValueDecisionActivity(ctx context.Context, ref string, decision string, approverID string, notes string) error {
	activity.GetLogger(ctx).Info("RecordLargeValueDecisionActivity", "ref", ref, "decision", decision)
	// GSIFI: WHERE (tenant_id = $1 OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
	return nil
}

func EscalateSLABreachActivity(ctx context.Context, input SWIFTLargeValueInput) error {
	activity.GetLogger(ctx).Info("EscalateSLABreachActivity", "ref", input.TransactionRef)
	return nil
}
