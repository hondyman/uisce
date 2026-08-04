package workflows

import (
	"fmt"
	"os"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type CustomizationIntelligenceInput struct {
	RunID        string `json:"run_id"`
	LookbackDays int    `json:"lookback_days"` // default 90
	MinTenants   int    `json:"min_tenants"`   // default 5
}

// CustomizationIntelligenceWorkflow runs the customization_clusters ETL job
// which reads CREATE events from audit_logs, clusters similar roles/policies using
// TF-IDF + MiniBatchKMeans, and upserts recommendations to fact_customization_telemetry.
//
// Cron schedule: "0 3 * * *" — runs at 03:00 UTC daily, after overnight audit ingestion.
// This pipeline populates data read by the Go /api/marketplace/product-evolution endpoint.
func CustomizationIntelligenceWorkflow(ctx workflow.Context, input CustomizationIntelligenceInput) error {
	logger := workflow.GetLogger(ctx)

	if input.LookbackDays == 0 {
		input.LookbackDays = 90
	}
	if input.MinTenants == 0 {
		input.MinTenants = 5
	}

	logger.Info("CustomizationIntelligenceWorkflow started",
		"runID", input.RunID,
		"lookbackDays", input.LookbackDays,
		"minTenants", input.MinTenants)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 30,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
			InitialInterval: time.Minute,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Run the Python ETL script as a Temporal activity.
	// scriptPath is set by the worker environment (e.g. /app/intelligence-engine/jobs/customization_clusters.py)
	scriptPath := os.Getenv("CUSTOMIZATION_ETL_SCRIPT_PATH")
	if scriptPath == "" {
		scriptPath = "/app/intelligence-engine/jobs/customization_clusters.py"
	}

	var result string
	err := workflow.ExecuteActivity(ctx,
		"RunCustomizationIntelligenceETL",
		input.RunID,
		scriptPath,
		input.LookbackDays,
		input.MinTenants,
	).Get(ctx, &result)
	if err != nil {
		logger.Error("CustomizationIntelligenceWorkflow failed", "error", err)
		return fmt.Errorf("ETL activity failed: %w", err)
	}

	logger.Info("CustomizationIntelligenceWorkflow completed", "result", result)
	return nil
}
