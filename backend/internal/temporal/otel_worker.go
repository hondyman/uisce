package temporal

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// Simple temporal worker setup without OTEL instrumentation
// (Full OTEL support requires go.opentelemetry.io/contrib/instrumentation/go.temporal.io)

// SetupTemporalClient creates a Temporal client with basic configuration
func SetupTemporalClient(ctx context.Context, hostPort string) (client.Client, error) {
	c, err := client.Dial(client.Options{
		HostPort: hostPort,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client: %w", err)
	}
	return c, nil
}

// DriftRequest represents the input to a drift detection workflow
type DriftRequest struct {
	PlanID   string
	Table    string
	Region   string
	Endpoint string
}

// DriftResult represents the output of a drift detection workflow
type DriftResult struct {
	PlanID        string
	DriftDetected bool
	Region        string
}

// DriftWorkflow orchestrates drift detection across regions
func DriftWorkflow(ctx workflow.Context, req *DriftRequest) (*DriftResult, error) {
	planID := req.PlanID

	logger := workflow.GetLogger(ctx)
	logger.Info("Starting drift workflow",
		"plan_id", planID,
		"table", req.Table,
		"region", req.Region,
	)

	// Note: OTEL tracing for workflow context requires go.opentelemetry.io/contrib/instrumentation support
	// For now, using Temporal's built-in logging
	logger.Debug("Drift workflow context initialized", "plan_id", planID)

	// Configure activity options with timeouts and retries
	activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    2 * time.Minute,
			MaximumAttempts:    3,
		},
	})

	// Execute regional drift activity
	var result *DriftResult
	err := workflow.ExecuteActivity(activityCtx, RegionalDriftActivity, req).Get(ctx, &result)
	if err != nil {
		logger.Error("Regional drift activity failed", "error", err)
		return nil, err
	}

	// Drift detection complete
	logger.Debug("Workflow execution complete",
		"plan_id", planID,
		"drift_detected", result.DriftDetected,
	)

	return result, nil
}

// RegionalDriftActivity checks for drift in a specific region
func RegionalDriftActivity(ctx context.Context, req *DriftRequest) (*DriftResult, error) {
	logger := activity.GetLogger(ctx)

	logger.Info("Starting regional drift check",
		"table", req.Table,
		"region", req.Region,
		"plan_id", req.PlanID,
	)

	// Example: Call commit service to get latest snapshot
	// (In production, use your actual HTTP client with proper error handling)

	logger.Debug("Drift check complete",
		"plan_id", req.PlanID,
		"region", req.Region,
	)

	return &DriftResult{
		PlanID:        req.PlanID,
		DriftDetected: false, // Your actual drift detection logic
		Region:        req.Region,
	}, nil
}

// StartTemporalWorker starts the Temporal worker with registered workflows and activities
func StartTemporalWorker(c client.Client, taskQueue string) (worker.Worker, error) {
	w := worker.New(c, taskQueue, worker.Options{})

	// Register workflow
	w.RegisterWorkflow(DriftWorkflow)

	// Register activity
	w.RegisterActivity(RegionalDriftActivity)

	// Start the worker
	if err := w.Start(); err != nil {
		return nil, fmt.Errorf("failed to start worker: %w", err)
	}

	return w, nil
}
