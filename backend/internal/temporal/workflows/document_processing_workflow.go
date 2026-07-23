package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// DocumentProcessingConfig contains configuration for the document processing workflow.
type DocumentProcessingConfig struct {
	TenantID         string                 `json:"tenant_id"`
	DocumentID       string                 `json:"document_id"`
	SourceURL        string                 `json:"source_url,omitempty"`
	Filename         string                 `json:"filename"`
	ContentType      string                 `json:"content_type"`
	DocumentType     string                 `json:"document_type"`
	EntityID         string                 `json:"entity_id,omitempty"`
	ExtractMetadata  bool                   `json:"extract_metadata"`
	NotifyOnComplete bool                   `json:"notify_on_complete"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// DocumentProcessingResult contains the result of document processing.
type DocumentProcessingResult struct {
	DocumentID    string                 `json:"document_id"`
	Status        string                 `json:"status"`
	ChunkCount    int                    `json:"chunk_count"`
	PageCount     int                    `json:"page_count"`
	ExtractedData map[string]interface{} `json:"extracted_data,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	ProcessedAt   time.Time              `json:"processed_at"`
	Error         string                 `json:"error,omitempty"`
}

// DocumentProcessingWorkflow orchestrates the document processing pipeline.
func DocumentProcessingWorkflow(ctx workflow.Context, config DocumentProcessingConfig) (*DocumentProcessingResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting document processing workflow",
		"document_id", config.DocumentID,
		"document_type", config.DocumentType,
	)

	// Configure activity options with retries
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	result := &DocumentProcessingResult{
		DocumentID: config.DocumentID,
		Status:     "processing",
		Metadata:   config.Metadata,
	}

	// Step 1: Extract text from document
	var extractedText string
	extractInput := map[string]interface{}{
		"tenant_id":    config.TenantID,
		"document_id":  config.DocumentID,
		"source_url":   config.SourceURL,
		"content_type": config.ContentType,
		"filename":     config.Filename,
	}

	err := workflow.ExecuteActivity(ctx, "ExtractTextFromDocument", extractInput).Get(ctx, &extractedText)
	if err != nil {
		result.Status = "failed"
		result.Error = "text extraction failed: " + err.Error()
		return result, nil
	}

	// Step 2: Chunk the document
	var chunks []map[string]interface{}
	err = workflow.ExecuteActivity(ctx, "ChunkDocument", extractedText, config.DocumentID).Get(ctx, &chunks)
	if err != nil {
		result.Status = "failed"
		result.Error = "chunking failed: " + err.Error()
		return result, nil
	}
	result.ChunkCount = len(chunks)

	// Step 3: Generate embeddings for chunks (longer timeout)
	embeddingOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute, // Embeddings can take longer for large docs
		HeartbeatTimeout:    time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    2 * time.Minute,
			MaximumAttempts:    5,
		},
	}
	embeddingCtx := workflow.WithActivityOptions(ctx, embeddingOptions)

	var embeddedChunks []map[string]interface{}
	err = workflow.ExecuteActivity(embeddingCtx, "GenerateEmbeddings", chunks).Get(ctx, &embeddedChunks)
	if err != nil {
		result.Status = "failed"
		result.Error = "embedding generation failed: " + err.Error()
		return result, nil
	}

	// Step 4: Store chunks in database
	err = workflow.ExecuteActivity(ctx, "StoreDocumentChunks",
		config.TenantID,
		config.DocumentID,
		config.EntityID,
		embeddedChunks,
	).Get(ctx, nil)
	if err != nil {
		result.Status = "failed"
		result.Error = "storage failed: " + err.Error()
		return result, nil
	}

	// Step 5: Extract structured data (if enabled)
	if config.ExtractMetadata && extractedText != "" {
		var extractedData map[string]interface{}
		err = workflow.ExecuteActivity(ctx, "ExtractStructuredData",
			extractedText,
			config.DocumentType,
			map[string]interface{}{}, // Will use default schema
		).Get(ctx, &extractedData)

		if err != nil {
			logger.Warn("Structured data extraction failed", "error", err)
			result.ExtractedData = map[string]interface{}{
				"extraction_error": err.Error(),
			}
		} else {
			result.ExtractedData = extractedData
		}
	}

	// Step 6: Update document record
	err = workflow.ExecuteActivity(ctx, "UpdateDocumentRecord",
		config.TenantID,
		config.DocumentID,
		map[string]interface{}{
			"status":         "processed",
			"chunk_count":    len(embeddedChunks),
			"extracted_data": result.ExtractedData,
			"processed_at":   workflow.Now(ctx),
		},
	).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to update document record", "error", err)
	}

	// Step 7: Send notification if enabled
	if config.NotifyOnComplete {
		_ = workflow.ExecuteActivity(ctx, "SendDocumentNotification",
			config.TenantID,
			config.DocumentID,
			"Document processing complete",
		).Get(ctx, nil)
	}

	result.Status = "completed"
	result.ProcessedAt = workflow.Now(ctx)
	result.PageCount = estimatePages(len(extractedText))

	logger.Info("Document processing completed",
		"document_id", config.DocumentID,
		"chunk_count", result.ChunkCount,
	)

	return result, nil
}

