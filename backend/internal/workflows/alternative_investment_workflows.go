package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// DocumentProcessingWorkflowInput contains input for document processing workflow
type DocumentProcessingWorkflowInput struct {
	DocumentID   string `json:"documentId"`
	DocumentType string `json:"documentType"`
	InvestmentID string `json:"investmentId"`
	FilePath     string `json:"filePath"`
}

// DocumentProcessingWorkflowResult contains the result of document processing
type DocumentProcessingWorkflowResult struct {
	DocumentID     string                 `json:"documentId"`
	ProcessedAt    time.Time              `json:"processedAt"`
	ExtractedData  map[string]interface{} `json:"extractedData"`
	RequiresReview bool                   `json:"requiresReview"`
	ReviewApproved bool                   `json:"reviewApproved"`
}

// DocumentProcessingWorkflow orchestrates AI-powered document extraction with human-in-the-loop
func DocumentProcessingWorkflow(ctx workflow.Context, input DocumentProcessingWorkflowInput) (*DocumentProcessingWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting document processing workflow", "documentId", input.DocumentID, "type", input.DocumentType)

	result := &DocumentProcessingWorkflowResult{
		DocumentID: input.DocumentID,
	}

	// Activity options with retry policy
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

	// Step 1: Extract data using AI (Gemini)
	var extractedData map[string]interface{}
	err := workflow.ExecuteActivity(ctx, "ExtractDocumentData", input).Get(ctx, &extractedData)
	if err != nil {
		logger.Error("Document extraction failed", "error", err)
		// Mark document as failed
		workflow.ExecuteActivity(ctx, "MarkDocumentFailed", input.DocumentID, err.Error())
		return nil, err
	}

	result.ExtractedData = extractedData
	result.ProcessedAt = workflow.Now(ctx)

	// Step 2: Store extracted data in database
	err = workflow.ExecuteActivity(ctx, "StoreExtractedData", input.DocumentID, extractedData).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to store extracted data", "error", err)
		return nil, err
	}

	// Step 3: Determine if human review is required
	var requiresReview bool
	err = workflow.ExecuteActivity(ctx, "CheckIfReviewRequired", input.DocumentID, extractedData).Get(ctx, &requiresReview)
	if err != nil {
		requiresReview = true // Default to requiring review on error
	}

	result.RequiresReview = requiresReview

	if !requiresReview {
		// Auto-approve if high confidence
		err = workflow.ExecuteActivity(ctx, "ApplyExtractedData", input.DocumentID, input.DocumentType).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to apply extracted data", "error", err)
			return nil, err
		}
		result.ReviewApproved = true
		return result, nil
	}

	// Step 4: Wait for human review
	logger.Info("Document requires human review, waiting for approval")

	var reviewApproved bool
	reviewChannel := workflow.GetSignalChannel(ctx, "document_review_complete")

	selector := workflow.NewSelector(ctx)

	// Wait for review signal
	selector.AddReceive(reviewChannel, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &reviewApproved)
	})

	// Timeout after 7 days
	selector.AddFuture(workflow.NewTimer(ctx, 7*24*time.Hour), func(f workflow.Future) {
		logger.Warn("Document review timed out after 7 days")
		reviewApproved = false
	})

	selector.Select(ctx)

	result.ReviewApproved = reviewApproved

	if !reviewApproved {
		logger.Warn("Document review not approved or timed out")
		return result, nil
	}

	// Step 5: Apply extracted data to investment records
	logger.Info("Review approved, applying extracted data")
	err = workflow.ExecuteActivity(ctx, "ApplyExtractedData", input.DocumentID, input.DocumentType).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to apply extracted data after review", "error", err)
		return nil, err
	}

	logger.Info("Document processing workflow completed successfully")
	return result, nil
}

// PerformanceCalculationWorkflowInput contains input for performance calculation
type PerformanceCalculationWorkflowInput struct {
	TenantID  string `json:"tenantId"`
	AsOfDate  string `json:"asOfDate"` // YYYY-MM-DD format
	BatchSize int    `json:"batchSize"`
}

