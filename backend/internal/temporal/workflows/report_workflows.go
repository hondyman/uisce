package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
)

// ReportGenerationWorkflowParams contains parameters for report generation.
//
// TOCTOU CONTRACT: Template is passed as a fully hydrated snapshot captured at
// trigger-authorization time by TriggerScheduleRun. Executing against this
// snapshot — rather than re-fetching inside the worker — is both a security
// requirement (eliminates TOCTOU race where template is deleted/modified between
// dispatch and activity execution) and the correct execution semantic: the report
// that was authorized and triggered is the report that runs.
//
// IDENTITY INVARIANT: Template.TenantID and Template.CreatedByID are the
// authoritative execution identity. All activities MUST use these values when
// writing to report_executions, never the HTTP caller's credentials.
type ReportGenerationWorkflowParams struct {
	// ExecutionID is the UUID of the pre-inserted report_executions row (status='pending').
	// Activities use this to update the row to 'completed' or 'failed'.
	ExecutionID string `json:"execution_id"`
	// Template is the trigger-time snapshot — see TOCTOU CONTRACT above.
	Template activities.HydratedReportTemplate `json:"template"`
	// ScheduleID is the schedule that triggered this run (for audit metadata).
	ScheduleID  string                 `json:"schedule_id"`
	// TriggerParams contains contextual metadata (trigger time, schedule ID) passed
	// from the schedule run API. Not used for identity resolution.
	TriggerParams map[string]interface{} `json:"trigger_params"`
}

// ReportGenerationWorkflowResult contains the result of report generation.
type ReportGenerationWorkflowResult struct {
	ExecutionID     string `json:"execution_id"`
	OutputURL       string `json:"output_url"`
	OutputSizeBytes int64  `json:"output_size_bytes"`
	RowsProcessed   int    `json:"rows_processed"`
	ExecutionTimeMS int    `json:"execution_time_ms"`
	Engine          string `json:"engine"`
}

// ReportGenerationWorkflow orchestrates async report generation.
//
// Workflow execution timeout: 10 minutes (wall-clock cap). If the workflow exceeds
// this, Temporal terminates it and the execution row should be swept to 'failed'
// by the stale-execution reconciler.
//
// Activity timeouts and retry policy:
//   - StartToCloseTimeout: 3 minutes per activity attempt
//   - ScheduleToCloseTimeout: 5 minutes total per activity (all attempts)
//   - MaximumAttempts: 3
//   - NonRetryableErrorTypes: ErrInvalidTemplate, ErrTenantAccessViolation, ErrSemanticViewNotFound
//
// StoreExecutionResultActivity is the sole exception — it does NOT retry on failure
// because it writes the final persistent state; a retry could write duplicate rows.
// Failure of StoreExecutionResultActivity fails the workflow so the reconciler can
// detect and mark the execution failed, rather than swallowing the error silently.
func ReportGenerationWorkflow(ctx workflow.Context, params ReportGenerationWorkflowParams) (*ReportGenerationWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	startedAt := workflow.Now(ctx)
	logger.Info("Starting report generation workflow",
		"execution_id", params.ExecutionID,
		"tenant_id", params.Template.TenantID,
		"template_id", params.Template.ID,
		"template_name", params.Template.TemplateName,
		"created_by_id", params.Template.CreatedByID,
	)

	// Pinned activity options: 3m StartToClose, 5m ScheduleToClose,
	// bounded retries with non-retryable identity/template errors.
	ao := workflow.ActivityOptions{
		StartToCloseTimeout:    3 * time.Minute,
		ScheduleToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
			// Non-retryable: identity/tenant violations and missing resources.
			// Retrying these would just fail again and waste quota.
			NonRetryableErrorTypes: []string{
				"ErrInvalidTemplate",
				"ErrTenantAccessViolation",
				"ErrSemanticViewNotFound",
			},
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Query semantic views (empty SemanticViewIDs is valid — returns zero-row result).
	var actImpl *activities.ReportActivities
	var semanticResult map[string]interface{}
	err := workflow.ExecuteActivity(ctx, actImpl.QuerySemanticViewsActivity, params.Template, params.TriggerParams).Get(ctx, &semanticResult)
	if err != nil {
		return nil, fmt.Errorf("report generation failed at QuerySemanticViewsActivity: %w", err)
	}

	// Step 2: Generate execution artifact with honest metadata.
	input := activities.GenerateArtifactInput{
		ExecutionID:  params.ExecutionID,
		Template:     params.Template,
		Params:       params.TriggerParams,
		StartedAt:    startedAt,
	}
	if v, ok := semanticResult["views_queried"]; ok {
		if vi, ok := v.(int); ok {
			input.ViewsQueried = vi
		}
	}

	var artifactResult activities.ArtifactResult
	err = workflow.ExecuteActivity(ctx, actImpl.GenerateArtifactActivity, input, semanticResult).Get(ctx, &artifactResult)
	if err != nil {
		return nil, fmt.Errorf("report generation failed at GenerateArtifactActivity: %w", err)
	}

	// Step 3: Persist execution result via WithTenantTransaction (satisfies FORCE RLS).
	// NOTE: StoreExecutionResultActivity uses a no-retry policy — it writes the
	// terminal execution state, so retrying a partial write is more dangerous than
	// failing loudly and letting the reconciler detect the inconsistency.
	storeOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 1 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1, // No retry on storage — fail loudly, let reconciler detect
		},
	}
	storeCtx := workflow.WithActivityOptions(ctx, storeOpts)
	err = workflow.ExecuteActivity(storeCtx, actImpl.StoreExecutionResultActivity, input, artifactResult).Get(storeCtx, nil)
	if err != nil {
		// Fail the workflow — the reconciler will detect a stale 'running' row
		// and mark it 'failed'. Do NOT swallow this error.
		return nil, fmt.Errorf("report generation failed at StoreExecutionResultActivity: %w", err)
	}

	logger.Info("Report generation workflow completed",
		"execution_id", params.ExecutionID,
		"output_url", artifactResult.OutputURL,
		"execution_time_ms", artifactResult.ExecutionTimeMS,
		"engine", artifactResult.Engine,
	)

	return &ReportGenerationWorkflowResult{
		ExecutionID:     params.ExecutionID,
		OutputURL:       artifactResult.OutputURL,
		OutputSizeBytes: artifactResult.OutputSizeBytes,
		RowsProcessed:   artifactResult.RowsProcessed,
		ExecutionTimeMS: artifactResult.ExecutionTimeMS,
		Engine:          artifactResult.Engine,
	}, nil
}


