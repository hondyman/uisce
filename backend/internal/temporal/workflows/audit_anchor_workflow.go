package workflows

import (
	"fmt"
	"time"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	AuditAnchorScheduleID = "violation-audit-anchor-sentinel"
	AuditAnchorInterval   = 1 * time.Hour
)

type AuditAnchorWorkflowInput struct {
	TenantID     string `json:"tenant_id"`
	MaxBatchSize int    `json:"max_batch_size,omitempty"`
}

type AuditAnchorWorkflowResult struct {
	Anchored bool `json:"anchored"`
	RowCount int  `json:"row_count"`
	SeqFrom  int64 `json:"seq_from"`
	SeqTo    int64 `json:"seq_to"`
	AnchorHash string `json:"anchor_hash"`
}

// ViolationAuditAnchorWorkflow runs on a Temporal Schedule to fold unchained violation
// records into hash chains and persist cryptographic anchor roots.
func ViolationAuditAnchorWorkflow(ctx workflow.Context, input AuditAnchorWorkflowInput) (*AuditAnchorWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting ViolationAuditAnchorWorkflow", "tenant_id", input.TenantID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		RetryPolicy: &sdktemporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var acts *activities.AuditAnchorActivities
	var res activities.AuditAnchorActivityResult

	actInput := activities.AuditAnchorActivityInput{
		TenantID:     input.TenantID,
		MaxBatchSize: input.MaxBatchSize,
	}

	err := workflow.ExecuteActivity(ctx, acts.RunAuditAnchorActivity, actInput).Get(ctx, &res)
	if err != nil {
		logger.Error("RunAuditAnchorActivity failed", "error", err, "tenant_id", input.TenantID)
		return nil, fmt.Errorf("run audit anchor activity: %w", err)
	}

	if !res.Anchored || res.AnchorInfo == nil {
		logger.Info("No unchained violations to anchor", "tenant_id", input.TenantID)
		return &AuditAnchorWorkflowResult{Anchored: false}, nil
	}

	logger.Info("ViolationAuditAnchorWorkflow completed",
		"tenant_id", input.TenantID,
		"rows", res.AnchorInfo.RowCount,
		"seq_from", res.AnchorInfo.SeqFrom,
		"seq_to", res.AnchorInfo.SeqTo,
		"anchor_hash", res.AnchorInfo.AnchorHash,
	)

	return &AuditAnchorWorkflowResult{
		Anchored:   true,
		RowCount:   res.AnchorInfo.RowCount,
		SeqFrom:    res.AnchorInfo.SeqFrom,
		SeqTo:      res.AnchorInfo.SeqTo,
		AnchorHash: res.AnchorInfo.AnchorHash,
	}, nil
}
