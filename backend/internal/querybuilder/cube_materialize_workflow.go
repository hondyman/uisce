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
	IcebergTable        string `json:"iceberg_table,omitempty"`
	Noop                bool   `json:"noop"`
	NoopReason          string `json:"noop_reason,omitempty"`
	HotApplied          bool   `json:"hot_applied"`
	ColdCommitted       bool   `json:"cold_committed"`
	RowCount            int64  `json:"row_count,omitempty"`
	DualCommitWatermark string `json:"dual_commit_watermark,omitempty"` // RFC3339 when set
	CompensatedHot      bool   `json:"compensated_hot,omitempty"`
}

// CubeMaterializeWorkflow validates → hot StarRocks → cold Iceberg → dual-commit Active.
//
// Lives in querybuilder (with schedule-style co-location) because
// handlers → temporal/workflows → querybuilder would form an import cycle.
//
// Workflow ID: cube-materialize-{tenant}-{cube}-v{ver}-{grain_hash}
// with single-flight while open (see CubeMaterializeStartOptions).
//
// Dual-commit (CUBE-1.3): Active + DualCommitWatermark only after both tiers OK.
// Cold failure compensates by dropping the hot MV and marking Failed.
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
			MaximumAttempts: 1, // partial physical apply must not auto-retry
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
		IcebergTable:        plan.IcebergTable,
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

	dropStaging := func() {
		if len(plan.StagingTables) == 0 {
			return
		}
		_ = workflow.ExecuteActivity(bookkeeping, ActCubeDropStaging, &plan).Get(ctx, nil)
	}

	// Track C: federated Extract-N → staging before ApplyHot when gated on.
	if plan.ExtractEnabled && len(plan.FederationSources) > 0 {
		var extracted CubeExtractResult
		if err := workflow.ExecuteActivity(long, ActCubeExtractSources, &plan).Get(ctx, &extracted); err != nil {
			dropStaging()
			_ = workflow.ExecuteActivity(bookkeeping, ActCubeFailAttempt, CubeFailAttemptInput{
				Plan:         &plan,
				ErrorMessage: err.Error(),
			}).Get(ctx, nil)
			return nil, err
		}
		if extracted.Plan != nil {
			plan = *extracted.Plan
		}
	}

	var hot CubeMaterializeHotResult
	if err := workflow.ExecuteActivity(long, ActCubeApplyHot, &plan).Get(ctx, &hot); err != nil {
		dropStaging()
		_ = workflow.ExecuteActivity(bookkeeping, ActCubeFailAttempt, CubeFailAttemptInput{
			Plan:         &plan,
			ErrorMessage: err.Error(),
		}).Get(ctx, nil)
		return nil, err
	}
	out.HotApplied = hot.AppliedDDL
	out.RowCount = hot.RowCount

	var cold CubeMaterializeColdResult
	if err := workflow.ExecuteActivity(long, ActCubeApplyCold, &plan, &hot).Get(ctx, &cold); err != nil {
		_ = workflow.ExecuteActivity(bookkeeping, ActCubeCompensateHot, &plan).Get(ctx, nil)
		out.CompensatedHot = true
		dropStaging()
		_ = workflow.ExecuteActivity(bookkeeping, ActCubeFailAttempt, CubeFailAttemptInput{
			Plan:         &plan,
			ErrorMessage: err.Error(),
		}).Get(ctx, nil)
		return out, err
	}
	out.ColdCommitted = cold.Applied
	out.IcebergTable = cold.IcebergTable
	if cold.RowCount > 0 {
		out.RowCount = cold.RowCount
	}

	if err := workflow.ExecuteActivity(bookkeeping, ActCubeCompleteDualCommit, &plan, &hot, &cold).Get(ctx, nil); err != nil {
		dropStaging()
		return nil, fmt.Errorf("hot+cold succeeded but dual-commit Active failed: %w", err)
	}
	dropStaging()
	if !cold.CommittedAt.IsZero() {
		out.DualCommitWatermark = cold.CommittedAt.UTC().Format(time.RFC3339Nano)
	}

	logger.Info("CubeMaterializeWorkflow complete",
		"materialization", hot.MaterializationName,
		"iceberg", cold.IcebergTable,
		"rowCount", out.RowCount,
		"extractApplied", plan.ExtractApplied,
	)
	return out, nil
}
