package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go.temporal.io/sdk/workflow"
)

// ComputeRequest represents a metric computation request
type ComputeRequest struct {
	TenantID    string    `json:"tenant_id"`
	MetricID    string    `json:"metric_id"`
	CalcType    string    `json:"calc_type"`    // "pop" | "anomaly"
	PeriodLabel string    `json:"period_label"` // e.g., "2024-08"
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	RunID       string    `json:"run_id"`
}

// MetricComputeWorkflow orchestrates PoP and anomaly calculations
// It's the top-level workflow that branches based on calc_type
func MetricComputeWorkflow(ctx workflow.Context, req ComputeRequest) error {
	// Set activity options with retries
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// 1) Mark run as "running" in Postgres
	if err := workflow.ExecuteActivity(ctx, UpsertRunStatus, req, "running").Get(ctx, nil); err != nil {
		workflow.GetLogger(ctx).Error("Failed to mark run as running", "error", err)
		return err
	}

	// 2) Branch by calc_type
	switch req.CalcType {
	case "pop":
		if err := workflow.ExecuteActivity(ctx, ComputeAndMergePoP, req).Get(ctx, nil); err != nil {
			workflow.GetLogger(ctx).Error("PoP computation failed", "error", err)
			_ = workflow.ExecuteActivity(ctx, UpsertRunStatus, req, "failed").Get(ctx, nil)
			return err
		}

	case "anomaly":
		if err := workflow.ExecuteActivity(ctx, ComputeAndMergeAnomalies, req).Get(ctx, nil); err != nil {
			workflow.GetLogger(ctx).Error("Anomaly computation failed", "error", err)
			_ = workflow.ExecuteActivity(ctx, UpsertRunStatus, req, "failed").Get(ctx, nil)
			return err
		}

	default:
		return errors.New("unsupported calc_type: " + req.CalcType)
	}

	// 3) Mark success and publish event
	if err := workflow.ExecuteActivity(ctx, UpsertRunStatus, req, "success").Get(ctx, nil); err != nil {
		workflow.GetLogger(ctx).Error("Failed to mark run as success", "error", err)
		return err
	}

	if err := workflow.ExecuteActivity(ctx, PublishCompletionEvent, req).Get(ctx, nil); err != nil {
		workflow.GetLogger(ctx).Error("Failed to publish completion event (non-fatal)", "error", err)
		// Don't fail the workflow for event publication failures
	}

	// 4) Trigger Cube pre-aggregation refresh (optional, for instant dashboards)
	if err := workflow.ExecuteActivity(ctx, RefreshCubePartitions, req).Get(ctx, nil); err != nil {
		workflow.GetLogger(ctx).Warn("Cube refresh failed (non-fatal)", "error", err)
		// Don't fail the workflow for Cube refresh failures
	}

	workflow.GetLogger(ctx).Info("Metric computation workflow completed successfully",
		"metric_id", req.MetricID, "calc_type", req.CalcType, "period_label", req.PeriodLabel)

	return nil
}

// ============================================================================
// ACTIVITY FUNCTION DECLARATIONS
// These are declared here and implemented in activities.go
// ============================================================================

// UpsertRunStatus updates metric_job_runs status in Postgres
func UpsertRunStatus(ctx context.Context, req ComputeRequest, status string) error {
	return nil // Implementation in activities.go
}

// ComputeAndMergePoP orchestrates PoP calculation via Trino
func ComputeAndMergePoP(ctx context.Context, req ComputeRequest) error {
	return nil // Implementation in activities.go
}

// ComputeAndMergeAnomalies orchestrates z-score anomaly detection via Trino
func ComputeAndMergeAnomalies(ctx context.Context, req ComputeRequest) error {
	return nil // Implementation in activities.go
}

// PublishCompletionEvent emits RabbitMQ event for downstream systems
func PublishCompletionEvent(ctx context.Context, req ComputeRequest) error {
	return nil // Implementation in activities.go
}

// RefreshCubePartitions calls Cube.dev API to refresh specific partitions
func RefreshCubePartitions(ctx context.Context, req ComputeRequest) error {
	return nil // Implementation in activities.go
}

// Helper function to marshal ComputeRequest to JSON
func (r ComputeRequest) MarshalJSON() ([]byte, error) {
	type Alias ComputeRequest
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(&r),
	})
}
