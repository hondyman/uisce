package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
)

// JobQueue interface defines operations on the job queue
type JobQueue interface {
	// Enqueue adds a new job to the queue
	Enqueue(ctx context.Context, job *models.AsyncJob) (*models.AsyncJob, error)

	// Dequeue retrieves the next job to process
	Dequeue(ctx context.Context, batchSize int) ([]*models.AsyncJob, error)

	// GetJobStatus returns current status of a job
	GetJobStatus(ctx context.Context, jobID string) (*models.AsyncJob, error)

	// GetJobProgress returns progress summary for a job
	GetJobProgress(ctx context.Context, jobID string) (*models.JobProgressSummary, error)

	// UpdateJobStatus updates the status of a job
	UpdateJobStatus(ctx context.Context, jobID string, status models.JobStatus) error

	// UpdateJobProgress updates processed/succeeded/failed counts
	UpdateJobProgress(ctx context.Context, jobID string, processed, succeeded, failed int) error

	// MarkJobStarted marks job as started
	MarkJobStarted(ctx context.Context, jobID string) error

	// FailJob marks job as failed
	FailJob(ctx context.Context, jobID string, errorDetails *json.RawMessage) error

	// CancelJob marks job as cancelled
	CancelJob(ctx context.Context, jobID string) error

	// ListJobs returns a list of jobs for a tenant
	ListJobs(ctx context.Context, tenantID string, status *models.JobStatus, limit int) ([]*models.AsyncJob, error)

	// CreateJobItems creates job items in batch
	CreateJobItems(ctx context.Context, jobID string, items []*models.JobItem) error

	// GetJobItems returns items for a job
	GetJobItems(ctx context.Context, jobID string, statusFilter *models.ItemStatus) ([]*models.JobItem, error)

	// UpdateJobItem updates a job item status
	UpdateJobItem(ctx context.Context, itemID string, status models.ItemStatus, resultID *string, errorMsg string) error

	// MarkWebhookSent marks webhook as sent
	MarkWebhookSent(ctx context.Context, jobID string, success bool) error

	// GetQueueStats returns queue statistics
	GetQueueStats(ctx context.Context, tenantID string) (*QueueStats, error)
}

// PostgresJobQueue is a PostgreSQL-backed implementation of JobQueue
type PostgresJobQueue struct {
	db *sql.DB
}

// QueueStats provides statistics about the job queue
type QueueStats struct {
	QueuedCount     int
	RunningCount    int
	CompletedCount  int
	FailedCount     int
	AverageWaitTime time.Duration
	AverageDuration time.Duration
}

// NewPostgresJobQueue creates a new PostgreSQL job queue
func NewPostgresJobQueue(db *sql.DB) JobQueue {
	return &PostgresJobQueue{db: db}
}

