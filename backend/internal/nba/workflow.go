package nba

import (
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

// Continuous monitoring workflow
func ClientSignalMonitorWorkflow(ctx workflow.Context, clientID uuid.UUID) error {
	logger := workflow.GetLogger(ctx)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Run indefinitely, checking signals every 4 hours
	for {
		var signals []DetectedSignal

		// Activity: Scan all signal sources
		var a *Activities
		err := workflow.ExecuteActivity(ctx, a.ScanClientSignalsActivity, clientID).Get(ctx, &signals)

		if err != nil {
			logger.Error("Signal detection failed", "error", err)
			workflow.Sleep(ctx, 15*time.Minute) // Backoff on error
			continue
		}

		// Process each detected signal
		for _, signal := range signals {
			if signal.Strength > 0.7 { // High confidence threshold
				// Spawn NBA generation workflow
				workflow.ExecuteChildWorkflow(
					ctx,
					GenerateNextBestActionWorkflow,
					signal,
				)
			}
		}

		// Sleep until next scan (adaptive interval based on client tier)
		// For now, fixed 4 hours
		workflow.Sleep(ctx, 4*time.Hour)
	}
}

func GenerateNextBestActionWorkflow(ctx workflow.Context, signal DetectedSignal) error {
	logger := workflow.GetLogger(ctx)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 1 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var a *Activities
	var actions []NextBestAction
	err := workflow.ExecuteActivity(ctx, a.GenerateNextBestActionActivity, signal).Get(ctx, &actions)
	if err != nil {
		logger.Error("Failed to generate NBA", "error", err)
		return err
	}

	// Save recommendations
	err = workflow.ExecuteActivity(ctx, a.SaveRecommendedActionsActivity, actions).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to save NBA recommendations", "error", err)
		return err
	}

	return nil
}
