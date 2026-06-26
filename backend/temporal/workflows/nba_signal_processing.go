package workflows

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/nba"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// SignalProcessingWorkflow orchestrates NBA signal detection and recommendation generation
func SignalProcessingWorkflow(ctx workflow.Context, tenantID uuid.UUID) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting NBA Signal Processing Workflow", "tenant_id", tenantID)

	// Configuration
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Step 1: Detect Portfolio Signals
	var portfolioSignals []nba.Signal
	err := workflow.ExecuteActivity(ctx, DetectPortfolioSignalsActivity, tenantID).Get(ctx, &portfolioSignals)
	if err != nil {
		logger.Error("Failed to detect portfolio signals", "error", err)
		return err
	}
	logger.Info("Detected portfolio signals", "count", len(portfolioSignals))

	// Step 2: Detect Behavioral Signals
	var behavioralSignals []nba.Signal
	err = workflow.ExecuteActivity(ctx, DetectBehavioralSignalsActivity, tenantID).Get(ctx, &behavioralSignals)
	if err != nil {
		logger.Error("Failed to detect behavioral signals", "error", err)
		return err
	}
	logger.Info("Detected behavioral signals", "count", len(behavioralSignals))

	// Step 3: Detect Lifecycle Signals
	var lifecycleSignals []nba.Signal
	err = workflow.ExecuteActivity(ctx, DetectLifecycleSignalsActivity, tenantID).Get(ctx, &lifecycleSignals)
	if err != nil {
		logger.Error("Failed to detect lifecycle signals", "error", err)
		// Non-fatal, continue
	}

	// Step 4: Save all signals
	allSignals := append(portfolioSignals, behavioralSignals...)
	allSignals = append(allSignals, lifecycleSignals...)

	if len(allSignals) > 0 {
		err = workflow.ExecuteActivity(ctx, SaveSignalsActivity, allSignals).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to save signals", "error", err)
			return err
		}
	}

	// Step 5: Generate NBA recommendations (will implement in Phase 2 with ML model)
	// For now, use rule-based recommendation generation
	logger.Info("Signal processing complete", "total_signals", len(allSignals))

	return nil
}

// DetectPortfolioSignalsActivity activity function
func DetectPortfolioSignalsActivity(ctx context.Context, tenantID uuid.UUID) ([]nba.Signal, error) {
	detector := GetSignalDetectorFromContext(ctx)
	return detector.DetectPortfolioSignals(ctx, tenantID)
}

// DetectBehavioralSignalsActivity activity function
func DetectBehavioralSignalsActivity(ctx context.Context, tenantID uuid.UUID) ([]nba.Signal, error) {
	detector := GetSignalDetectorFromContext(ctx)
	return detector.DetectBehavioralSignals(ctx, tenantID)
}

// DetectLifecycleSignalsActivity activity function
func DetectLifecycleSignalsActivity(ctx context.Context, tenantID uuid.UUID) ([]nba.Signal, error) {
	// Placeholder for lifecycle signal detection
	return []nba.Signal{}, nil
}

// SaveSignalsActivity saves signals to database
func SaveSignalsActivity(ctx context.Context, signals []nba.Signal) error {
	detector := GetSignalDetectorFromContext(ctx)
	return detector.SaveSignals(ctx, signals)
}

// GetSignalDetectorFromContext retrieves signal detector from context
func GetSignalDetectorFromContext(ctx context.Context) *nba.SignalDetector {
	// This would be injected via context in worker initialization
	// Placeholder implementation
	return nil
}

// ScheduledSignalDetectionCron runs signal detection on a schedule (e.g., every 4 hours)
func ScheduledSignalDetectionCron(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)

	// Run forever with 4-hour intervals
	for {
		// Get all active tenants
		var tenants []uuid.UUID
		err := workflow.ExecuteActivity(ctx, GetActiveTenantsActivity).Get(ctx, &tenants)
		if err != nil {
			logger.Error("Failed to get active tenants", "error", err)
			workflow.Sleep(ctx, 4*time.Hour)
			continue
		}

		// Process each tenant in parallel
		futures := []workflow.Future{}
		for _, tenantID := range tenants {
			childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
				WorkflowID: fmt.Sprintf("nba-signal-processing-%s-%d", tenantID, time.Now().Unix()),
			})
			future := workflow.ExecuteChildWorkflow(childCtx, SignalProcessingWorkflow, tenantID)
			futures = append(futures, future)
		}

		// Wait for all to complete
		for _, future := range futures {
			future.Get(ctx, nil) // Ignore errors for individual tenants
		}

		logger.Info("Signal detection cycle complete", "tenants_processed", len(tenants))

		// Sleep until next run
		workflow.Sleep(ctx, 4*time.Hour)
	}
}

// GetActiveTenantsActivity returns list of active tenant IDs
func GetActiveTenantsActivity(ctx context.Context) ([]uuid.UUID, error) {
	// Query database for active tenants
	// Placeholder
	return []uuid.UUID{}, nil
}
