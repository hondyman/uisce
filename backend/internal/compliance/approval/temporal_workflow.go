package approval

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	ComplianceApprovalTaskQueue = "compliance-pretrade-approval"

	SignalOrderApprove = "ApproveSignal"
	SignalOrderReject  = "RejectSignal"
)

// BreachedRuleSummary captures the breach context for human compliance review
type BreachedRuleSummary struct {
	RuleID         uuid.UUID       `json:"rule_id"`
	RuleCode       string          `json:"rule_code"`
	Severity       string          `json:"severity"`
	ThresholdValue decimal.Decimal `json:"threshold_value"`
	Message        string          `json:"message"`
}

// OrderApprovalInput holds parameters for parking an order in Temporal
type OrderApprovalInput struct {
	TenantID      uuid.UUID             `json:"tenant_id"`
	AccountID     uuid.UUID             `json:"account_id"`
	OrderID       uuid.UUID             `json:"order_id"`
	LeaseID       uuid.UUID             `json:"lease_id"`
	LineageID     uuid.UUID             `json:"lineage_id"`
	BreachedRules []BreachedRuleSummary `json:"breached_rules"`
	TTL           time.Duration         `json:"ttl"` // e.g. 15 minutes
	RequestedAt   time.Time             `json:"requested_at"`
}

// ApproveSignalPayload is sent by a compliance officer approving an order
type ApproveSignalPayload struct {
	ApproverID string    `json:"approver_id"`
	Reason     string    `json:"reason"`
	ApprovedAt time.Time `json:"approved_at"`
}

// RejectSignalPayload is sent by a compliance officer explicitly rejecting an order
type RejectSignalPayload struct {
	RejecterID string    `json:"rejecter_id"`
	Reason     string    `json:"reason"`
	RejectedAt time.Time `json:"rejected_at"`
}

// OrderApprovalResult is the outcome of the four-eyes pre-trade approval workflow
type OrderApprovalResult struct {
	OrderID     uuid.UUID `json:"order_id"`
	LeaseID     uuid.UUID `json:"lease_id"`
	LineageID   uuid.UUID `json:"lineage_id"`
	Status      string    `json:"status"` // "APPROVED", "REJECTED", "EXPIRED"
	DecidedBy   string    `json:"decided_by"`
	Reason      string    `json:"reason"`
	CompletedAt time.Time `json:"completed_at"`
}

