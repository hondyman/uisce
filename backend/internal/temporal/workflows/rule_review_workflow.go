package workflows

import (
	"fmt"
	"time"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	RuleGovernanceTaskQueue = "rule-governance"

	SignalRuleApprove = "Approve"
	SignalRuleReject  = "Reject"
)

type RuleReviewWorkflowInput struct {
	TenantID   string        `json:"tenant_id"`
	RuleNodeID string        `json:"rule_node_id"`
	Version    int           `json:"version"`
	AuthorID   string        `json:"author_id"`
	ReviewSLA  time.Duration `json:"review_sla"` // e.g. 72h
	ExpirySLA  time.Duration `json:"expiry_sla"` // e.g. 168h
}

type ApprovalSignalPayload struct {
	ApproverID string `json:"approver_id"`
}

type RejectionSignalPayload struct {
	RejecterID string `json:"rejecter_id"`
	Reason     string `json:"reason"`
}

type RuleReviewWorkflowResult struct {
	RuleNodeID string `json:"rule_node_id"`
	Status     string `json:"status"` // "published" | "rejected" | "expired"
	Version    int    `json:"version,omitempty"`
	Checksum   string `json:"checksum,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// RuleReviewWorkflow orchestrates the asynchronous, durable four-eyes review lifecycle
// for a validation rule node. The workflow event history serves as the immutable SEC/MiFID audit trail.
func RuleReviewWorkflow(ctx workflow.Context, input RuleReviewWorkflowInput) (*RuleReviewWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("RuleReviewWorkflow started", "rule_node_id", input.RuleNodeID, "tenant_id", input.TenantID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &sdktemporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    5,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	reviewSLA := input.ReviewSLA
	if reviewSLA <= 0 {
		reviewSLA = 72 * time.Hour
	}
	expirySLA := input.ExpirySLA
	if expirySLA <= 0 {
		expirySLA = 168 * time.Hour // 7 days
	}

	approveChan := workflow.GetSignalChannel(ctx, SignalRuleApprove)
	rejectChan := workflow.GetSignalChannel(ctx, SignalRuleReject)

	var acts *activities.RuleGovernanceActivities

	// Phase 1: Wait for decision or SLA breach timer
	selector := workflow.NewSelector(ctx)
	var approvalPayload ApprovalSignalPayload
	var rejectionPayload RejectionSignalPayload
	var decisionMade bool
	var approved bool

	selector.AddReceive(approveChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &approvalPayload)
		decisionMade = true
		approved = true
	})

	selector.AddReceive(rejectChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &rejectionPayload)
		decisionMade = true
		approved = false
	})

	slaTimer := workflow.NewTimer(ctx, reviewSLA)
	var slaBreached bool
	selector.AddFuture(slaTimer, func(f workflow.Future) {
		slaBreached = true
	})

	selector.Select(ctx)

	// If SLA was breached before a decision, trigger escalation activity and wait for expiry timer
	if slaBreached && !decisionMade {
		logger.Warn("Review SLA timer expired, escalating", "rule_node_id", input.RuleNodeID)
		_ = workflow.ExecuteActivity(ctx, acts.EscalateReviewActivity, activities.EscalateReviewInput{
			TenantID:   input.TenantID,
			RuleNodeID: input.RuleNodeID,
			Reason:     fmt.Sprintf("review exceeded %s SLA", reviewSLA),
		}).Get(ctx, nil)

		// Wait for decision or total expiry
		expiryTimer := workflow.NewTimer(ctx, expirySLA-reviewSLA)
		expirySelector := workflow.NewSelector(ctx)
		var expired bool

		expirySelector.AddReceive(approveChan, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, &approvalPayload)
			decisionMade = true
			approved = true
		})
		expirySelector.AddReceive(rejectChan, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, &rejectionPayload)
			decisionMade = true
			approved = false
		})
		expirySelector.AddFuture(expiryTimer, func(f workflow.Future) {
			expired = true
		})

		for !decisionMade && !expired {
			expirySelector.Select(ctx)
		}

		if expired && !decisionMade {
			logger.Warn("Rule review SLA fully expired, auto-rejecting", "rule_node_id", input.RuleNodeID)
			_ = workflow.ExecuteActivity(ctx, acts.RecordRejectionActivity, activities.RejectionActivityInput{
				TenantID:   input.TenantID,
				RuleNodeID: input.RuleNodeID,
				RejecterID: "system:sla_expiry",
				Reason:     "review_expired",
			}).Get(ctx, nil)

			return &RuleReviewWorkflowResult{
				RuleNodeID: input.RuleNodeID,
				Status:     "expired",
				Reason:     "review_expired",
			}, nil
		}
	}

	// Handle decision
	if !approved {
		logger.Info("Rule rejected", "rule_node_id", input.RuleNodeID, "rejecter", rejectionPayload.RejecterID)
		err := workflow.ExecuteActivity(ctx, acts.RecordRejectionActivity, activities.RejectionActivityInput{
			TenantID:   input.TenantID,
			RuleNodeID: input.RuleNodeID,
			RejecterID: rejectionPayload.RejecterID,
			Reason:     rejectionPayload.Reason,
		}).Get(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("record rejection activity: %w", err)
		}

		return &RuleReviewWorkflowResult{
			RuleNodeID: input.RuleNodeID,
			Status:     "rejected",
			Reason:     rejectionPayload.Reason,
		}, nil
	}

	// Record Approval (enforces author != approver)
	logger.Info("Rule approved, recording approval", "rule_node_id", input.RuleNodeID, "approver", approvalPayload.ApproverID)
	err := workflow.ExecuteActivity(ctx, acts.RecordApprovalActivity, activities.ApprovalActivityInput{
		TenantID:   input.TenantID,
		RuleNodeID: input.RuleNodeID,
		ApproverID: approvalPayload.ApproverID,
	}).Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("record approval activity: %w", err)
	}

	// Publish version (writes to validation_rule_versions & fires trg_catalog_node_notify)
	var pubResult activities.PublishVersionResult
	err = workflow.ExecuteActivity(ctx, acts.PublishVersionActivity, activities.PublishVersionActivityInput{
		TenantID:    input.TenantID,
		RuleNodeID:  input.RuleNodeID,
		PublisherID: approvalPayload.ApproverID,
	}).Get(ctx, &pubResult)
	if err != nil {
		return nil, fmt.Errorf("publish version activity: %w", err)
	}

	logger.Info("Rule review completed and version published",
		"rule_node_id", input.RuleNodeID, "version", pubResult.Version, "checksum", pubResult.Checksum)

	return &RuleReviewWorkflowResult{
		RuleNodeID: input.RuleNodeID,
		Status:     "published",
		Version:    pubResult.Version,
		Checksum:   pubResult.Checksum,
	}, nil
}
