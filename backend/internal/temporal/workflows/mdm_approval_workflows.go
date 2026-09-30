package workflows

import (
	"fmt"
	"time"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	MDMGovernanceTaskQueue = "mdm-governance"

	SignalMDMApprove  = "MDMApprove"
	SignalMDMReject   = "MDMReject"
	SignalMDMWithdraw = "MDMWithdraw"
)

// MDMOverrideWorkflowInput defines parameters for an override approval workflow.
type MDMOverrideWorkflowInput struct {
	TenantID          string        `json:"tenant_id"`
	Entity            string        `json:"entity"`
	OverrideID        string        `json:"override_id"`
	GoldenID          string        `json:"golden_id"`
	Attribute         string        `json:"attribute"`
	Action            string        `json:"action"` // SET | CLEAR
	ProposerID        string        `json:"proposer_id"`
	ProposerName      string        `json:"proposer_name"`
	ApprovalsRequired int           `json:"approvals_required"`
	ReviewSLA         time.Duration `json:"review_sla"`
	ExpirySLA         time.Duration `json:"expiry_sla"`
}

// MDMMergeWorkflowInput defines parameters for a duplicate merge approval workflow.
type MDMMergeWorkflowInput struct {
	TenantID          string        `json:"tenant_id"`
	Entity            string        `json:"entity"`
	RequestID         string        `json:"request_id"`
	CandidateID       string        `json:"candidate_id"`
	Keep              string        `json:"keep"` // "a" | "b"
	ProposerID        string        `json:"proposer_id"`
	ProposerName      string        `json:"proposer_name"`
	ApprovalsRequired int           `json:"approvals_required"`
	ReviewSLA         time.Duration `json:"review_sla"`
	ExpirySLA         time.Duration `json:"expiry_sla"`
}

// MDMConfigWorkflowInput defines parameters for a config change maker-checker workflow.
type MDMConfigWorkflowInput struct {
	TenantID     string        `json:"tenant_id"`
	Entity       string        `json:"entity,omitempty"`
	ChangeID     string        `json:"change_id"`
	Kind         string        `json:"kind"`
	Action       string        `json:"action"`
	ProposerID   string        `json:"proposer_id"`
	ProposerName string        `json:"proposer_name"`
	ReviewSLA    time.Duration `json:"review_sla"`
	ExpirySLA    time.Duration `json:"expiry_sla"`
}

// MDMReconciliationInput defines parameters for the periodic reconciliation workflow.
type MDMReconciliationInput = activities.MDMReconciliationInput

// MDMApprovalSignalPayload represents an approval decision signal.
type MDMApprovalSignalPayload struct {
	ApproverID   string `json:"approver_id"`
	ApproverName string `json:"approver_name"`
	Comment      string `json:"comment"`
}

// MDMRejectionSignalPayload represents a rejection decision signal.
type MDMRejectionSignalPayload struct {
	RejecterID   string `json:"rejecter_id"`
	RejecterName string `json:"rejecter_name"`
	Reason       string `json:"reason"`
}

// MDMWithdrawSignalPayload represents a withdrawal signal from the proposer.
type MDMWithdrawSignalPayload struct {
	ProposerID string `json:"proposer_id"`
	Reason     string `json:"reason"`
}

// MDMApprovalWorkflowResult encapsulates the outcome of an MDM approval workflow.
type MDMApprovalWorkflowResult struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"` // "APPLIED" | "REJECTED" | "WITHDRAWN" | "EXPIRED"
	DecidedBy string    `json:"decided_by,omitempty"`
	DecidedAt time.Time `json:"decided_at,omitempty"`
	Reason    string    `json:"reason,omitempty"`
}

