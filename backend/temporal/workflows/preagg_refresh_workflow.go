package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// RefreshPreAggWorkflow orchestrates the refresh of a pre-aggregation.
// This workflow handles refreshing both the Iceberg rollup table (via Trino)
// and the StarRocks materialized view.
func RefreshPreAggWorkflow(ctx workflow.Context, input RefreshPreAggInput) (*RefreshPreAggResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting pre-aggregation refresh workflow",
		"preAggID", input.PreAggID,
		"tenantID", input.TenantID,
	)

	// Configure activity options with retry policy
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute, // Allow time for large refreshes
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    30 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	result := &RefreshPreAggResult{
		PreAggID: input.PreAggID,
		TenantID: input.TenantID,
	}

	// Step 1: Mark pre-aggregation as refreshing
	err := workflow.ExecuteActivity(ctx, "MarkPreAggRefreshingActivity", input.PreAggID).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to mark pre-agg as refreshing", "error", err)
		return nil, err
	}

	// Step 2: Refresh Iceberg rollup table via Trino
	var icebergResult RefreshLayerResult
	err = workflow.ExecuteActivity(ctx, "RefreshIcebergRollupActivity", RefreshLayerInput{
		PreAggID:   input.PreAggID,
		TenantID:   input.TenantID,
		TargetName: input.IcebergTable,
	}).Get(ctx, &icebergResult)
	if err != nil {
		// Mark as failed and return
		_ = workflow.ExecuteActivity(ctx, "MarkPreAggFailedActivity", MarkPreAggFailedInput{
			PreAggID:     input.PreAggID,
			ErrorMessage: err.Error(),
			Layer:        "iceberg",
		}).Get(ctx, nil)
		return nil, err
	}
	result.IcebergRefreshed = true
	result.IcebergRowCount = icebergResult.RowCount

	// Step 3: Refresh StarRocks materialized view
	var starrocksResult RefreshLayerResult
	err = workflow.ExecuteActivity(ctx, "RefreshStarRocksMVActivity", RefreshLayerInput{
		PreAggID:   input.PreAggID,
		TenantID:   input.TenantID,
		TargetName: input.StarRocksMV,
	}).Get(ctx, &starrocksResult)
	if err != nil {
		// Mark as failed but Iceberg succeeded
		_ = workflow.ExecuteActivity(ctx, "MarkPreAggFailedActivity", MarkPreAggFailedInput{
			PreAggID:     input.PreAggID,
			ErrorMessage: err.Error(),
			Layer:        "starrocks",
		}).Get(ctx, nil)
		return nil, err
	}
	result.StarRocksRefreshed = true
	result.StarRocksRowCount = starrocksResult.RowCount

	// Step 4: Update statistics (row count, size)
	var stats PreAggStats
	err = workflow.ExecuteActivity(ctx, "FetchPreAggStatsActivity", input.PreAggID).Get(ctx, &stats)
	if err != nil {
		logger.Warn("Failed to fetch stats, continuing anyway", "error", err)
	} else {
		result.RowCount = stats.RowCount
		result.SizeBytes = stats.SizeBytes
	}

	// Step 5: Mark pre-aggregation as active
	err = workflow.ExecuteActivity(ctx, "MarkPreAggActiveActivity", MarkPreAggActiveInput{
		PreAggID:  input.PreAggID,
		RowCount:  result.RowCount,
		SizeBytes: result.SizeBytes,
	}).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to mark pre-agg as active", "error", err)
		return nil, err
	}

	// Step 6: Schedule next refresh if interval-based
	if input.RefreshIntervalMinutes > 0 {
		err = workflow.ExecuteActivity(ctx, "ScheduleNextRefreshActivity", ScheduleNextRefreshInput{
			PreAggID:             input.PreAggID,
			IntervalMinutes:      input.RefreshIntervalMinutes,
			NextScheduledRefresh: time.Now().Add(time.Duration(input.RefreshIntervalMinutes) * time.Minute),
		}).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to schedule next refresh", "error", err)
		}
	}

	result.Success = true
	result.Message = "Pre-aggregation refreshed successfully"
	result.RefreshedAt = time.Now()

	logger.Info("Pre-aggregation refresh workflow complete",
		"preAggID", input.PreAggID,
		"icebergRows", result.IcebergRowCount,
		"starrocksRows", result.StarRocksRowCount,
	)

	return result, nil
}

// RefreshPreAggInput is the workflow input
type RefreshPreAggInput struct {
	PreAggID               string `json:"preagg_id"`
	TenantID               string `json:"tenant_id"`
	IcebergTable           string `json:"iceberg_table"`
	StarRocksMV            string `json:"starrocks_mv"`
	RefreshIntervalMinutes int    `json:"refresh_interval_minutes"`
}

// RefreshPreAggResult is the workflow result
type RefreshPreAggResult struct {
	PreAggID           string    `json:"preagg_id"`
	TenantID           string    `json:"tenant_id"`
	Success            bool      `json:"success"`
	Message            string    `json:"message"`
	IcebergRefreshed   bool      `json:"iceberg_refreshed"`
	StarRocksRefreshed bool      `json:"starrocks_refreshed"`
	IcebergRowCount    int64     `json:"iceberg_row_count,omitempty"`
	StarRocksRowCount  int64     `json:"starrocks_row_count,omitempty"`
	RowCount           int64     `json:"row_count,omitempty"`
	SizeBytes          int64     `json:"size_bytes,omitempty"`
	RefreshedAt        time.Time `json:"refreshed_at"`
}

// RefreshLayerInput is input for layer-specific refresh activities
type RefreshLayerInput struct {
	PreAggID   string `json:"preagg_id"`
	TenantID   string `json:"tenant_id"`
	TargetName string `json:"target_name"`
}

// RefreshLayerResult is the result of a layer refresh
type RefreshLayerResult struct {
	Success  bool  `json:"success"`
	RowCount int64 `json:"row_count"`
}

// MarkPreAggFailedInput is input for marking a pre-agg as failed
type MarkPreAggFailedInput struct {
	PreAggID     string `json:"preagg_id"`
	ErrorMessage string `json:"error_message"`
	Layer        string `json:"layer"` // "iceberg" or "starrocks"
}

// MarkPreAggActiveInput is input for marking a pre-agg as active
type MarkPreAggActiveInput struct {
	PreAggID  string `json:"preagg_id"`
	RowCount  int64  `json:"row_count"`
	SizeBytes int64  `json:"size_bytes"`
}

// ScheduleNextRefreshInput is input for scheduling the next refresh
type ScheduleNextRefreshInput struct {
	PreAggID             string    `json:"preagg_id"`
	IntervalMinutes      int       `json:"interval_minutes"`
	NextScheduledRefresh time.Time `json:"next_scheduled_refresh"`
}

// PreAggStats contains statistics for a pre-aggregation
type PreAggStats struct {
	RowCount  int64 `json:"row_count"`
	SizeBytes int64 `json:"size_bytes"`
}
