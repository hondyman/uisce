package activities

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// --- Shared Types for Pre-Agg Refresh Workflow/Activities ---

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

// PreAggRefreshActivities contains the activities for pre-aggregation refresh workflows.
type PreAggRefreshActivities struct {
	db               *sqlx.DB
	preAggSvc        *analytics.PreAggregationService
	lifecycleSvc     *analytics.PreAggLifecycleService
	templateRenderer *analytics.PreAggTemplateRenderer
	trinoConn        *sqlx.DB // Trino connection for Iceberg
	starrocksConn    *sqlx.DB // StarRocks connection
}

// NewPreAggRefreshActivities creates a new set of pre-agg refresh activities.
func NewPreAggRefreshActivities(
	db *sqlx.DB,
	preAggSvc *analytics.PreAggregationService,
	lifecycleSvc *analytics.PreAggLifecycleService,
	trinoConn, starrocksConn *sqlx.DB,
) *PreAggRefreshActivities {
	renderer, _ := analytics.NewPreAggTemplateRenderer()
	return &PreAggRefreshActivities{
		db:               db,
		preAggSvc:        preAggSvc,
		lifecycleSvc:     lifecycleSvc,
		templateRenderer: renderer,
		trinoConn:        trinoConn,
		starrocksConn:    starrocksConn,
	}
}

// MarkPreAggRefreshingActivity marks a pre-aggregation as refreshing.
func (a *PreAggRefreshActivities) MarkPreAggRefreshingActivity(ctx context.Context, preAggID string) error {
	id, err := uuid.Parse(preAggID)
	if err != nil {
		return fmt.Errorf("invalid preagg ID: %w", err)
	}
	return a.lifecycleSvc.MarkRefreshing(ctx, id)
}

// RefreshIcebergRollupActivity refreshes the Iceberg rollup table via Trino.
func (a *PreAggRefreshActivities) RefreshIcebergRollupActivity(ctx context.Context, input RefreshLayerInput) (*RefreshLayerResult, error) {
	if a.trinoConn == nil {
		// No Trino connection, skip Iceberg refresh
		return &RefreshLayerResult{Success: true, RowCount: 0}, nil
	}

	// For Iceberg, we typically do INSERT OVERWRITE or DELETE+INSERT
	// This is a simplified version - production would need proper incremental logic
	refreshSQL := fmt.Sprintf(`
		INSERT OVERWRITE iceberg.%s_analytics.%s
		SELECT * FROM iceberg.%s_analytics.%s
	`, input.TenantID, input.TargetName, input.TenantID, input.TargetName)

	_, err := a.trinoConn.ExecContext(ctx, refreshSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh Iceberg rollup: %w", err)
	}

	// Get row count
	var rowCount int64
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM iceberg.%s_analytics.%s`, input.TenantID, input.TargetName)
	_ = a.trinoConn.GetContext(ctx, &rowCount, countSQL)

	return &RefreshLayerResult{Success: true, RowCount: rowCount}, nil
}

// RefreshStarRocksMVActivity refreshes the StarRocks materialized view.
func (a *PreAggRefreshActivities) RefreshStarRocksMVActivity(ctx context.Context, input RefreshLayerInput) (*RefreshLayerResult, error) {
	if a.starrocksConn == nil {
		// No StarRocks connection, skip
		return &RefreshLayerResult{Success: true, RowCount: 0}, nil
	}

	// Render refresh SQL using template
	data := analytics.PreAggTemplateData{
		Tenant:     input.TenantID,
		Datasource: input.TargetName, // Simplified - real impl would parse from pre-agg
		PreAggID:   input.PreAggID,
	}

	refreshSQL, err := a.templateRenderer.RenderStarRocksRefresh(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render refresh SQL: %w", err)
	}

	_, err = a.starrocksConn.ExecContext(ctx, refreshSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh StarRocks MV: %w", err)
	}

	// Get row count from information_schema
	var rowCount int64
	countSQL := fmt.Sprintf(`
		SELECT IFNULL(table_rows, 0) 
		FROM information_schema.tables 
		WHERE table_name = '%s'
	`, input.TargetName)
	_ = a.starrocksConn.GetContext(ctx, &rowCount, countSQL)

	return &RefreshLayerResult{Success: true, RowCount: rowCount}, nil
}

// MarkPreAggFailedActivity marks a pre-aggregation as failed.
func (a *PreAggRefreshActivities) MarkPreAggFailedActivity(ctx context.Context, input MarkPreAggFailedInput) error {
	id, err := uuid.Parse(input.PreAggID)
	if err != nil {
		return fmt.Errorf("invalid preagg ID: %w", err)
	}
	return a.lifecycleSvc.MarkFailed(ctx, id, errors.New(input.ErrorMessage))
}

// MarkPreAggActiveActivity marks a pre-aggregation as active with updated stats.
func (a *PreAggRefreshActivities) MarkPreAggActiveActivity(ctx context.Context, input MarkPreAggActiveInput) error {
	id, err := uuid.Parse(input.PreAggID)
	if err != nil {
		return fmt.Errorf("invalid preagg ID: %w", err)
	}

	stats := &models.PreAggStats{
		RowCount:  input.RowCount,
		SizeBytes: input.SizeBytes,
	}
	return a.lifecycleSvc.MarkActive(ctx, id, stats)
}

// FetchPreAggStatsActivity fetches statistics for a pre-aggregation from StarRocks.
func (a *PreAggRefreshActivities) FetchPreAggStatsActivity(ctx context.Context, preAggID string) (*PreAggStats, error) {
	// This would query information_schema or StarRocks system tables
	// Simplified implementation returning placeholder stats
	return &PreAggStats{
		RowCount:  0,
		SizeBytes: 0,
	}, nil
}

// ScheduleNextRefreshActivity schedules the next refresh for a pre-aggregation.
func (a *PreAggRefreshActivities) ScheduleNextRefreshActivity(ctx context.Context, input ScheduleNextRefreshInput) error {
	id, err := uuid.Parse(input.PreAggID)
	if err != nil {
		return fmt.Errorf("invalid preagg ID: %w", err)
	}

	return a.lifecycleSvc.UpdateNextScheduledRefresh(ctx, id, input.NextScheduledRefresh)
}