// MDMOverrideApprovalWorkflow coordinates the multi-party approval lifecycle for a golden attribute or price override.
func MDMOverrideApprovalWorkflow(ctx workflow.Context, input MDMOverrideWorkflowInput) (*MDMApprovalWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("MDMOverrideApprovalWorkflow started",
		"override_id", input.OverrideID,
		"entity", input.Entity,
		"tenant_id", input.TenantID,
		"required_approvals", input.ApprovalsRequired,
	)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &sdktemporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    5,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var acts *activities.MDMApprovalActivities

	// Pre-flight check: if already applied/closed, complete immediately (prevents orphan workflows)
	var statusCheck activities.MDMCheckStatusResult
	err := workflow.ExecuteActivity(ctx, acts.CheckProposalStatusActivity, activities.MDMCheckStatusInput{
		TenantID: input.TenantID,
		TargetID: input.OverrideID,
		Kind:     "OVERRIDE",
	}).Get(ctx, &statusCheck)
	if err == nil && !statusCheck.Active {
		logger.Info("Override already closed before workflow run", "status", statusCheck.Status)
		return &MDMApprovalWorkflowResult{
			ID:        input.OverrideID,
			Status:    statusCheck.Status,
			DecidedAt: workflow.Now(ctx),
		}, nil
	}

	reviewSLA := input.ReviewSLA
	if reviewSLA <= 0 {
		reviewSLA = 72 * time.Hour
	}
	expirySLA := input.ExpirySLA
	if expirySLA <= 0 {
		expirySLA = 168 * time.Hour // 7 days
	}
	if input.ApprovalsRequired <= 0 {
		input.ApprovalsRequired = 1
	}

	approveChan := workflow.GetSignalChannel(ctx, SignalMDMApprove)
	rejectChan := workflow.GetSignalChannel(ctx, SignalMDMReject)
	withdrawChan := workflow.GetSignalChannel(ctx, SignalMDMWithdraw)

	approvers := make(map[string]string) // ID -> Name
	var rejected bool
	var withdrawn bool
	var applying bool
	var rejectPayload MDMRejectionSignalPayload
	var withdrawPayload MDMWithdrawSignalPayload

	slaTimer := workflow.NewTimer(ctx, reviewSLA)
	var slaBreached bool

	for len(approvers) < input.ApprovalsRequired && !rejected && !withdrawn {
		selector := workflow.NewSelector(ctx)

		selector.AddReceive(approveChan, func(c workflow.ReceiveChannel, more bool) {
			var sig MDMApprovalSignalPayload
			c.Receive(ctx, &sig)
			if sig.ApproverID == "" || sig.ApproverID == input.ProposerID {
				logger.Warn("Proposer or empty approver attempted to approve override", "approver", sig.ApproverID)
				return
			}
			approvers[sig.ApproverID] = sig.ApproverName

			// Record vote activity (with four-eyes defense-in-depth)
			_ = workflow.ExecuteActivity(ctx, acts.RecordOverrideVoteActivity, activities.MDMOverrideVoteInput{
				TenantID:     input.TenantID,
				Entity:       input.Entity,
				OverrideID:   input.OverrideID,
				ApproverID:   sig.ApproverID,
				ApproverName: sig.ApproverName,
				Decision:     "APPROVE",
				Comment:      sig.Comment,
			}).Get(ctx, nil)
		})

		selector.AddReceive(rejectChan, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, &rejectPayload)
			// Remove approver from approval map if they previously approved (vote change)
			delete(approvers, rejectPayload.RejecterID)
			rejected = true
		})

		selector.AddReceive(withdrawChan, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, &withdrawPayload)
			// Proposer validation defense-in-depth
			if withdrawPayload.ProposerID == input.ProposerID && !applying {
				withdrawn = true
			}
		})

		if !slaBreached {
			selector.AddFuture(slaTimer, func(f workflow.Future) {
				slaBreached = true
			})
		}

		selector.Select(ctx)

		if slaBreached && len(approvers) < input.ApprovalsRequired && !rejected && !withdrawn {
			logger.Warn("Review SLA timer expired, escalating override", "override_id", input.OverrideID)
			_ = workflow.ExecuteActivity(ctx, acts.EscalateMDMReviewActivity, activities.MDMEscalateInput{
				TenantID: input.TenantID,
				Entity:   input.Entity,
				TargetID: input.OverrideID,
				Kind:     "OVERRIDE",
				Reason:   fmt.Sprintf("override approval exceeded %s SLA", reviewSLA),
			}).Get(ctx, nil)
		}
	}

	if withdrawn {
		logger.Info("Override proposal withdrawn by proposer", "override_id", input.OverrideID)
		var actRes activities.MDMActivityResult
		_ = workflow.ExecuteActivity(ctx, acts.WithdrawOverrideActivity, activities.MDMWithdrawOverrideInput{
			TenantID:   input.TenantID,
			Entity:     input.Entity,
			OverrideID: input.OverrideID,
			ActorID:    withdrawPayload.ProposerID,
			Reason:     withdrawPayload.Reason,
		}).Get(ctx, &actRes)

		return &MDMApprovalWorkflowResult{
			ID:        input.OverrideID,
			Status:    actRes.Status,
			DecidedBy: withdrawPayload.ProposerID,
			DecidedAt: workflow.Now(ctx),
			Reason:    withdrawPayload.Reason,
		}, nil
	}

	if rejected {
		logger.Info("Override proposal rejected", "override_id", input.OverrideID, "rejecter", rejectPayload.RejecterID)
		var actRes activities.MDMActivityResult
		_ = workflow.ExecuteActivity(ctx, acts.RejectOverrideActivity, activities.MDMRejectOverrideInput{
			TenantID:     input.TenantID,
			Entity:       input.Entity,
			OverrideID:   input.OverrideID,
			RejecterID:   rejectPayload.RejecterID,
			RejecterName: rejectPayload.RejecterName,
			Reason:       rejectPayload.Reason,
		}).Get(ctx, &actRes)

		return &MDMApprovalWorkflowResult{
			ID:        input.OverrideID,
			Status:    actRes.Status,
			DecidedBy: rejectPayload.RejecterID,
			DecidedAt: workflow.Now(ctx),
			Reason:    rejectPayload.Reason,
		}, nil
	}

	// Lock transition so late withdraw cannot win race
	applying = true
	logger.Info("All required approvals obtained, applying override", "override_id", input.OverrideID)
	var applyRes activities.MDMActivityResult
	err = workflow.ExecuteActivity(ctx, acts.ApplyOverrideActivity, activities.MDMApplyOverrideInput{
		TenantID:   input.TenantID,
		Entity:     input.Entity,
		OverrideID: input.OverrideID,
	}).Get(ctx, &applyRes)
	if err != nil {
		logger.Error("Failed to apply override in activity", "override_id", input.OverrideID, "error", err)
		return nil, err
	}

	return &MDMApprovalWorkflowResult{
		ID:        input.OverrideID,
		Status:    applyRes.Status,
		DecidedAt: workflow.Now(ctx),
	}, nil
}

