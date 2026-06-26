package temporal

import (
	"time"

	"github.com/hondyman/semlayer/backend/internal/events"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// GoldCopyConnectionPropagation orchestrates the propagation of connection changes
func GoldCopyConnectionPropagation(ctx workflow.Context, event events.GoldCopyConnectionEvent) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting Gold Copy Connection Propagation",
		"EventID", event.EventID,
		"ConnectionID", event.ConnectionID,
		"Action", event.Action)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 1 * time.Minute,
		RetryPolicy: &sdktemporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    5,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// 1. Propagate Connection to Downstream Tenants
	err := workflow.ExecuteActivity(ctx, "PropagateConnectionActivity", event).Get(ctx, nil)
	if err != nil {
		logger.Error("Propagation Failed", "Error", err)
		return err
	}

	// 2. Log Audit to Iceberg
	// We execute this even if propagation fails? No, if propagation fails we retry.
	// If it finally fails, the workflow fails, and we might want to log failure.
	// For now, simple linear sequence.
	err = workflow.ExecuteActivity(ctx, "LogConnectionAuditActivity", event).Get(ctx, nil)
	if err != nil {
		logger.Error("Audit Logging Failed", "Error", err)
		// We might not want to fail the whole workflow if audit fails?
		// But requirement says "log these actions".
		return err
	}

	logger.Info("Gold Copy Connection Propagation Completed Successfully")
	return nil
}