// PerformanceCalculationWorkflow calculates performance metrics for all investments
func PerformanceCalculationWorkflow(ctx workflow.Context, input PerformanceCalculationWorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting performance calculation workflow", "tenantId", input.TenantID, "asOfDate", input.AsOfDate)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Get all active investments
	var investmentIDs []string
	err := workflow.ExecuteActivity(ctx, "GetActiveInvestmentIDs", input.TenantID).Get(ctx, &investmentIDs)
	if err != nil {
		logger.Error("Failed to get active investments", "error", err)
		return err
	}

	logger.Info("Calculating performance for investments", "count", len(investmentIDs))

	// Process in batches to avoid overwhelming the system
	batchSize := input.BatchSize
	if batchSize == 0 {
		batchSize = 10
	}

	for i := 0; i < len(investmentIDs); i += batchSize {
		end := i + batchSize
		if end > len(investmentIDs) {
			end = len(investmentIDs)
		}

		batch := investmentIDs[i:end]

		// Calculate performance for batch concurrently
		var futures []workflow.Future
		for _, invID := range batch {
			future := workflow.ExecuteActivity(ctx, "CalculateInvestmentPerformance", invID, input.AsOfDate)
			futures = append(futures, future)
		}

		// Wait for all activities in batch to complete
		for j, future := range futures {
			if err := future.Get(ctx, nil); err != nil {
				logger.Error("Performance calculation failed", "investmentId", batch[j], "error", err)
				// Continue with other investments even if one fails
			}
		}
	}

	logger.Info("Performance calculation workflow completed", "processed", len(investmentIDs))
	return nil
}

// CapitalCallForecastingWorkflowInput contains input for forecasting
type CapitalCallForecastingWorkflowInput struct {
	TenantID string `json:"tenantId"`
}

// CapitalCallForecastingWorkflow generates capital call forecasts for all investments
func CapitalCallForecastingWorkflow(ctx workflow.Context, input CapitalCallForecastingWorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting capital call forecasting workflow", "tenantId", input.TenantID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Get investments with unfunded commitments
	var investmentIDs []string
	err := workflow.ExecuteActivity(ctx, "GetInvestmentsWithUnfundedCommitments", input.TenantID).Get(ctx, &investmentIDs)
	if err != nil {
		logger.Error("Failed to get investments with unfunded commitments", "error", err)
		return err
	}

	logger.Info("Generating forecasts for investments", "count", len(investmentIDs))

	// Generate forecasts for each investment
	for _, invID := range investmentIDs {
		err := workflow.ExecuteActivity(ctx, "GenerateCapitalCallForecast", invID).Get(ctx, nil)
		if err != nil {
			logger.Error("Forecast generation failed", "investmentId", invID, "error", err)
			// Continue with other investments
		}
	}

	// Check for upcoming capital calls and send alerts
	err = workflow.ExecuteActivity(ctx, "CheckUpcomingCapitalCallsAndAlert", input.TenantID).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to check upcoming capital calls", "error", err)
		return err
	}

	logger.Info("Capital call forecasting workflow completed")
	return nil
}

// ScheduleRecurringWorkflows sets up cron schedules for recurring workflows
func ScheduleRecurringWorkflows(ctx workflow.Context) error {
	// This would typically be called from a parent workflow or configured in Temporal
	logger := workflow.GetLogger(ctx)
	logger.Info("Setting up recurring workflow schedules")

	// Performance calculation: Monthly on the 1st at 2 AM
	workflow.ExecuteChildWorkflow(ctx, PerformanceCalculationWorkflow,
		PerformanceCalculationWorkflowInput{
			TenantID: "default",
			AsOfDate: time.Now().Format("2006-01-02"),
		})

	// Capital call forecasting: Weekly on Mondays
	workflow.ExecuteChildWorkflow(ctx, CapitalCallForecastingWorkflow,
		CapitalCallForecastingWorkflowInput{
			TenantID: "default",
		})

	return nil
}