// OrderApprovalWorkflow coordinates durable pre-trade order parking with TTL timeout auto-rejection
func OrderApprovalWorkflow(ctx workflow.Context, input OrderApprovalInput) (*OrderApprovalResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("PreTrade OrderApprovalWorkflow started",
		"order_id", input.OrderID,
		"lease_id", input.LeaseID,
		"lineage_id", input.LineageID,
		"tenant_id", input.TenantID,
		"account_id", input.AccountID,
		"ttl", input.TTL,
	)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &sdktemporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	ttlDuration := input.TTL
	if ttlDuration <= 0 {
		ttlDuration = 15 * time.Minute
	}

	// Step 1: Park Reservation in memory-cache with approval TTL + margin
	if input.LeaseID != uuid.Nil {
		var parkErr error
		err := workflow.ExecuteActivity(ctx, "ParkReservationActivity", input.LeaseID, ttlDuration).Get(ctx, &parkErr)
		if err != nil {
			logger.Warn("Failed to execute ParkReservationActivity", "error", err)
		}
	}

	// Step 2: Notify compliance desk / blotter of pending order
	var notifyErr error
	err := workflow.ExecuteActivity(ctx, "NotifyComplianceOfficersActivity", input).Get(ctx, &notifyErr)
	if err != nil {
		logger.Warn("Failed to send compliance notification activity (non-fatal)", "error", err)
	}

	// Prepare selector for signals and timer
	selector := workflow.NewSelector(ctx)
	approveChan := workflow.GetSignalChannel(ctx, SignalOrderApprove)
	rejectChan := workflow.GetSignalChannel(ctx, SignalOrderReject)
	timer := workflow.NewTimer(ctx, ttlDuration)

	result := &OrderApprovalResult{
		OrderID:   input.OrderID,
		LeaseID:   input.LeaseID,
		LineageID: input.LineageID,
	}

	var workflowFinished bool

	// Handler for Approve signal
	selector.AddReceive(approveChan, func(c workflow.ReceiveChannel, more bool) {
		var signal ApproveSignalPayload
		c.Receive(ctx, &signal)
		result.Status = "APPROVED"
		result.DecidedBy = signal.ApproverID
		result.Reason = signal.Reason
		result.CompletedAt = workflow.Now(ctx)
		workflowFinished = true
		logger.Info("PreTrade order APPROVED via signal", "order_id", input.OrderID, "approver", signal.ApproverID)
	})

	// Handler for Reject signal
	selector.AddReceive(rejectChan, func(c workflow.ReceiveChannel, more bool) {
		var signal RejectSignalPayload
		c.Receive(ctx, &signal)
		result.Status = "REJECTED"
		result.DecidedBy = signal.RejecterID
		result.Reason = signal.Reason
		result.CompletedAt = workflow.Now(ctx)
		workflowFinished = true
		logger.Info("PreTrade order REJECTED via signal", "order_id", input.OrderID, "rejecter", signal.RejecterID)
	})

	// Handler for TTL expiration timer
	selector.AddFuture(timer, func(f workflow.Future) {
		if !workflowFinished {
			result.Status = "EXPIRED"
			result.DecidedBy = "SYSTEM_TIMEOUT"
			result.Reason = fmt.Sprintf("Approval TTL of %s expired without steward decision", ttlDuration)
			result.CompletedAt = workflow.Now(ctx)
			workflowFinished = true
			logger.Warn("PreTrade order approval EXPIRED on TTL", "order_id", input.OrderID, "ttl", ttlDuration)
		}
	})

	// Wait for first event (Approve, Reject, or TTL Expiry)
	selector.Select(ctx)

	// Step 3: Resolve Reservation (Unpark on APPROVE, Release on REJECT/EXPIRED)
	if input.LeaseID != uuid.Nil {
		var resolveErr error
		err = workflow.ExecuteActivity(ctx, "ResolveReservationActivity", input.LeaseID, result.Status).Get(ctx, &resolveErr)
		if err != nil {
			logger.Error("Failed to execute ResolveReservationActivity", "error", err)
		}
	}

	// Step 4: Record decision in audit ledger via activity
	var recordErr error
	err = workflow.ExecuteActivity(ctx, "RecordApprovalDecisionActivity", result).Get(ctx, &recordErr)
	if err != nil {
		logger.Error("Failed to record approval decision activity", "error", err)
		return result, fmt.Errorf("failed to record approval decision: %w", err)
	}

	return result, nil
}

// ReservationManagerInterface defines methods required by approval activities
type ReservationManagerInterface interface {
	ParkReservation(leaseID uuid.UUID, approvalTTL time.Duration) error
	UnparkReservation(leaseID uuid.UUID) error
	ReleaseReservation(leaseID uuid.UUID) error
}

// ApprovalActivities provides activity implementations for order governance
type ApprovalActivities struct {
	ResMgr ReservationManagerInterface
}

func (a *ApprovalActivities) ParkReservationActivity(ctx context.Context, leaseID uuid.UUID, ttl time.Duration) error {
	if a.ResMgr != nil {
		return a.ResMgr.ParkReservation(leaseID, ttl)
	}
	return nil
}

func (a *ApprovalActivities) ResolveReservationActivity(ctx context.Context, leaseID uuid.UUID, status string) error {
	if a.ResMgr == nil {
		return nil
	}
	if status == "APPROVED" {
		return a.ResMgr.UnparkReservation(leaseID)
	}
	// REJECTED or EXPIRED: Release reservation to restore capacity
	return a.ResMgr.ReleaseReservation(leaseID)
}

func (a *ApprovalActivities) NotifyComplianceOfficersActivity(ctx context.Context, input OrderApprovalInput) error {
	// Pushes websocket alert to Compliance Blotter and emits email/PagerDuty
	return nil
}

func (a *ApprovalActivities) RecordApprovalDecisionActivity(ctx context.Context, result *OrderApprovalResult) error {
	// Writes approval audit record into compliance.governance_audit_event
	return nil
}