// AISemanticCubeWorkflowParams contains parameters for AI semantic cube generation
type AISemanticCubeWorkflowParams struct {
	TenantID     string   `json:"tenant_id"`
	DatasourceID string   `json:"datasource_id"`
	Tables       []string `json:"tables"`
	ModelType    string   `json:"model_type"` // "gemini", "gpt-4", etc.
}

// AISemanticCubeWorkflow generates semantic views using AI
func AISemanticCubeWorkflow(ctx workflow.Context, params AISemanticCubeWorkflowParams) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting AI semantic cube workflow", "tenant_id", params.TenantID)

	// Activity options - 10 minute timeout for AI inference
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 5,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute * 2,
			MaximumAttempts:    2, // AI is expensive, limit retries
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Fetch table schemas
	var schemas interface{}
	err := workflow.ExecuteActivity(ctx, "FetchTableSchemasActivity", params.DatasourceID, params.Tables).Get(ctx, &schemas)
	if err != nil {
		return fmt.Errorf("failed to fetch table schemas: %w", err)
	}

	// Step 2: Call AI to generate semantic mappings
	var semanticMappings interface{}
	err = workflow.ExecuteActivity(ctx, "AIGenerateSemanticMappingsActivity", schemas, params.ModelType).Get(ctx, &semanticMappings)
	if err != nil {
		return fmt.Errorf("failed to generate semantic mappings: %w", err)
	}

	// Step 3: Validate and store semantic views
	err = workflow.ExecuteActivity(ctx, "StoreSemanticViewsActivity", params.TenantID, params.DatasourceID, semanticMappings).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to store semantic views: %w", err)
	}

	logger.Info("AI semantic cube workflow completed")
	return nil
}

// BatchReconciliationWorkflowParams contains parameters for batch reconciliation
type BatchReconciliationWorkflowParams struct {
	TenantID      string    `json:"tenant_id"`
	DatasourceIDs []string  `json:"datasource_ids"`
	ReportDate    time.Time `json:"report_date"`
}

// BatchReconciliationWorkflow runs nightly batch reconciliation
func BatchReconciliationWorkflow(ctx workflow.Context, params BatchReconciliationWorkflowParams) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting batch reconciliation workflow", "tenant_id", params.TenantID, "date", params.ReportDate)

	// Activity options - 30 minute timeout for large batches
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		HeartbeatTimeout:    time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 10,
			BackoffCoefficient: 1.5,
			MaximumInterval:    time.Minute * 5,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Process each datasource in parallel
	var futures []workflow.Future
	for _, dsID := range params.DatasourceIDs {
		future := workflow.ExecuteActivity(ctx, "ReconcileDatasourceActivity", params.TenantID, dsID, params.ReportDate)
		futures = append(futures, future)
	}

	// Wait for all reconciliations to complete
	for i, future := range futures {
		var result interface{}
		if err := future.Get(ctx, &result); err != nil {
			logger.Error("Reconciliation failed for datasource", "datasource_id", params.DatasourceIDs[i], "error", err)
			// Continue with other datasources
		}
	}

	// Generate reconciliation summary report
	err := workflow.ExecuteActivity(ctx, "GenerateReconciliationSummaryActivity", params.TenantID, params.ReportDate).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to generate summary", "error", err)
	}

	logger.Info("Batch reconciliation workflow completed")
	return nil
}

