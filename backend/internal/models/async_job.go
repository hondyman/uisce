package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

// JobStatus represents the status of an async job
type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// OperationType represents the type of async operation
type OperationType string

const (
	OperationBulkCreate  OperationType = "bulk-create"
	OperationBulkPublish OperationType = "bulk-publish"
	OperationBulkPromote OperationType = "bulk-promote"
)

// ItemStatus represents the status of an individual job item
type ItemStatus string

const (
	ItemStatusPending    ItemStatus = "pending"
	ItemStatusProcessing ItemStatus = "processing"
	ItemStatusSucceeded  ItemStatus = "succeeded"
	ItemStatusFailed     ItemStatus = "failed"
	ItemStatusSkipped    ItemStatus = "skipped"
)

// AsyncJob represents an asynchronous bulk operation job
type AsyncJob struct {
	ID              string           `json:"id"`
	TenantID        string           `json:"tenantId"`
	OperationType   OperationType    `json:"operationType"`
	Status          JobStatus        `json:"status"`
	TotalItems      int              `json:"totalItems"`
	ProcessedItems  int              `json:"processedItems"`
	SucceededItems  int              `json:"succeededItems"`
	FailedItems     int              `json:"failedItems"`
	Payload         json.RawMessage  `json:"payload"`
	ResultIDs       pq.StringArray   `json:"resultIds"`
	ErrorDetails    *json.RawMessage `json:"errorDetails,omitempty"`
	WebhookURL      string           `json:"webhookUrl,omitempty"`
	WebhookSent     bool             `json:"webhookSent"`
	WebhookAttempts int              `json:"webhookAttempts"`
	CreatedBy       string           `json:"createdBy"`
	CreatedAt       time.Time        `json:"createdAt"`
	StartedAt       *time.Time       `json:"startedAt,omitempty"`
	CompletedAt     *time.Time       `json:"completedAt,omitempty"`
	Priority        int              `json:"priority"`
	RetryCount      int              `json:"retryCount"`
	MaxRetries      int              `json:"maxRetries"`
}

// JobItem represents a single item in a bulk operation
type JobItem struct {
	ID           string          `json:"id"`
	JobID        string          `json:"jobId"`
	ItemIndex    int             `json:"itemIndex"`
	ItemName     string          `json:"itemName"`
	ItemData     json.RawMessage `json:"itemData"`
	Status       ItemStatus      `json:"status"`
	ErrorMessage string          `json:"errorMessage,omitempty"`
	ResultID     *string         `json:"resultId,omitempty"`
	ProcessedAt  *time.Time      `json:"processedAt,omitempty"`
}

// JobProgressSummary provides aggregated job progress information
type JobProgressSummary struct {
	ID              string        `json:"id"`
	TenantID        string        `json:"tenantId"`
	OperationType   OperationType `json:"operationType"`
	Status          JobStatus     `json:"status"`
	TotalItems      int           `json:"totalItems"`
	ProcessedItems  int           `json:"processedItems"`
	SucceededItems  int           `json:"succeededItems"`
	FailedItems     int           `json:"failedItems"`
	PendingItems    int           `json:"pendingItems"`
	ProcessingItems int           `json:"processingItems"`
	ItemErrors      int           `json:"itemErrors"`
	ProgressPercent int           `json:"progressPercent"`
	CreatedAt       time.Time     `json:"createdAt"`
	StartedAt       *time.Time    `json:"startedAt,omitempty"`
	CompletedAt     *time.Time    `json:"completedAt,omitempty"`
	DurationSeconds int           `json:"durationSeconds"`
}

// CreateAsyncJobRequest is the request body for creating an async job
type CreateAsyncJobRequest struct {
	OperationType string          `json:"operationType" binding:"required"`
	Items         json.RawMessage `json:"items" binding:"required"`
	WebhookURL    string          `json:"webhookUrl,omitempty"`
	Priority      int             `json:"priority,omitempty"`
}

// AsyncJobResponse is returned when job is created
type AsyncJobResponse struct {
	JobID         string    `json:"jobId"`
	Status        JobStatus `json:"status"`
	StatusURL     string    `json:"statusUrl"`
	EstimatedTime string    `json:"estimatedTime,omitempty"`
	OperationType string    `json:"operationType"`
	TotalItems    int       `json:"totalItems"`
	Message       string    `json:"message,omitempty"`
}

