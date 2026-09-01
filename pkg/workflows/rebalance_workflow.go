package workflows

import (
	"time"

	"github.com/hondyman/uisce/pkg/activities"
	"go.temporal.io/sdk/workflow"
)

// RebalanceWorkflowInput defines input for RebalanceWorkflow.
type RebalanceWorkflowInput struct {
	TenantID         string      `json:"tenant_id"`
	TotalOrderAmount float64     `json:"total_order_amount"`
	AccountIDs       []int64     `json:"account_ids"`
	TargetSizes      []float64   `json:"target_sizes"`
	CustomFactors    [][]float64 `json:"custom_factors"`
}

// RebalanceWorkflowResult defines result returned by RebalanceWorkflow.
type RebalanceWorkflowResult struct {
	ProcessedRows int64     `json:"processed_rows"`
	Allocations   []float64 `json:"allocations"`
	Status        string    `json:"status"`
}

// RebalanceWorkflow coordinates deterministic execution of vectorized rebalance activities.
func RebalanceWorkflow(ctx workflow.Context, input RebalanceWorkflowInput) (*RebalanceWorkflowResult, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		HeartbeatTimeout:    10 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	logger := workflow.GetLogger(ctx)
	logger.Info("Starting RebalanceWorkflow", "TenantID", input.TenantID)

	var acts *activities.VectorizedExecutionActivities
	var resp activities.RebalanceResponse

	req := activities.RebalanceRequest{
		TenantID:         input.TenantID,
		TotalOrderAmount: input.TotalOrderAmount,
		AccountIDs:       input.AccountIDs,
		TargetSizes:      input.TargetSizes,
		CustomFactors:    input.CustomFactors,
	}

	err := workflow.ExecuteActivity(ctx, acts.VectorizedRebalanceActivity, req).Get(ctx, &resp)
	if err != nil {
		logger.Error("VectorizedRebalanceActivity failed", "error", err)
		return nil, err
	}

	logger.Info("Completed RebalanceWorkflow", "processedRows", resp.ProcessedRows)

	return &RebalanceWorkflowResult{
		ProcessedRows: resp.ProcessedRows,
		Allocations:   resp.Allocations,
		Status:        "COMPLETED",
	}, nil
}
