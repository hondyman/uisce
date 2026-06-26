package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
)

// JobProcessor handles background processing of async jobs
type JobProcessor struct {
	queue            JobQueue
	db               *sql.DB
	operationHandler OperationHandler
	webhookNotifier  WebhookNotifier
	workerCount      int
	batchSize        int
	pollInterval     time.Duration
	stopChan         chan struct{}
	wg               sync.WaitGroup
	isRunning        bool
	mu               sync.Mutex
}

// OperationHandler defines the interface for operation-specific processing
type OperationHandler interface {
	// ProcessItem processes a single job item
	ProcessItem(ctx context.Context, job *models.AsyncJob, item *models.JobItem) (resultID *string, err error)

	// ValidateJob validates the job before processing
	ValidateJob(ctx context.Context, job *models.AsyncJob) error

	// PostProcess runs after all items are processed
	PostProcess(ctx context.Context, job *models.AsyncJob) error
}

// WebhookNotifier handles sending webhook notifications
type WebhookNotifier interface {
	// NotifyJobCompletion sends a webhook notification for completed job
	NotifyJobCompletion(ctx context.Context, job *models.AsyncJob, payload *models.JobWebhookPayload) error
}

// NewJobProcessor creates a new job processor
func NewJobProcessor(
	queue JobQueue,
	db *sql.DB,
	operationHandler OperationHandler,
	webhookNotifier WebhookNotifier,
	workerCount int,
) *JobProcessor {
	if workerCount <= 0 {
		workerCount = 4
	}

	return &JobProcessor{
		queue:            queue,
		db:               db,
		operationHandler: operationHandler,
		webhookNotifier:  webhookNotifier,
		workerCount:      workerCount,
		batchSize:        100,
		pollInterval:     5 * time.Second,
		stopChan:         make(chan struct{}),
	}
}

// Start begins processing jobs
func (jp *JobProcessor) Start(ctx context.Context) error {
	jp.mu.Lock()
	if jp.isRunning {
		jp.mu.Unlock()
		return fmt.Errorf("job processor already running")
	}
	jp.isRunning = true
	jp.mu.Unlock()

	log.Printf("[JobProcessor] Starting with %d workers", jp.workerCount)

	// Start worker goroutines
	for i := 0; i < jp.workerCount; i++ {
		jp.wg.Add(1)
		go jp.workerLoop(ctx, i)
	}

	return nil
}

// Stop gracefully stops the job processor
func (jp *JobProcessor) Stop(timeout time.Duration) error {
	jp.mu.Lock()
	if !jp.isRunning {
		jp.mu.Unlock()
		return fmt.Errorf("job processor not running")
	}
	jp.mu.Unlock()

	log.Printf("[JobProcessor] Stopping...")
	close(jp.stopChan)

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		jp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		jp.mu.Lock()
		jp.isRunning = false
		jp.mu.Unlock()
		log.Printf("[JobProcessor] Stopped gracefully")
		return nil
	case <-time.After(timeout):
		jp.mu.Lock()
		jp.isRunning = false
		jp.mu.Unlock()
		return fmt.Errorf("job processor stop timeout after %v", timeout)
	}
}

// workerLoop is the main loop for each worker goroutine
func (jp *JobProcessor) workerLoop(ctx context.Context, workerID int) {
	defer jp.wg.Done()

	log.Printf("[JobProcessor-Worker-%d] Started", workerID)

	ticker := time.NewTicker(jp.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-jp.stopChan:
			log.Printf("[JobProcessor-Worker-%d] Stopping", workerID)
			return

		case <-ticker.C:
			// Poll for jobs (one at a time in this implementation)
			jobs, err := jp.queue.Dequeue(ctx, 1)
			if err != nil {
				log.Printf("[JobProcessor-Worker-%d] Error dequeueing: %v", workerID, err)
				continue
			}

			for _, job := range jobs {
				if err := jp.processJob(ctx, job, workerID); err != nil {
					log.Printf("[JobProcessor-Worker-%d] Error processing job %s: %v", workerID, job.ID, err)
				}
			}
		}
	}
}

