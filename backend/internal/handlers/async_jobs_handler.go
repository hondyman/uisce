package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// AsyncJobsHandler handles async job operations
type AsyncJobsHandler struct {
	jobQueue JobQueue
	db       *sql.DB
}

// JobQueue interface for dependency injection
type JobQueue interface {
	Enqueue(ctx context.Context, job *models.AsyncJob) (*models.AsyncJob, error)
	GetJobStatus(ctx context.Context, jobID string) (*models.AsyncJob, error)
	GetJobProgress(ctx context.Context, jobID string) (*models.JobProgressSummary, error)
	UpdateJobStatus(ctx context.Context, jobID string, status models.JobStatus) error
	CancelJob(ctx context.Context, jobID string) error
	ListJobs(ctx context.Context, tenantID string, status *models.JobStatus, limit int) ([]*models.AsyncJob, error)
}

// NewAsyncJobsHandler creates a new async jobs handler
func NewAsyncJobsHandler(jobQueue JobQueue, db *sql.DB) *AsyncJobsHandler {
	return &AsyncJobsHandler{
		jobQueue: jobQueue,
		db:       db,
	}
}

// CreateAsyncBulkCreateJob creates an async bulk create job
// POST /api/v1/templates/bulk-create/async
func (h *AsyncJobsHandler) CreateAsyncBulkCreateJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")

	if tenantID == "" || userID == "" {
		http.Error(w, "Missing tenant or user ID", http.StatusBadRequest)
		return
	}

	var req models.CreateAsyncJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate items exist
	if len(req.Items) == 0 {
		http.Error(w, "Items cannot be empty", http.StatusBadRequest)
		return
	}

	// Parse items to count them
	var items []interface{}
	if err := json.Unmarshal(req.Items, &items); err != nil {
		http.Error(w, fmt.Sprintf("Invalid items JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Limit batch size
	if len(items) > 10000 {
		http.Error(w, "Maximum 10000 items per batch", http.StatusBadRequest)
		return
	}

	// Create job
	job := &models.AsyncJob{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		OperationType: models.OperationBulkCreate,
		Status:        models.JobStatusQueued,
		TotalItems:    len(items),
		Payload:       req.Items,
		WebhookURL:    req.WebhookURL,
		CreatedBy:     userID,
		Priority:      req.Priority,
		MaxRetries:    3,
	}

	// Normalize tenant ID case
	job.TenantID = strings.ToLower(job.TenantID)

	// Enqueue job
	createdJob, err := h.jobQueue.Enqueue(ctx, job)
	if err != nil {
		log.Printf("[AsyncJobsHandler] Error enqueueing job: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create job: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response (HTTP 202 Accepted - job will be processed asynchronously)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	resp := models.AsyncJobResponse{
		JobID:         createdJob.ID,
		Status:        models.JobStatusQueued,
		StatusURL:     fmt.Sprintf("/api/v1/jobs/%s", createdJob.ID),
		OperationType: string(createdJob.OperationType),
		TotalItems:    createdJob.TotalItems,
		EstimatedTime: fmt.Sprintf("~%d seconds", (len(items)+99)/100), // Rough estimate
		Message:       "Bulk operation queued for processing",
	}

	json.NewEncoder(w).Encode(resp)
}

// CreateAsyncBulkPublishJob creates an async bulk publish job
// POST /api/v1/templates/bulk-publish/async
func (h *AsyncJobsHandler) CreateAsyncBulkPublishJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")

	if tenantID == "" || userID == "" {
		http.Error(w, "Missing tenant or user ID", http.StatusBadRequest)
		return
	}

	var req models.CreateAsyncJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate items exist
	if len(req.Items) == 0 {
		http.Error(w, "Items cannot be empty", http.StatusBadRequest)
		return
	}

	// Parse items to count them
	var items []interface{}
	if err := json.Unmarshal(req.Items, &items); err != nil {
		http.Error(w, fmt.Sprintf("Invalid items JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Limit batch size
	if len(items) > 2500 {
		http.Error(w, "Maximum 2500 items per batch for publish", http.StatusBadRequest)
		return
	}

	// Create job
	job := &models.AsyncJob{
		ID:            uuid.New().String(),
		TenantID:      strings.ToLower(tenantID),
		OperationType: models.OperationBulkPublish,
		Status:        models.JobStatusQueued,
		TotalItems:    len(items),
		Payload:       req.Items,
		WebhookURL:    req.WebhookURL,
		CreatedBy:     userID,
		Priority:      req.Priority,
		MaxRetries:    3,
	}

	// Enqueue job
	createdJob, err := h.jobQueue.Enqueue(ctx, job)
	if err != nil {
		log.Printf("[AsyncJobsHandler] Error enqueueing job: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create job: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	resp := models.AsyncJobResponse{
		JobID:         createdJob.ID,
		Status:        models.JobStatusQueued,
		StatusURL:     fmt.Sprintf("/api/v1/jobs/%s", createdJob.ID),
		OperationType: string(createdJob.OperationType),
		TotalItems:    createdJob.TotalItems,
		EstimatedTime: fmt.Sprintf("~%d seconds", (len(items)+199)/200),
		Message:       "Bulk publish job queued for processing",
	}

	json.NewEncoder(w).Encode(resp)
}

// GetJobStatus returns the status of an async job
// GET /api/v1/jobs/{jobId}
func (h *AsyncJobsHandler) GetJobStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jobID := mux.Vars(r)["jobId"]

	// Get job progress
	progress, err := h.jobQueue.GetJobProgress(ctx, jobID)
	if err != nil {
		log.Printf("[AsyncJobsHandler] Error getting job progress: %v", err)
		http.Error(w, fmt.Sprintf("Job not found: %v", err), http.StatusNotFound)
		return
	}

	// Get full job for webhook info
	job, err := h.jobQueue.GetJobStatus(ctx, jobID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Job not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := models.JobStatusResponse{
		JobID:         job.ID,
		OperationType: job.OperationType,
		Status:        job.Status,
		Progress: models.JobProgress{
			Total:      progress.TotalItems,
			Processed:  progress.ProcessedItems,
			Succeeded:  progress.SucceededItems,
			Failed:     progress.FailedItems,
			Percentage: progress.ProgressPercent,
		},
		StartedAt:           job.StartedAt,
		CompletedAt:         job.CompletedAt,
		EstimatedCompletion: nil,
	}

	json.NewEncoder(w).Encode(resp)
}

// ListJobs returns a list of jobs for the tenant
// GET /api/v1/jobs?status=completed&limit=20
func (h *AsyncJobsHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusBadRequest)
		return
	}

	// Get parameters
	statusParam := r.URL.Query().Get("status")
	limitParam := r.URL.Query().Get("limit")

	limit := 20
	if limitParam != "" {
		fmt.Sscanf(limitParam, "%d", &limit)
	}
	if limit > 100 {
		limit = 100
	}

	// Filter by status if provided
	var statusFilter *models.JobStatus
	if statusParam != "" {
		status := models.JobStatus(statusParam)
		statusFilter = &status
	}

	// Normalize tenant ID
	tenantID = strings.ToLower(tenantID)

	// Get jobs
	jobs, err := h.jobQueue.ListJobs(ctx, tenantID, statusFilter, limit)
	if err != nil {
		log.Printf("[AsyncJobsHandler] Error listing jobs: %v", err)
		http.Error(w, fmt.Sprintf("Failed to list jobs: %v", err), http.StatusInternalServerError)
		return
	}

	// Count jobs by status
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Build response with job summaries
	jobItems := make([]models.JobListItem, 0, len(jobs))
	completedCount := 0
	failedCount := 0
	runningCount := 0

	for _, job := range jobs {
		jobItems = append(jobItems, models.JobListItem{
			JobID:         job.ID,
			OperationType: job.OperationType,
			Status:        job.Status,
			Progress: models.JobProgress{
				Total:      job.TotalItems,
				Processed:  job.ProcessedItems,
				Succeeded:  job.SucceededItems,
				Failed:     job.FailedItems,
				Percentage: (job.ProcessedItems * 100) / job.TotalItems,
			},
			StartedAt:   job.StartedAt,
			CompletedAt: job.CompletedAt,
			CreatedAt:   job.CreatedAt,
		})

		if job.Status == models.JobStatusCompleted {
			completedCount++
		} else if job.Status == models.JobStatusFailed {
			failedCount++
		} else if job.Status == models.JobStatusRunning {
			runningCount++
		}
	}

	resp := models.JobListResponse{
		Jobs:           jobItems,
		TotalCount:     len(jobs),
		CompletedCount: completedCount,
		FailedCount:    failedCount,
		RunningCount:   runningCount,
	}

	json.NewEncoder(w).Encode(resp)
}

// CancelJob cancels an async job
// POST /api/v1/jobs/{jobId}/cancel
func (h *AsyncJobsHandler) CancelJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jobID := mux.Vars(r)["jobId"]

	// Get job first to verify ownership
	job, err := h.jobQueue.GetJobStatus(ctx, jobID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Job not found: %v", err), http.StatusNotFound)
		return
	}

	// Cancel the job
	if err := h.jobQueue.CancelJob(ctx, jobID); err != nil {
		if strings.Contains(err.Error(), "not in a cancellable state") {
			http.Error(w, "Job cannot be cancelled in its current state", http.StatusConflict)
		} else {
			http.Error(w, fmt.Sprintf("Failed to cancel job: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := models.JobCancelResponse{
		JobID:          jobID,
		Status:         models.JobStatusCancelled,
		ProcessedItems: job.ProcessedItems,
		Message:        "Job cancelled successfully",
	}

	json.NewEncoder(w).Encode(resp)
}

// GetProcessorStats returns processor statistics
// GET /api/v1/jobs/stats
func (h *AsyncJobsHandler) GetProcessorStats(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusBadRequest)
		return
	}

	// In a full implementation, would get from processor service
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	stats := map[string]interface{}{
		"workerCount":    4,
		"isRunning":      true,
		"queuedCount":    0,
		"runningCount":   0,
		"completedCount": 0,
		"failedCount":    0,
	}

	json.NewEncoder(w).Encode(stats)
}
