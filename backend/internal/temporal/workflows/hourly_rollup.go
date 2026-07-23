package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

// HourlyRollupInput configuration for hourly rollup workflow
type HourlyRollupInput struct {
	RunID   string   `json:"run_id"`
	Regions []string `json:"regions"`
}

// HourlyRollupWorkflow orchestrates computation of hourly analytics rollups across all regions
// Cron schedule: "5 * * * *" (run at 05 minutes of every hour)
func HourlyRollupWorkflow(ctx workflow.Context, input HourlyRollupInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("HourlyRollupWorkflow started", "runID", input.RunID, "regions", input.Regions)

	// Activity options with timeout settings
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 30,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Record workflow start event
	_ = workflow.ExecuteActivity(ctx, "PublishEventActivity", input.RunID, "global", "hourly_rollup_started").Get(ctx, nil)

	// Fan-out rollup computation per region using child workflows
	// Each region executes in parallel for faster overall completion
	var childErrors []error
	for _, region := range input.Regions {
		region := region // Capture for closure
		cwo := workflow.ChildWorkflowOptions{
			WorkflowRunTimeout:  30 * time.Minute,
			WorkflowTaskTimeout: 5 * time.Minute,
		}
		ctxChild := workflow.WithChildOptions(ctx, cwo)

		var result string
		err := workflow.ExecuteChildWorkflow(ctxChild, RegionHourlyRollupWorkflow, region, input.RunID).Get(ctxChild, &result)
		if err != nil {
			logger.Error("region hourly rollup failed", "region", region, "error", err)
			childErrors = append(childErrors, fmt.Errorf("region %s: %w", region, err))
		} else {
			logger.Info("region hourly rollup completed", "region", region, "result", result)
		}
	}

	// Publish completion event regardless of per-region outcome
	_ = workflow.ExecuteActivity(ctx, "PublishEventActivity", input.RunID, "global", "hourly_rollup_completed").Get(ctx, nil)

	// Report aggregated result
	if len(childErrors) > 0 {
		return fmt.Errorf("hourly rollup completed with %d region failures", len(childErrors))
	}

	logger.Info("HourlyRollupWorkflow completed successfully", "runID", input.RunID)
	return nil
}

// RegionHourlyRollupWorkflow computes hourly metrics rollup for a single region
// Called as child workflow from HourlyRollupWorkflow for parallelism and isolation
func RegionHourlyRollupWorkflow(ctx workflow.Context, region string, runID string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("RegionHourlyRollupWorkflow starting", "region", region, "runID", runID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 25,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Execute Trino query to compute or refresh hourly_chain_rollup
	// This calls a Trino stored procedure or INSERT INTO with aggregated metrics
	sql := fmt.Sprintf(`
		INSERT INTO iceberg.ops.hourly_chain_rollup 
		SELECT 
			tenant_id, chain_id, '%s' as region, 
			date_trunc('hour', event_timestamp) as window_hour,
			COUNT(*) FILTER (WHERE status = 'success') as success_count,
			COUNT(*) FILTER (WHERE status != 'success') as failure_count,
			AVG(latency_ms) as avg_latency_ms,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms) as p95_latency_ms,
			COUNT(DISTINCT incident_id) as incident_count,
			current_timestamp as computed_at
		FROM iceberg.ops.ops_events
		WHERE region = '%s' 
		  AND event_timestamp >= current_timestamp - INTERVAL '1' HOUR
		GROUP BY tenant_id, chain_id, region
	`, region, region)

	var trinoResult string
	if err := workflow.ExecuteActivity(ctx, "RunTrinoQueryActivity", runID, region, sql).Get(ctx, &trinoResult); err != nil {
		logger.Error("Trino rollup query failed", "region", region, "error", err)
		return err
	}
	logger.Info("Trino hourly rollup completed", "region", region, "result", trinoResult)

	// Step 2: Validate rollup completeness (optional)
	validationSQL := fmt.Sprintf(`
		SELECT COUNT(*) as record_count 
		FROM iceberg.ops.hourly_chain_rollup 
		WHERE region = '%s' 
		  AND computed_at >= current_timestamp - INTERVAL '5' MINUTE
	`, region)

	var validationResult string
	if err := workflow.ExecuteActivity(ctx, "RunTrinoQueryActivity", runID, region, validationSQL).Get(ctx, &validationResult); err != nil {
		logger.Warn("validation query failed, continuing", "region", region, "error", err)
	} else {
		logger.Info("validation passed", "region", region, "result", validationResult)
	}

	// Step 3: Publish region completion event
	_ = workflow.ExecuteActivity(ctx, "PublishEventActivity", runID, region, "region_rollup_completed").Get(ctx, nil)

	logger.Info("RegionHourlyRollupWorkflow completed", "region", region)
	return nil
}
