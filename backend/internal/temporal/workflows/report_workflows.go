package workflows

import (
	"fmt"
	"time"

"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ReportGenerationWorkflowParams contains parameters for report generation
type ReportGenerationWorkflowParams struct {
	TenantID   string                 `json:"tenant_id"`
	TemplateID string                 `json:"template_id"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ReportGenerationWorkflowResult contains the result of report generation
type ReportGenerationWorkflowResult struct {
	ExecutionID     string `json:"execution_id"`
	OutputURL       string `json:"output_url"`
	OutputSizeBytes int    `json:"output_size_bytes"`
	RowsProcessed   int    `json:"rows_processed"`
}

// ReportGenerationWorkflow orchestrates async report generation
func ReportGenerationWorkflow(ctx workflow.Context, params ReportGenerationWorkflowParams) (*ReportGenerationWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting report generation workflow", "tenant_id", params.TenantID, "template_id", params.TemplateID)

	// Activity options - 5 minute timeout
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Fetch template
	var template interface{}
	err := workflow.ExecuteActivity(ctx, "FetchTemplateActivity", params.TemplateID, params.TenantID).Get(ctx, &template)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template: %w", err)
	}

	// Step 2: Query semantic views
	var semanticData interface{}
	err = workflow.ExecuteActivity(ctx, "QuerySemanticViewsActivity", template, params.Parameters).Get(ctx, &semanticData)
	if err != nil {
		return nil, fmt.Errorf("failed to query semantic views: %w", err)
	}

	// Step 3: Transform data
	var transformedData interface{}
	err = workflow.ExecuteActivity(ctx, "TransformDataActivity", semanticData, template).Get(ctx, &transformedData)
	if err != nil {
		return nil, fmt.Errorf("failed to transform data: %w", err)
	}

	// Step 4: Generate PDF
	var pdfResult struct {
		URL       string `json:"url"`
		SizeBytes int    `json:"size_bytes"`
		Rows      int    `json:"rows"`
	}
	err = workflow.ExecuteActivity(ctx, "GeneratePDFActivity", transformedData, template).Get(ctx, &pdfResult)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	// Step 5: Store execution result
	executionID := workflow.GetInfo(ctx).WorkflowExecution.ID
	err = workflow.ExecuteActivity(ctx, "StoreExecutionResultActivity", executionID, pdfResult).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to store execution result", "error", err)
		// Don't fail workflow if storage fails
	}

	logger.Info("Report generation workflow completed", "execution_id", executionID, "output_url", pdfResult.URL)

	return &ReportGenerationWorkflowResult{
		ExecutionID:     executionID,
		OutputURL:       pdfResult.URL,
		OutputSizeBytes: pdfResult.SizeBytes,
		RowsProcessed:   pdfResult.Rows,
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