// Enqueue adds a new job to the queue
func (q *PostgresJobQueue) Enqueue(ctx context.Context, job *models.AsyncJob) (*models.AsyncJob, error) {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	if job.Priority < 0 {
		job.Priority = 0
	}

	// Convert CreatedBy (string/user ID) to UUID
	createdByUUID := uuid.Nil
	if job.CreatedBy != "" {
		// Try to parse if it's a valid UUID, otherwise generate deterministic UUID from string
		if parsedUUID, err := uuid.Parse(job.CreatedBy); err == nil {
			createdByUUID = parsedUUID
		} else {
			// Generate deterministic UUID from user ID string
			createdByUUID = uuid.NewSHA1(uuid.NameSpaceDNS, []byte(job.CreatedBy))
		}
	}

	query := `
		INSERT INTO edm.async_jobs (
			id, tenant_id, operation_type, status, total_items, 
			payload, webhook_url, created_by, priority, max_retries
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`

	err := q.db.QueryRowContext(ctx, query,
		job.ID,
		job.TenantID,
		job.OperationType,
		models.JobStatusQueued,
		job.TotalItems,
		job.Payload,
		job.WebhookURL,
		createdByUUID,
		job.Priority,
		job.MaxRetries,
	).Scan(&job.ID, &job.CreatedAt)

	if err != nil {
		log.Printf("[JobQueue] Error enqueueing job: %v", err)
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	job.Status = models.JobStatusQueued
	log.Printf("[JobQueue] Job enqueued: %s (type: %s, items: %d)", job.ID, job.OperationType, job.TotalItems)
	return job, nil
}

// Dequeue retrieves the next jobs to process
func (q *PostgresJobQueue) Dequeue(ctx context.Context, batchSize int) ([]*models.AsyncJob, error) {
	query := `
		UPDATE edm.async_jobs
		SET status = $1
		WHERE id IN (
			SELECT id FROM edm.async_jobs
			WHERE status = $2 AND tenant_id = CAST(current_setting('app.current_tenant_id') AS UUID)
			ORDER BY priority DESC, created_at ASC
			LIMIT $3
		)
		RETURNING id, tenant_id, operation_type, status, total_items, processed_items,
		          succeeded_items, failed_items, payload, result_ids, error_details,
		          webhook_url, webhook_sent, webhook_attempts, created_by, created_at,
		          started_at, completed_at, priority, retry_count, max_retries
	`

	rows, err := q.db.QueryContext(ctx, query, models.JobStatusRunning, models.JobStatusQueued, batchSize)
	if err != nil {
		log.Printf("[JobQueue] Error dequeueing jobs: %v", err)
		return nil, fmt.Errorf("failed to dequeue jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*models.AsyncJob
	for rows.Next() {
		job := &models.AsyncJob{}
		err := rows.Scan(
			&job.ID, &job.TenantID, &job.OperationType, &job.Status,
			&job.TotalItems, &job.ProcessedItems, &job.SucceededItems, &job.FailedItems,
			&job.Payload, &job.ResultIDs, &job.ErrorDetails, &job.WebhookURL, &job.WebhookSent,
			&job.WebhookAttempts, &job.CreatedBy, &job.CreatedAt, &job.StartedAt,
			&job.CompletedAt, &job.Priority, &job.RetryCount, &job.MaxRetries,
		)
		if err != nil {
			log.Printf("[JobQueue] Error scanning job: %v", err)
			continue
		}
		jobs = append(jobs, job)
	}

	log.Printf("[JobQueue] Dequeued %d jobs", len(jobs))
	return jobs, rows.Err()
}

// GetJobStatus returns the current status of a job
func (q *PostgresJobQueue) GetJobStatus(ctx context.Context, jobID string) (*models.AsyncJob, error) {
	// Set RLS context
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the job first to know the tenant
	jobQuery := `SELECT tenant_id FROM edm.async_jobs WHERE id = $1`
	var tenantID string
	err = tx.QueryRowContext(ctx, jobQuery, jobID).Scan(&tenantID)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	// Set RLS context
	if _, err := tx.ExecContext(ctx, "SELECT set_config('app.current_tenant_id', $1, false)", tenantID); err != nil {
		return nil, fmt.Errorf("failed to set RLS context: %w", err)
	}

	query := `
		SELECT id, tenant_id, operation_type, status, total_items, processed_items,
		       succeeded_items, failed_items, payload, result_ids, error_details,
		       webhook_url, webhook_sent, webhook_attempts, created_by, created_at,
		       started_at, completed_at, priority, retry_count, max_retries
		FROM edm.async_jobs
		WHERE id = $1
	`

	job := &models.AsyncJob{}
	err = tx.QueryRowContext(ctx, query, jobID).Scan(
		&job.ID, &job.TenantID, &job.OperationType, &job.Status,
		&job.TotalItems, &job.ProcessedItems, &job.SucceededItems, &job.FailedItems,
		&job.Payload, &job.ResultIDs, &job.ErrorDetails, &job.WebhookURL, &job.WebhookSent,
		&job.WebhookAttempts, &job.CreatedBy, &job.CreatedAt, &job.StartedAt,
		&job.CompletedAt, &job.Priority, &job.RetryCount, &job.MaxRetries,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	if err != nil {
		log.Printf("[JobQueue] Error getting job status: %v", err)
		return nil, fmt.Errorf("failed to get job status: %w", err)
	}

	return job, nil
}

// GetJobProgress returns progress summary for a job
func (q *PostgresJobQueue) GetJobProgress(ctx context.Context, jobID string) (*models.JobProgressSummary, error) {
	// Set RLS context
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the job first to know the tenant
	jobQuery := `SELECT tenant_id FROM edm.async_jobs WHERE id = $1`
	var tenantID string
	err = tx.QueryRowContext(ctx, jobQuery, jobID).Scan(&tenantID)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	// Set RLS context
	if _, err := tx.ExecContext(ctx, "SELECT set_config('app.current_tenant_id', $1, false)", tenantID); err != nil {
		return nil, fmt.Errorf("failed to set RLS context: %w", err)
	}

	query := `
		SELECT id, tenant_id, operation_type, status, total_items, processed_items,
		       succeeded_items, failed_items, pending_items, processing_items, item_errors,
		       progress_percent, created_at, started_at, completed_at, duration_seconds
		FROM edm.job_progress_summary
		WHERE id = $1
	`

	progress := &models.JobProgressSummary{}
	err = tx.QueryRowContext(ctx, query, jobID).Scan(
		&progress.ID, &progress.TenantID, &progress.OperationType, &progress.Status,
		&progress.TotalItems, &progress.ProcessedItems, &progress.SucceededItems, &progress.FailedItems,
		&progress.PendingItems, &progress.ProcessingItems, &progress.ItemErrors,
		&progress.ProgressPercent, &progress.CreatedAt, &progress.StartedAt,
		&progress.CompletedAt, &progress.DurationSeconds,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	if err != nil {
		log.Printf("[JobQueue] Error getting job progress: %v", err)
		return nil, fmt.Errorf("failed to get job progress: %w", err)
	}

	return progress, nil
}

// UpdateJobStatus updates the status of a job
func (q *PostgresJobQueue) UpdateJobStatus(ctx context.Context, jobID string, status models.JobStatus) error {
	query := `
		UPDATE edm.async_jobs
		SET status = $1, 
		    completed_at = CASE WHEN $1 IN ('completed', 'failed', 'cancelled') THEN NOW() ELSE completed_at END
		WHERE id = $2 AND tenant_id = CAST(current_setting('app.current_tenant_id') AS UUID)
	`

	result, err := q.db.ExecContext(ctx, query, status, jobID)
	if err != nil {
		log.Printf("[JobQueue] Error updating job status: %v", err)
		return fmt.Errorf("failed to update job status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("job not found: %s", jobID)
	}

	log.Printf("[JobQueue] Job %s status updated to %s", jobID, status)
	return nil
}

// UpdateJobProgress updates progress counters
func (q *PostgresJobQueue) UpdateJobProgress(ctx context.Context, jobID string, processed, succeeded, failed int) error {
	query := `
		SELECT edm.update_job_progress($1, $2, $3, $4)
	`

	_, err := q.db.ExecContext(ctx, query, jobID, processed, succeeded, failed)
	if err != nil {
		log.Printf("[JobQueue] Error updating job progress: %v", err)
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	return nil
}

// MarkJobStarted marks a job as started
func (q *PostgresJobQueue) MarkJobStarted(ctx context.Context, jobID string) error {
	query := `SELECT edm.mark_job_started($1)`

	_, err := q.db.ExecContext(ctx, query, jobID)
	if err != nil {
		log.Printf("[JobQueue] Error marking job as started: %v", err)
		return fmt.Errorf("failed to mark job started: %w", err)
	}

	log.Printf("[JobQueue] Job %s marked as started", jobID)
	return nil
}

// FailJob marks a job as failed
func (q *PostgresJobQueue) FailJob(ctx context.Context, jobID string, errorDetails *json.RawMessage) error {
	query := `SELECT edm.fail_job($1, $2)`

	_, err := q.db.ExecContext(ctx, query, jobID, errorDetails)
	if err != nil {
		log.Printf("[JobQueue] Error failing job: %v", err)
		return fmt.Errorf("failed to fail job: %w", err)
	}

	log.Printf("[JobQueue] Job %s marked as failed", jobID)
	return nil
}

// CancelJob marks a job as cancelled
func (q *PostgresJobQueue) CancelJob(ctx context.Context, jobID string) error {
	query := `
		UPDATE edm.async_jobs
		SET status = $1, completed_at = NOW()
		WHERE id = $2 AND tenant_id = CAST(current_setting('app.current_tenant_id') AS UUID)
		AND status IN ('queued', 'running')
	`

	result, err := q.db.ExecContext(ctx, query, models.JobStatusCancelled, jobID)
	if err != nil {
		log.Printf("[JobQueue] Error cancelling job: %v", err)
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("job not found or not in a cancellable state: %s", jobID)
	}

	log.Printf("[JobQueue] Job %s cancelled", jobID)
	return nil
}

// ListJobs returns jobs for a tenant
func (q *PostgresJobQueue) ListJobs(ctx context.Context, tenantID string, statusFilter *models.JobStatus, limit int) ([]*models.AsyncJob, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	query := `
		SELECT id, tenant_id, operation_type, status, total_items, processed_items,
		       succeeded_items, failed_items, payload, result_ids, error_details,
		       webhook_url, webhook_sent, webhook_attempts, created_by, created_at,
		       started_at, completed_at, priority, retry_count, max_retries
		FROM edm.async_jobs
		WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}

	if statusFilter != nil {
		query += ` AND status = $2`
		args = append(args, *statusFilter)
	}

	query += ` ORDER BY created_at DESC LIMIT $` + fmt.Sprintf("%d", len(args)+1)
	args = append(args, limit)

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("[JobQueue] Error listing jobs: %v", err)
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*models.AsyncJob
	for rows.Next() {
		job := &models.AsyncJob{}
		err := rows.Scan(
			&job.ID, &job.TenantID, &job.OperationType, &job.Status,
			&job.TotalItems, &job.ProcessedItems, &job.SucceededItems, &job.FailedItems,
			&job.Payload, &job.ResultIDs, &job.ErrorDetails, &job.WebhookURL, &job.WebhookSent,
			&job.WebhookAttempts, &job.CreatedBy, &job.CreatedAt, &job.StartedAt,
			&job.CompletedAt, &job.Priority, &job.RetryCount, &job.MaxRetries,
		)
		if err != nil {
			log.Printf("[JobQueue] Error scanning job: %v", err)
			continue
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// CreateJobItems creates job items
func (q *PostgresJobQueue) CreateJobItems(ctx context.Context, jobID string, items []*models.JobItem) error {
	if len(items) == 0 {
		return nil
	}

	stmt, err := q.db.PrepareContext(ctx, `
		INSERT INTO edm.job_items (job_id, item_index, item_name, item_data, status)
		VALUES ($1, $2, $3, $4, $5)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, item := range items {
		_, err := stmt.ExecContext(ctx,
			jobID,
			item.ItemIndex,
			item.ItemName,
			item.ItemData,
			models.ItemStatusPending,
		)
		if err != nil {
			log.Printf("[JobQueue] Error creating job item: %v", err)
			return fmt.Errorf("failed to create job item: %w", err)
		}
	}

	log.Printf("[JobQueue] Created %d job items for job %s", len(items), jobID)
	return nil
}

// GetJobItems returns items for a job
func (q *PostgresJobQueue) GetJobItems(ctx context.Context, jobID string, statusFilter *models.ItemStatus) ([]*models.JobItem, error) {
	query := `
		SELECT id, job_id, item_index, item_name, item_data, status, error_message, result_id, processed_at
		FROM edm.job_items
		WHERE job_id = $1
	`

	args := []interface{}{jobID}

	if statusFilter != nil {
		query += ` AND status = $2`
		args = append(args, *statusFilter)
	}

	query += ` ORDER BY item_index ASC`

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("[JobQueue] Error getting job items: %v", err)
		return nil, fmt.Errorf("failed to get job items: %w", err)
	}
	defer rows.Close()

	var items []*models.JobItem
	for rows.Next() {
		item := &models.JobItem{}
		err := rows.Scan(
			&item.ID, &item.JobID, &item.ItemIndex, &item.ItemName,
			&item.ItemData, &item.Status, &item.ErrorMessage, &item.ResultID,
			&item.ProcessedAt,
		)
		if err != nil {
			log.Printf("[JobQueue] Error scanning job item: %v", err)
			continue
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// UpdateJobItem updates a job item
func (q *PostgresJobQueue) UpdateJobItem(ctx context.Context, itemID string, status models.ItemStatus, resultID *string, errorMsg string) error {
	query := `
		UPDATE edm.job_items
		SET status = $1, result_id = $2, error_message = $3, processed_at = NOW()
		WHERE id = $4
	`

	_, err := q.db.ExecContext(ctx, query, status, resultID, errorMsg, itemID)
	if err != nil {
		log.Printf("[JobQueue] Error updating job item: %v", err)
		return fmt.Errorf("failed to update job item: %w", err)
	}

	return nil
}

// MarkWebhookSent marks webhook as sent
func (q *PostgresJobQueue) MarkWebhookSent(ctx context.Context, jobID string, success bool) error {
	query := `
		UPDATE edm.async_jobs
		SET webhook_sent = $1, webhook_attempts = webhook_attempts + 1
		WHERE id = $2
	`

	_, err := q.db.ExecContext(ctx, query, success, jobID)
	if err != nil {
		log.Printf("[JobQueue] Error marking webhook sent: %v", err)
		return fmt.Errorf("failed to mark webhook sent: %w", err)
	}

	return nil
}

// GetQueueStats returns stats about the queue
func (q *PostgresJobQueue) GetQueueStats(ctx context.Context, tenantID string) (*QueueStats, error) {
	query := `
		SELECT
		  COUNT(CASE WHEN status = 'queued' THEN 1 END) as queued_count,
		  COUNT(CASE WHEN status = 'running' THEN 1 END) as running_count,
		  COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_count,
		  COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_count,
		  EXTRACT(EPOCH FROM AVG(EXTRACT(EPOCH FROM (started_at - created_at))))::BIGINT as avg_wait,
		  EXTRACT(EPOCH FROM AVG(EXTRACT(EPOCH FROM (completed_at - started_at))))::BIGINT as avg_duration
		FROM edm.async_jobs
		WHERE tenant_id = $1
	`

	stats := &QueueStats{}
	var avgWait, avgDuration *int64

	err := q.db.QueryRowContext(ctx, query, tenantID).Scan(
		&stats.QueuedCount,
		&stats.RunningCount,
		&stats.CompletedCount,
		&stats.FailedCount,
		&avgWait,
		&avgDuration,
	)

	if err != nil {
		log.Printf("[JobQueue] Error getting queue stats: %v", err)
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}

	if avgWait != nil {
		stats.AverageWaitTime = time.Duration(*avgWait) * time.Second
	}
	if avgDuration != nil {
		stats.AverageDuration = time.Duration(*avgDuration) * time.Second
	}

	return stats, nil
}
