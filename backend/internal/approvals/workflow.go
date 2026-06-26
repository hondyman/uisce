package approvals

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// ProcessApprovalWorkflow orchestrates the multi-stage approval process.
func ProcessApprovalWorkflow(ctx workflow.Context, req ApprovalRequest) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting ProcessApprovalWorkflow", "RequestID", req.ID)

	// Define stages based on risk level
	stages := []string{"PeerReview", "DomainReview"}
	if req.RiskLevel == "High" {
		stages = append(stages, "ComplianceReview", "SecurityGovernance", "ExecutiveSignoff")
	} else if req.RiskLevel == "Medium" {
		stages = append(stages, "ComplianceReview")
	}

	for _, stage := range stages {
		logger.Info("Entering Approval Stage", "Stage", stage)
		
		// Update request status/stage (in a real app, we'd call an activity to update DB)
		req.Stage = stage
		
		// Wait for approval signal
		var decision ApprovalDecision
		signalChan := workflow.GetSignalChannel(ctx, "ApprovalDecisionSignal")
		
		selector := workflow.NewSelector(ctx)
		selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, &decision)
		})

		// SLA Timer
		// For demo, 24 hours for all stages
		timer := workflow.NewTimer(ctx, 24*time.Hour)
		selector.AddFuture(timer, func(f workflow.Future) {
			logger.Info("Approval SLA Breached", "Stage", stage)
			// In real app: Send escalation notification
		})

		selector.Select(ctx)

		if decision.Decision == StatusRejected {
			logger.Info("Request Rejected", "Stage", stage, "Actor", decision.ActorID)
			// Emit Audit Log for Rejection
			return nil // End workflow
		}

		logger.Info("Stage Approved", "Stage", stage, "Actor", decision.ActorID)
		// Emit Audit Log for Approval
	}

	logger.Info("All Stages Approved. Ready for Publish.")
	// Trigger Publish Activity
	return nil
}