// MDMMergeApprovalWorkflow coordinates the multi-party approval lifecycle for duplicate entity merges.
func MDMMergeApprovalWorkflow(ctx workflow.Context, input MDMMergeWorkflowInput) (*MDMApprovalWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("MDMMergeApprovalWorkflow started",
		"request_id", input.RequestID,
		"candidate_id", input.CandidateID,
		"entity", input.Entity,
		"tenant_id", input.TenantID,
	)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &sdktemporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    5,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var acts *activities.MDMApprovalActivities

	// Pre-flight status check
	var statusCheck activities.MDMCheckStatusResult
	err := workflow.ExecuteActivity(ctx, acts.CheckProposalStatusActivity, activities.MDMCheckStatusInput{
		TenantID: input.TenantID,
		TargetID: input.RequestID,
		Kind:     "MERGE",
	}).Get(ctx, &statusCheck)
	if err == nil && !statusCheck.Active {
		return &MDMApprovalWorkflowResult{
			ID:        input.RequestID,
			Status:    statusCheck.Status,
			DecidedAt: workflow.Now(ctx),
		}, nil
	}

	reviewSLA := input.ReviewSLA
	if reviewSLA <= 0 {
		reviewSLA = 72 * time.Hour
	}
	if input.ApprovalsRequired <= 0 {
		input.ApprovalsRequired = 1
	}

	approveChan := workflow.GetSignalChannel(ctx, SignalMDMApprove)
	rejectChan := workflow.GetSignalChannel(ctx, SignalMDMReject)

	approvers := make(map[string]string)
	var rejected bool
	var rejectPayload MDMRejectionSignalPayload

	slaTimer := workflow.NewTimer(ctx, reviewSLA)
	var slaBreached bool

	for len(approvers) < input.ApprovalsRequired && !rejected {
		selector := workflow.NewSelector(ctx)

		selector.AddReceive(approveChan, func(c workflow.ReceiveChannel, more bool) {
			var sig MDMApprovalSignalPayload
			c.Receive(ctx, &sig)
			if sig.ApproverID == "" || sig.ApproverID == input.ProposerID {
				return
			}
			approvers[sig.ApproverID] = sig.ApproverName

			_ = workflow.ExecuteActivity(ctx, acts.RecordMergeVoteActivity, activities.MDMMergeVoteInput{
				TenantID:     input.TenantID,
				Entity:       input.Entity,
				RequestID:    input.RequestID,
				ApproverID:   sig.ApproverID,
				ApproverName: sig.ApproverName,
				Decision:     "APPROVE",
				Comment:      sig.Comment,
			}).Get(ctx, nil)
		})

		selector.AddReceive(rejectChan, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, &rejectPayload)
			delete(approvers, rejectPayload.RejecterID)
			rejected = true
		})

		if !slaBreached {
			selector.AddFuture(slaTimer, func(f workflow.Future) {
				slaBreached = true
			})
		}

		selector.Select(ctx)

		if slaBreached && len(approvers) < input.ApprovalsRequired && !rejected {
			_ = workflow.ExecuteActivity(ctx, acts.EscalateMDMReviewActivity, activities.MDMEscalateInput{
				TenantID: input.TenantID,
				Entity:   input.Entity,
				TargetID: input.RequestID,
				Kind:     "MERGE",
				Reason:   fmt.Sprintf("merge approval exceeded %s SLA", reviewSLA),
			}).Get(ctx, nil)
		}
	}

	if rejected {
		var actRes activities.MDMActivityResult
		_ = workflow.ExecuteActivity(ctx, acts.RejectMergeActivity, activities.MDMRejectMergeInput{
			TenantID:     input.TenantID,
			Entity:       input.Entity,
			RequestID:    input.RequestID,
			RejecterID:   rejectPayload.RejecterID,
			RejecterName: rejectPayload.RejecterName,
			Reason:       rejectPayload.Reason,
		}).Get(ctx, &actRes)

		return &MDMApprovalWorkflowResult{
			ID:        input.RequestID,
			Status:    actRes.Status,
			DecidedBy: rejectPayload.RejecterID,
			DecidedAt: workflow.Now(ctx),
			Reason:    rejectPayload.Reason,
		}, nil
	}

	// Apply merge
	var applyRes activities.MDMActivityResult
	err = workflow.ExecuteActivity(ctx, acts.ApplyMergeActivity, activities.MDMApplyMergeInput{
		TenantID:  input.TenantID,
		Entity:    input.Entity,
		RequestID: input.RequestID,
	}).Get(ctx, &applyRes)
	if err != nil {
		return nil, err
	}

	return &MDMApprovalWorkflowResult{
		ID:        input.RequestID,
		Status:    applyRes.Status,
		DecidedAt: workflow.Now(ctx),
	}, nil
}

