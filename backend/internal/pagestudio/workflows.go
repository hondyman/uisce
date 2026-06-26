package pagestudio

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// PageUpgradeReconciliationWorkflow orchestrates the diffing of core updates against all tenant overlays
func PageUpgradeReconciliationWorkflow(ctx workflow.Context, req AnalyzeImpactRequest) error {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, options)

	var a *Activities
	var impactIDs []string // We use string for compatibility in some Temporal versions or just as identifiers

	err := workflow.ExecuteActivity(ctx, a.AnalyzeCoreUpgradeImpact, req).Get(ctx, &impactIDs)
	if err != nil {
		return err
	}

	// In a real system, we might send notifications or trigger downstream tasks for each impactID
	return nil
}