// JobStatusResponse is returned when querying job status
type JobStatusResponse struct {
	JobID               string        `json:"jobId"`
	OperationType       OperationType `json:"operationType"`
	Status              JobStatus     `json:"status"`
	Progress            JobProgress   `json:"progress"`
	StartedAt           *time.Time    `json:"startedAt,omitempty"`
	EstimatedCompletion *time.Time    `json:"estimatedCompletion,omitempty"`
	CompletedAt         *time.Time    `json:"completedAt,omitempty"`
	Results             JobResults    `json:"results,omitempty"`
}

// JobProgress tracks progress of a job
type JobProgress struct {
	Total      int `json:"total"`
	Processed  int `json:"processed"`
	Succeeded  int `json:"succeeded"`
	Failed     int `json:"failed"`
	Percentage int `json:"percentage"`
}

// JobResults contains the results of completed job
type JobResults struct {
	SuccessCount int             `json:"successCount"`
	FailureCount int             `json:"failureCount"`
	SkippedCount int             `json:"skippedCount"`
	SuccessIDs   []string        `json:"successIds,omitempty"`
	FailedItems  []FailedJobItem `json:"failedItems,omitempty"`
	ErrorSummary map[string]int  `json:"errorSummary,omitempty"`
}

// FailedJobItem represents a failed item in job results
type FailedJobItem struct {
	Index   int    `json:"index"`
	Name    string `json:"name"`
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// JobListResponse is returned when listing jobs
type JobListResponse struct {
	Jobs           []JobListItem `json:"jobs"`
	TotalCount     int           `json:"totalCount"`
	CompletedCount int           `json:"completedCount"`
	FailedCount    int           `json:"failedCount"`
	RunningCount   int           `json:"runningCount"`
}

// JobListItem is a summary of a job for list response
type JobListItem struct {
	JobID         string        `json:"jobId"`
	OperationType OperationType `json:"operationType"`
	Status        JobStatus     `json:"status"`
	Progress      JobProgress   `json:"progress"`
	StartedAt     *time.Time    `json:"startedAt,omitempty"`
	CompletedAt   *time.Time    `json:"completedAt,omitempty"`
	CreatedAt     time.Time     `json:"createdAt"`
}

// JobCancelResponse is returned when cancelling a job
type JobCancelResponse struct {
	JobID          string    `json:"jobId"`
	Status         JobStatus `json:"status"`
	ProcessedItems int       `json:"processedItems"`
	Message        string    `json:"message"`
}

// JobWebhookPayload is sent to webhook URL on job completion
type JobWebhookPayload struct {
	Event           string        `json:"event"`
	JobID           string        `json:"jobId"`
	OperationType   OperationType `json:"operationType"`
	Status          JobStatus     `json:"status"`
	TotalItems      int           `json:"totalItems"`
	SucceededItems  int           `json:"succeededItems"`
	FailedItems     int           `json:"failedItems"`
	CompletionTime  time.Time     `json:"completionTime"`
	DurationSeconds int           `json:"durationSeconds"`
	Results         *JobResults   `json:"results,omitempty"`
	ErrorDetails    string        `json:"errorDetails,omitempty"`
	StatusURL       string        `json:"statusUrl"`
}

// Value implements the driver.Valuer interface for JobStatus
func (js JobStatus) Value() (driver.Value, error) {
	return string(js), nil
}

// Scan implements the sql.Scanner interface for JobStatus
func (js *JobStatus) Scan(value interface{}) error {
	if value == nil {
		*js = ""
		return nil
	}
	*js = JobStatus(value.(string))
	return nil
}

// Value implements the driver.Valuer interface for OperationType
func (ot OperationType) Value() (driver.Value, error) {
	return string(ot), nil
}

// Scan implements the sql.Scanner interface for OperationType
func (ot *OperationType) Scan(value interface{}) error {
	if value == nil {
		*ot = ""
		return nil
	}
	*ot = OperationType(value.(string))
	return nil
}

// Value implements the driver.Valuer interface for ItemStatus
func (is ItemStatus) Value() (driver.Value, error) {
	return string(is), nil
}

// Scan implements the sql.Scanner interface for ItemStatus
func (is *ItemStatus) Scan(value interface{}) error {
	if value == nil {
		*is = ""
		return nil
	}
	*is = ItemStatus(value.(string))
	return nil
}