// processJob handles processing of a single job
func (jp *JobProcessor) processJob(ctx context.Context, job *models.AsyncJob, workerID int) error {
	log.Printf("[JobProcessor-Worker-%d] Processing job %s (%s)", workerID, job.ID, job.OperationType)

	// Mark job as started
	if err := jp.queue.MarkJobStarted(ctx, job.ID); err != nil {
		return fmt.Errorf("failed to mark job started: %w", err)
	}

	// Validate job
	if err := jp.operationHandler.ValidateJob(ctx, job); err != nil {
		errorMsg, _ := json.Marshal(map[string]string{"error": err.Error()})
		errorJSON := json.RawMessage(errorMsg)
		_ = jp.queue.FailJob(ctx, job.ID, &errorJSON)
		return fmt.Errorf("job validation failed: %w", err)
	}

	// Create job items from payload
	items, err := jp.parsePayloadToItems(job)
	if err != nil {
		errorMsg, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("failed to parse payload: %v", err)})
		errorJSON := json.RawMessage(errorMsg)
		_ = jp.queue.FailJob(ctx, job.ID, &errorJSON)
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	// Create items in database
	if err := jp.queue.CreateJobItems(ctx, job.ID, items); err != nil {
		errorMsg, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("failed to create job items: %v", err)})
		errorJSON := json.RawMessage(errorMsg)
		_ = jp.queue.FailJob(ctx, job.ID, &errorJSON)
		return fmt.Errorf("failed to create job items: %w", err)
	}

	// Process items in batches
	processedCount := 0
	succeededCount := 0
	failedCount := 0

	for _, item := range items {
		// Process the item
		resultID, err := jp.operationHandler.ProcessItem(ctx, job, item)

		// Determine item status
		var itemStatus models.ItemStatus
		var errorMsg string

		if err != nil {
			itemStatus = models.ItemStatusFailed
			errorMsg = err.Error()
			failedCount++
			log.Printf("[JobProcessor-Worker-%d] Item %d failed: %v", workerID, item.ItemIndex, err)
		} else {
			itemStatus = models.ItemStatusSucceeded
			succeededCount++
		}

		// Update item in database
		if err := jp.queue.UpdateJobItem(ctx, item.ID, itemStatus, resultID, errorMsg); err != nil {
			log.Printf("[JobProcessor-Worker-%d] Error updating item status: %v", workerID, err)
		}

		processedCount++

		// Update job progress every 10 items
		if processedCount%10 == 0 {
			_ = jp.queue.UpdateJobProgress(ctx, job.ID, processedCount, succeededCount, failedCount)
		}
	}

	// Final progress update
	if err := jp.queue.UpdateJobProgress(ctx, job.ID, processedCount, succeededCount, failedCount); err != nil {
		log.Printf("[JobProcessor-Worker-%d] Error updating final progress: %v", workerID, err)
	}

	// Run post-processing
	if err := jp.operationHandler.PostProcess(ctx, job); err != nil {
		log.Printf("[JobProcessor-Worker-%d] Post-process error: %v", workerID, err)
	}

	// Mark as completed
	if err := jp.queue.UpdateJobStatus(ctx, job.ID, models.JobStatusCompleted); err != nil {
		log.Printf("[JobProcessor-Worker-%d] Error marking job completed: %v", workerID, err)
	}

	// Send webhook notification if configured
	if job.WebhookURL != "" {
		jp.sendWebhookNotification(ctx, job, succeededCount, failedCount, workerID)
	}

	log.Printf("[JobProcessor-Worker-%d] Job %s completed (processed: %d, succeeded: %d, failed: %d)",
		workerID, job.ID, processedCount, succeededCount, failedCount)

	return nil
}

// parsePayloadToItems converts job payload to job items
func (jp *JobProcessor) parsePayloadToItems(job *models.AsyncJob) ([]*models.JobItem, error) {
	var itemsData []interface{}
	if err := json.Unmarshal(job.Payload, &itemsData); err != nil {
		return nil, fmt.Errorf("invalid payload JSON: %w", err)
	}

	var items []*models.JobItem
	for i, itemData := range itemsData {
		itemJSON, err := json.Marshal(itemData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal item: %w", err)
		}

		// Extract name if available
		var nameField string
		if m, ok := itemData.(map[string]interface{}); ok {
			if name, exists := m["name"]; exists {
				nameField = fmt.Sprintf("%v", name)
			}
		}

		items = append(items, &models.JobItem{
			ID:        "",
			JobID:     job.ID,
			ItemIndex: i,
			ItemName:  nameField,
			ItemData:  itemJSON,
			Status:    models.ItemStatusPending,
		})
	}

	return items, nil
}

// sendWebhookNotification sends completion notification via webhook
func (jp *JobProcessor) sendWebhookNotification(ctx context.Context, job *models.AsyncJob, succeeded, failed int, workerID int) {
	// Get completed job to get final status
	jobStatus, err := jp.queue.GetJobStatus(ctx, job.ID)
	if err != nil {
		log.Printf("[JobProcessor-Worker-%d] Failed to get final job status: %v", workerID, err)
		return
	}

	// Build webhook payload
	duration := int(0)
	if jobStatus.CompletedAt != nil && jobStatus.StartedAt != nil {
		duration = int(jobStatus.CompletedAt.Sub(*jobStatus.StartedAt).Seconds())
	}

	payload := &models.JobWebhookPayload{
		Event:           "bulk_operation_completed",
		JobID:           job.ID,
		OperationType:   job.OperationType,
		Status:          jobStatus.Status,
		TotalItems:      job.TotalItems,
		SucceededItems:  succeeded,
		FailedItems:     failed,
		CompletionTime:  time.Now(),
		DurationSeconds: duration,
		StatusURL:       fmt.Sprintf("/api/v1/jobs/%s", job.ID),
	}

	// Send via webhook notifier
	if err := jp.webhookNotifier.NotifyJobCompletion(ctx, job, payload); err != nil {
		log.Printf("[JobProcessor-Worker-%d] Error sending webhook: %v", workerID, err)
		_ = jp.queue.MarkWebhookSent(ctx, job.ID, false)
	} else {
		_ = jp.queue.MarkWebhookSent(ctx, job.ID, true)
	}
}

// IsRunning returns whether the processor is running
func (jp *JobProcessor) IsRunning() bool {
	jp.mu.Lock()
	defer jp.mu.Unlock()
	return jp.isRunning
}

// GetStats returns current processor statistics
func (jp *JobProcessor) GetStats(ctx context.Context, tenantID string) (*ProcessorStats, error) {
	stats, err := jp.queue.GetQueueStats(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &ProcessorStats{
		WorkerCount:     jp.workerCount,
		IsRunning:       jp.IsRunning(),
		QueuedCount:     stats.QueuedCount,
		RunningCount:    stats.RunningCount,
		CompletedCount:  stats.CompletedCount,
		FailedCount:     stats.FailedCount,
		AverageWaitTime: stats.AverageWaitTime,
		AverageDuration: stats.AverageDuration,
	}, nil
}

// ProcessorStats provides statistics about the processor
type ProcessorStats struct {
	WorkerCount     int
	IsRunning       bool
	QueuedCount     int
	RunningCount    int
	CompletedCount  int
	FailedCount     int
	AverageWaitTime time.Duration
	AverageDuration time.Duration
}
