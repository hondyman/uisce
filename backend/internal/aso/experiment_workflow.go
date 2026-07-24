package aso

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// Temporal Workflow
// ============================================================================

var temporalRetryPolicy = temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumInterval:    time.Minute,
	MaximumAttempts:    5,
}

type ExperimentRunnerInput struct {
	ExperimentID uuid.UUID
}

// ExperimentRunnerWorkflow manages the lifecycle of an A/B experiment
func ExperimentRunnerWorkflow(ctx workflow.Context, input ExperimentRunnerInput) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 10,
		RetryPolicy:         &temporalRetryPolicy,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Run loop until stopped or max duration reached
	// We'll check every 5 minutes (mock interval)
	for {
		// Calculate and store metrics for the current window
		var metrics []ExperimentMetrics
		if err := workflow.ExecuteActivity(ctx, ComputeExperimentMetricsActivity, input.ExperimentID).Get(ctx, &metrics); err != nil {
			workflow.GetLogger(ctx).Error("Failed to compute metrics", "Error", err)
			// Continue despite error to not kill the experiment
		}

		// Check stopping conditions
		shouldStop := false
		if err := workflow.ExecuteActivity(ctx, CheckStopConditionActivity, input.ExperimentID).Get(ctx, &shouldStop); err != nil {
			workflow.GetLogger(ctx).Error("Failed to check stop condition", "Error", err)
		}

		if shouldStop {
			// Stop experiment
			return workflow.ExecuteActivity(ctx, StopExperimentActivity, input.ExperimentID).Get(ctx, nil)
		}

		// Wait for next cycle
		workflow.Sleep(ctx, time.Minute*5)
	}
}

// ============================================================================
// Activities
// ============================================================================

func ComputeExperimentMetricsActivity(ctx context.Context, experimentID uuid.UUID) ([]ExperimentMetrics, error) {
	// In a real implementation:
	// 1. Query telemetry for events tagged with this experimentID
	// 2. Group by variant (control/treatment)
	// 3. Insert into aso.experiment_metrics

	// Mock implementation
	return []ExperimentMetrics{}, nil
}

func CheckStopConditionActivity(ctx context.Context, experimentID uuid.UUID) (bool, error) {
	// 1. Check if manually stopped via DB status
	// 2. Check if metrics indicate significant negative impact (kill switch)
	// 3. Check if max duration reached

	return false, nil
}

func StopExperimentActivity(ctx context.Context, experimentID uuid.UUID) error {
	// Update status to stopped
	// Notify user
	return nil
}
