package workflows

import (
	"time"

	"go.temporal.io/sdk/workflow"

	"github.com/hondyman/semlayer/backend/internal/cbo"
	"github.com/hondyman/semlayer/backend/internal/observability/activities"
)

// SLOEvaluationWorkflowInput is the input for the workflow
type SLOEvaluationWorkflowInput struct {
	Env string
}

// SLOEvaluationWorkflow orchestration for checking SLOs
func SLOEvaluationWorkflow(ctx workflow.Context, input SLOEvaluationWorkflowInput) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)
	logger := workflow.GetLogger(ctx)

	// Define activities
	var a *activities.SLOActivities

	logger.Info("Starting SLO Evaluation Workflow", "Env", input.Env)

	// 1. Load active SLOs
	var slos []cbo.SLODefinition
	err := workflow.ExecuteActivity(ctx, a.LoadActiveSLOsActivity, input.Env).Get(ctx, &slos)
	if err != nil {
		logger.Error("Failed to load active SLOs", "Error", err)
		return err
	}

	logger.Info("Loaded active SLOs", "Count", len(slos))

	// 2. Evaluate each SLO
	for _, slo := range slos {
		var eval cbo.SLOEvaluation
		err := workflow.ExecuteActivity(ctx, a.EvaluateSLOActivity, slo).Get(ctx, &eval)
		if err != nil {
			logger.Error("Failed to evaluate SLO", "SLOID", slo.ID, "Error", err)
			continue // Continue with others
		}

		// 3. Handle violations if needed
		// Note: The evaluator itself triggers ASO tuning updates
		// We could add external notifications here if needed
		if eval.Status == "violated" {
			logger.Warn("SLO Violation detected", "SLOID", slo.ID, "Scope", slo.ScopeID)
			_ = workflow.ExecuteActivity(ctx, a.HandleSLOViolationActivity, eval)
		}
	}

	// 4. Sleep and continue as new (Periodic execution)
	// Run every 15 minutes
	logger.Info("SLO Evaluation cycle complete, sleeping...")
	if err := workflow.Sleep(ctx, 15*time.Minute); err != nil {
		return err
	}

	return workflow.NewContinueAsNewError(ctx, SLOEvaluationWorkflow, input)
}