// MDMConfigChangeApprovalWorkflow coordinates maker-checker configuration change approvals.
func MDMConfigChangeApprovalWorkflow(ctx workflow.Context, input MDMConfigWorkflowInput) (*MDMApprovalWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("MDMConfigChangeApprovalWorkflow started",
		"change_id", input.ChangeID,
		"kind", input.Kind,
		"tenant_id", input.TenantID,
	)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &sdktemporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    5,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var acts *activities.MDMApprovalActivities

	// Pre-flight status check
	var statusCheck activities.MDMCheckStatusResult
	err := workflow.ExecuteActivity(ctx, acts.CheckProposalStatusActivity, activities.MDMCheckStatusInput{
		TenantID: input.TenantID,
		TargetID: input.ChangeID,
		Kind:     "CONFIG",
	}).Get(ctx, &statusCheck)
	if err == nil && !statusCheck.Active {
		return &MDMApprovalWorkflowResult{
			ID:        input.ChangeID,
			Status:    statusCheck.Status,
			DecidedAt: workflow.Now(ctx),
		}, nil
	}

	reviewSLA := input.ReviewSLA
	if reviewSLA <= 0 {
		reviewSLA = 72 * time.Hour
	}

	approveChan := workflow.GetSignalChannel(ctx, SignalMDMApprove)
	rejectChan := workflow.GetSignalChannel(ctx, SignalMDMReject)
	withdrawChan := workflow.GetSignalChannel(ctx, SignalMDMWithdraw)

	var approved bool
	var rejected bool
	var withdrawn bool
	var approvalPayload MDMApprovalSignalPayload
	var rejectPayload MDMRejectionSignalPayload
	var withdrawPayload MDMWithdrawSignalPayload

	selector := workflow.NewSelector(ctx)

	selector.AddReceive(approveChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &approvalPayload)
		if approvalPayload.ApproverID != "" && approvalPayload.ApproverID != input.ProposerID {
			approved = true
		}
	})

	selector.AddReceive(rejectChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &rejectPayload)
		rejected = true
	})

	selector.AddReceive(withdrawChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &withdrawPayload)
		if withdrawPayload.ProposerID == input.ProposerID {
			withdrawn = true
		}
	})

	slaTimer := workflow.NewTimer(ctx, reviewSLA)
	selector.AddFuture(slaTimer, func(f workflow.Future) {
		_ = workflow.ExecuteActivity(ctx, acts.EscalateMDMReviewActivity, activities.MDMEscalateInput{
			TenantID: input.TenantID,
			Entity:   input.Entity,
			TargetID: input.ChangeID,
			Kind:     "CONFIG",
			Reason:   fmt.Sprintf("config change review exceeded %s SLA", reviewSLA),
		}).Get(ctx, nil)
	})

	for !approved && !rejected && !withdrawn {
		selector.Select(ctx)
	}

	if withdrawn {
		var actRes activities.MDMActivityResult
		_ = workflow.ExecuteActivity(ctx, acts.WithdrawConfigChangeActivity, activities.MDMWithdrawConfigInput{
			TenantID: input.TenantID,
			ChangeID: input.ChangeID,
			ActorID:  withdrawPayload.ProposerID,
			Reason:   withdrawPayload.Reason,
		}).Get(ctx, &actRes)

		return &MDMApprovalWorkflowResult{
			ID:        input.ChangeID,
			Status:    actRes.Status,
			DecidedBy: withdrawPayload.ProposerID,
			DecidedAt: workflow.Now(ctx),
		}, nil
	}

	if rejected {
		var actRes activities.MDMActivityResult
		_ = workflow.ExecuteActivity(ctx, acts.RejectConfigChangeActivity, activities.MDMRejectConfigInput{
			TenantID:     input.TenantID,
			ChangeID:     input.ChangeID,
			RejecterID:   rejectPayload.RejecterID,
			RejecterName: rejectPayload.RejecterName,
			Reason:       rejectPayload.Reason,
		}).Get(ctx, &actRes)

		return &MDMApprovalWorkflowResult{
			ID:        input.ChangeID,
			Status:    actRes.Status,
			DecidedBy: rejectPayload.RejecterID,
			DecidedAt: workflow.Now(ctx),
			Reason:    rejectPayload.Reason,
		}, nil
	}

	// Apply config change
	var applyRes activities.MDMActivityResult
	err = workflow.ExecuteActivity(ctx, acts.ApplyConfigChangeActivity, activities.MDMApplyConfigInput{
		TenantID:     input.TenantID,
		ChangeID:     input.ChangeID,
		ApproverID:   approvalPayload.ApproverID,
		ApproverName: approvalPayload.ApproverName,
		Comment:      approvalPayload.Comment,
	}).Get(ctx, &applyRes)
	if err != nil {
		return nil, err
	}

	return &MDMApprovalWorkflowResult{
		ID:        input.ChangeID,
		Status:    applyRes.Status,
		DecidedBy: approvalPayload.ApproverID,
		DecidedAt: workflow.Now(ctx),
	}, nil
}

// MDMReconciliationWorkflow is a scheduled workflow that invokes ReconcilePendingMDMWorkflowsActivity.
func MDMReconciliationWorkflow(ctx workflow.Context, input MDMReconciliationInput) (*activities.MDMReconciliationResult, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &sdktemporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var acts *activities.MDMApprovalActivities
	var res activities.MDMReconciliationResult
	err := workflow.ExecuteActivity(ctx, acts.ReconcilePendingMDMWorkflowsActivity, input).Get(ctx, &res)
	return &res, err
}
