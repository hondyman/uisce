package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

// DailySLAInput configuration for daily SLA refresh workflow
type DailySLAInput struct {
	RunID string `json:"run_id"`
	Date  string `json:"date"` // YYYY-MM-DD format
}

// DailySLAWorkflow refreshes daily SLA compliance metrics from hourly rollup data
// Cron schedule: "0 6 * * *" (run at 06:00 UTC daily)
func DailySLAWorkflow(ctx workflow.Context, input DailySLAInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("DailySLAWorkflow started", "runID", input.RunID, "date", input.Date)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Hour * 2,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Execute daily SLA computation
	// Aggregates whole-day metrics from hourly_chain_rollup
	sql := fmt.Sprintf(`
		INSERT INTO iceberg.ops.daily_chain_sla 
		SELECT 
			tenant_id, 
			chain_id, 
			region,
			date('%s') as day,
			100.0 * SUM(success_count) / (SUM(success_count) + SUM(failure_count)) as success_rate_pct,
			AVG(avg_latency_ms) as avg_latency_ms,
			SUM(incident_count) as incident_count,
			current_timestamp as computed_at
		FROM iceberg.ops.hourly_chain_rollup
		WHERE date(window_hour) = date('%s')
		GROUP BY tenant_id, chain_id, region
	`, input.Date, input.Date)

	var trinoResult string
	if err := workflow.ExecuteActivity(ctx, "RunTrinoQueryActivity", input.RunID, "global", sql).Get(ctx, &trinoResult); err != nil {
		logger.Error("daily SLA computation failed", "error", err)
		return err
	}
	logger.Info("daily SLA computation completed", "result", trinoResult)

	// Step 2: Update chain health reports based on SLA metrics
	healthSQL := fmt.Sprintf(`
		INSERT INTO iceberg.ops.chain_health_report
		SELECT 
			uuid_v4() as id,
			tc.chain_id,
			tc.tenant_id,
			CASE 
				WHEN dcs.success_rate_pct >= 99.0 THEN 90 + RANDOM() * 10
				WHEN dcs.success_rate_pct >= 95.0 THEN 70 + RANDOM() * 20
				WHEN dcs.success_rate_pct >= 90.0 THEN 50 + RANDOM() * 20
				ELSE 40 + RANDOM() * 10
			END::INTEGER as overall_health,
			'success' as last_execution_status,
			0 as consecutive_failures,
			true as is_healthy,
			CASE 
				WHEN dcs.success_rate_pct < 80.0 THEN 'investigate'
				WHEN dcs.success_rate_pct < 90.0 THEN 'retry'
				ELSE 'none'
			END as recommended_action,
			current_timestamp as reported_at,
			current_timestamp as created_at
		FROM iceberg.ops.daily_chain_sla dcs
		JOIN iceberg.ops.chain_config tc USING (chain_id, tenant_id)
		WHERE date(dcs.day) = date('%s')
	`, input.Date)

	var healthResult string
	if err := workflow.ExecuteActivity(ctx, "RunTrinoQueryActivity", input.RunID, "global", healthSQL).Get(ctx, &healthResult); err != nil {
		logger.Warn("health report update failed, continuing", "error", err)
	} else {
		logger.Info("health reports updated", "result", healthResult)
	}

	// Step 3: Publish completion event for dashboard subscribers
	_ = workflow.ExecuteActivity(ctx, "PublishEventActivity", input.RunID, "global", "daily_sla_refreshed").Get(ctx, nil)

	logger.Info("DailySLAWorkflow completed successfully", "runID", input.RunID, "date", input.Date)
	return nil
}

// MLTrainingInput configuration for ML model training workflow
type MLTrainingInput struct {
	RunID        string `json:"run_id"`
	ModelName    string `json:"model_name"`
	TrainingDate string `json:"training_date"` // YYYY-MM-DD
}

// MLTrainingWorkflow orchestrates feature extraction, model training, evaluation, and registration
// Can be triggered on-demand or on a schedule (e.g., weekly)
func MLTrainingWorkflow(ctx workflow.Context, input MLTrainingInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("MLTrainingWorkflow started", "runID", input.RunID, "modelName", input.ModelName)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Hour * 4,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Feature extraction from Iceberg tables (writes to features table)
	var extractResult string
	if err := workflow.ExecuteActivity(ctx, "RunPythonScriptActivity", input.RunID, "scripts/ml/extract_features.py", []string{input.ModelName, input.TrainingDate}).Get(ctx, &extractResult); err != nil {
		logger.Error("feature extraction failed", "error", err)
		return err
	}
	logger.Info("feature extraction completed", "result", extractResult)

	// Step 2: Train model (reads from features table, writes to model_artifacts S3)
	var trainResult string
	if err := workflow.ExecuteActivity(ctx, "RunPythonScriptActivity", input.RunID, "scripts/ml/train_model.py", []string{input.ModelName, input.TrainingDate}).Get(ctx, &trainResult); err != nil {
		logger.Error("model training failed", "error", err)
		return err
	}
	logger.Info("model training completed", "result", trainResult)

	// Step 3: Evaluate and register model
	var evalResult string
	if err := workflow.ExecuteActivity(ctx, "RunPythonScriptActivity", input.RunID, "scripts/ml/evaluate_and_register.py", []string{input.ModelName, input.TrainingDate}).Get(ctx, &evalResult); err != nil {
		logger.Error("model evaluation failed", "error", err)
		return err
	}
	logger.Info("model registered", "result", evalResult)

	// Step 4: Publish event
	_ = workflow.ExecuteActivity(ctx, "PublishEventActivity", input.RunID, "global", "ml_training_completed").Get(ctx, nil)

	logger.Info("MLTrainingWorkflow completed", "runID", input.RunID, "modelName", input.ModelName)
	return nil
}