func estimatePages(textLength int) int {
	return (textLength / 3000) + 1
}

// BatchDocumentProcessingConfig configures batch document processing.
type BatchDocumentProcessingConfig struct {
	TenantID      string                     `json:"tenant_id"`
	Documents     []DocumentProcessingConfig `json:"documents"`
	MaxConcurrent int                        `json:"max_concurrent"`
}

// BatchDocumentProcessingResult contains results of batch processing.
type BatchDocumentProcessingResult struct {
	TotalDocuments int                        `json:"total_documents"`
	Successful     int                        `json:"successful"`
	Failed         int                        `json:"failed"`
	Results        []DocumentProcessingResult `json:"results"`
}

// BatchDocumentProcessingWorkflow processes multiple documents in parallel.
func BatchDocumentProcessingWorkflow(ctx workflow.Context, config BatchDocumentProcessingConfig) (*BatchDocumentProcessingResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting batch document processing",
		"document_count", len(config.Documents),
		"max_concurrent", config.MaxConcurrent,
	)

	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 5
	}

	result := &BatchDocumentProcessingResult{
		TotalDocuments: len(config.Documents),
		Results:        make([]DocumentProcessingResult, len(config.Documents)),
	}

	// Use semaphore pattern for concurrency control
	sem := make(chan struct{}, config.MaxConcurrent)
	futures := make([]workflow.Future, len(config.Documents))

	childOptions := workflow.ChildWorkflowOptions{
		WorkflowID: "doc-process-" + workflow.GetInfo(ctx).WorkflowExecution.ID,
	}
	childCtx := workflow.WithChildOptions(ctx, childOptions)

	for i, doc := range config.Documents {
		// Acquire semaphore
		sem <- struct{}{}

		doc.TenantID = config.TenantID

		// Start child workflow
		futures[i] = workflow.ExecuteChildWorkflow(childCtx, DocumentProcessingWorkflow, doc)

		// Release semaphore when done
		go func() {
			<-sem
		}()
	}

	// Collect results
	for i, future := range futures {
		var docResult DocumentProcessingResult
		err := future.Get(ctx, &docResult)
		if err != nil {
			result.Results[i] = DocumentProcessingResult{
				DocumentID: config.Documents[i].DocumentID,
				Status:     "failed",
				Error:      err.Error(),
			}
			result.Failed++
		} else {
			result.Results[i] = docResult
			if docResult.Status == "completed" {
				result.Successful++
			} else {
				result.Failed++
			}
		}
	}

	logger.Info("Batch document processing completed",
		"total", result.TotalDocuments,
		"successful", result.Successful,
		"failed", result.Failed,
	)

	return result, nil
}

// SECFilingIngestionConfig configures SEC filing ingestion.
type SECFilingIngestionConfig struct {
	TenantID    string   `json:"tenant_id"`
	EntityID    string   `json:"entity_id"`
	CIK         string   `json:"cik"`
	FilingTypes []string `json:"filing_types"` // e.g., ["10-K", "10-Q", "8-K"]
	StartDate   string   `json:"start_date"`
	EndDate     string   `json:"end_date"`
	MaxFilings  int      `json:"max_filings"`
}

// SECFilingIngestionWorkflow fetches and processes SEC filings.
func SECFilingIngestionWorkflow(ctx workflow.Context, config SECFilingIngestionConfig) (*BatchDocumentProcessingResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting SEC filing ingestion",
		"cik", config.CIK,
		"filing_types", config.FilingTypes,
	)

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Fetch filing list from SEC EDGAR
	var filings []map[string]interface{}
	err := workflow.ExecuteActivity(ctx, "FetchSECFilingList",
		config.CIK,
		config.FilingTypes,
		config.StartDate,
		config.EndDate,
		config.MaxFilings,
	).Get(ctx, &filings)
	if err != nil {
		return nil, err
	}

	// Convert filings to document processing configs
	documents := make([]DocumentProcessingConfig, len(filings))
	for i, filing := range filings {
		documents[i] = DocumentProcessingConfig{
			TenantID:         config.TenantID,
			DocumentID:       filing["accession_number"].(string),
			SourceURL:        filing["url"].(string),
			Filename:         filing["filename"].(string),
			ContentType:      "text/html",
			DocumentType:     filing["form_type"].(string),
			EntityID:         config.EntityID,
			ExtractMetadata:  true,
			NotifyOnComplete: false,
		}
	}

	// Process all filings using batch workflow
	batchConfig := BatchDocumentProcessingConfig{
		TenantID:      config.TenantID,
		Documents:     documents,
		MaxConcurrent: 3, // SEC rate limits
	}

	childOptions := workflow.ChildWorkflowOptions{
		WorkflowID: "sec-batch-" + config.CIK,
	}
	childCtx := workflow.WithChildOptions(ctx, childOptions)

	var result BatchDocumentProcessingResult
	err = workflow.ExecuteChildWorkflow(childCtx, BatchDocumentProcessingWorkflow, batchConfig).Get(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