// ClientBurstReportWorkflowParams contains parameters for client bursting report execution
type ClientBurstReportWorkflowParams struct {
	TenantID   string    `json:"tenant_id"`
	ScheduleID string    `json:"schedule_id"`
	EvalTime   time.Time `json:"eval_time"`
}

// ClientBurstReportWorkflowResult contains the execution summary of a burst run
type ClientBurstReportWorkflowResult struct {
	BatchID           string `json:"batch_id"`
	TotalClients      int    `json:"total_clients"`
	SuccessfulRenders int    `json:"successful_renders"`
	FailedRenders     int    `json:"failed_renders"`
	Status            string `json:"status"`
}

// ClientBurstReportWorkflow orchestrates calendar evaluation and parallel client slice document generation
func ClientBurstReportWorkflow(ctx workflow.Context, params ClientBurstReportWorkflowParams) (*ClientBurstReportWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting ClientBurstReportWorkflow", "tenant_id", params.TenantID, "schedule_id", params.ScheduleID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 2,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute * 2,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Calendar validation & schedule evaluation
	var evalResult struct {
		Allowed       bool      `json:"allowed"`
		EffectiveDate time.Time `json:"effective_date"`
		BurstDim      string    `json:"burst_dimension"`
		ExportFormat  string    `json:"export_format"`
	}
	err := workflow.ExecuteActivity(ctx, "EvaluateReportCalendarActivity", params.TenantID, params.ScheduleID, params.EvalTime).Get(ctx, &evalResult)
	if err != nil {
		return nil, fmt.Errorf("failed evaluating schedule calendar: %w", err)
	}
	if !evalResult.Allowed {
		logger.Info("Execution skipped due to calendar/holiday rules", "schedule_id", params.ScheduleID)
		return &ClientBurstReportWorkflowResult{
			Status: "SKIPPED_CALENDAR",
		}, nil
	}

	// Step 2: Resolve client slices
	var clientIDs []string
	err = workflow.ExecuteActivity(ctx, "ResolveClientSlicesActivity", params.TenantID, evalResult.BurstDim).Get(ctx, &clientIDs)
	if err != nil {
		return nil, fmt.Errorf("failed resolving client slices: %w", err)
	}

	// Step 3: Initialize burst batch in ledger
	var batchID string
	err = workflow.ExecuteActivity(ctx, "InitBurstBatchActivity", params.TenantID, params.ScheduleID, evalResult.EffectiveDate).Get(ctx, &batchID)
	if err != nil {
		return nil, fmt.Errorf("failed initializing burst batch: %w", err)
	}

	// Step 4: Fan-out parallel rendering child activities
	var futures []workflow.Future
	for _, clientID := range clientIDs {
		future := workflow.ExecuteActivity(ctx, "RenderAndStoreClientArtifactActivity", params.TenantID, batchID, params.ScheduleID, clientID, evalResult.ExportFormat, evalResult.EffectiveDate)
		futures = append(futures, future)
	}

	successCount := 0
	failCount := 0
	for _, f := range futures {
		var renderOk bool
		if err := f.Get(ctx, &renderOk); err != nil || !renderOk {
			failCount++
		} else {
			successCount++
		}
	}

	// Step 5: Finalize batch status & emit outbox notifications
	finalStatus := "COMPLETED"
	if failCount > 0 {
		if successCount == 0 {
			finalStatus = "FAILED"
		} else {
			finalStatus = "PARTIAL"
		}
	}

	_ = workflow.ExecuteActivity(ctx, "FinalizeBurstBatchActivity", params.TenantID, batchID, finalStatus, len(clientIDs), successCount, failCount).Get(ctx, nil)

	// Step 6: Dispatch client distributions (Email, SFTP, Webhooks)
	if successCount > 0 {
		_ = workflow.ExecuteActivity(ctx, "DispatchClientDistributionsActivity", params.TenantID, batchID).Get(ctx, nil)
	}

	logger.Info("ClientBurstReportWorkflow completed", "batch_id", batchID, "status", finalStatus, "total", len(clientIDs), "success", successCount, "failed", failCount)

	return &ClientBurstReportWorkflowResult{
		BatchID:           batchID,
		TotalClients:      len(clientIDs),
		SuccessfulRenders: successCount,
		FailedRenders:     failCount,
		Status:            finalStatus,
	}, nil
}
