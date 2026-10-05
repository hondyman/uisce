package querybuilder

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// CubeMaterializeWorkflowResult is returned to Deploy/Refresh callers.
type CubeMaterializeWorkflowResult struct {
	TenantID            string `json:"tenant_id"`
	CubeID              string `json:"cube_id"`
	ContractVersion     int    `json:"contract_version"`
	GrainHash           string `json:"grain_hash"`
	AttemptID           string `json:"attempt_id"`
	NodeID              string `json:"node_id"`
	MaterializationName string `json:"materialization_name"`
	TargetDatabase      string `json:"target_database"`
	Noop                bool   `json:"noop"`
	NoopReason          string `json:"noop_reason,omitempty"`
	HotApplied          bool   `json:"hot_applied"`
	RowCount            int64  `json:"row_count,omitempty"`
	// ColdCommitted is reserved for CUBE-1.3 dual-commit.
	ColdCommitted bool `json:"cold_committed"`
}

// CubeMaterializeWorkflow validates → plans DDL → loads StarRocks hot grain.
//
// Lives in querybuilder (with schedule-style co-location) because
// handlers → temporal/workflows → querybuilder would form an import cycle.
//
// Workflow ID: cube-materialize-{tenant}-{cube}-v{ver}-{grain_hash}
// with REJECT_DUPLICATE (see CubeMaterializeStartOptions).
//
// Cold Iceberg commit + dual-commit watermark are CUBE-1.3 on the same attempt_id.
func CubeMaterializeWorkflow(ctx workflow.Context, req CubeMaterializeRequest) (*CubeMaterializeWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("CubeMaterializeWorkflow started",
		"tenantID", req.TenantID,
		"cubeID", req.CubeID,
		"grain", req.Grain,
	)

	short := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	})
	long := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		HeartbeatTimeout:    2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1, // partial StarRocks apply must not auto-retry
		},
	})
	bookkeeping := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	})

	var plan CubeMaterializePlan
	if err := workflow.ExecuteActivity(short, ActCubeValidateAndPlan, req).Get(ctx, &plan); err != nil {
		return nil, err
	}

	out := &CubeMaterializeWorkflowResult{
		TenantID:            plan.TenantID,
		CubeID:              plan.CubeID,
		ContractVersion:     plan.ContractVersion,
		GrainHash:           plan.GrainHash,
		AttemptID:           plan.AttemptID,
		NodeID:              plan.NodeID,
		MaterializationName: plan.MaterializationName,
		TargetDatabase:      plan.TargetDatabase,
		Noop:                plan.Noop,
		NoopReason:          plan.NoopReason,
	}

	if plan.Noop {
		logger.Info("CubeMaterializeWorkflow noop", "reason", plan.NoopReason)
		return out, nil
	}

	if err := workflow.ExecuteActivity(bookkeeping, ActCubeBeginAttempt, &plan).Get(ctx, nil); err != nil {
		return nil, err
	}

	var hot CubeMaterializeHotResult
	if err := workflow.ExecuteActivity(long, ActCubeApplyHot, &plan).Get(ctx, &hot); err != nil {
		_ = workflow.ExecuteActivity(bookkeeping, ActCubeFailAttempt, CubeFailAttemptInput{
			Plan:         &plan,
			ErrorMessage: err.Error(),
		}).Get(ctx, nil)
		return nil, err
	}

	if err := workflow.ExecuteActivity(bookkeeping, ActCubeCompleteAttempt, &plan, &hot).Get(ctx, nil); err != nil {
		return nil, fmt.Errorf("hot load succeeded but lifecycle Active failed: %w", err)
	}

	out.HotApplied = hot.AppliedDDL
	out.RowCount = hot.RowCount
	logger.Info("CubeMaterializeWorkflow complete",
		"materialization", hot.MaterializationName,
		"rowCount", hot.RowCount,
	)
	return out, nil
}
