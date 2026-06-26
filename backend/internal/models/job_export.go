package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// JobExport represents an export job
type JobExport struct {
	ID                    uuid.UUID                  `json:"id"`
	JobID                 uuid.UUID                  `json:"job_id"`
	TenantID              uuid.UUID                  `json:"tenant_id"`
	ExportFormat          string                     `json:"export_format"`
	Status                string                     `json:"status"`
	FileLocation          string                     `json:"file_location,omitempty"`
	FileSize              int64                      `json:"file_size"`
	RecordCount           int                        `json:"record_count"`
	PresignedURL          string                     `json:"presigned_url,omitempty"`
	PresignedURLExpires   *time.Time                 `json:"presigned_url_expires,omitempty"`
	DownloadCount         int                        `json:"download_count"`
	FilterCriteria        map[string]interface{}     `json:"filter_criteria,omitempty"`
	IncludeErrors         bool                       `json:"include_errors"`
	CreatedBy             uuid.UUID                  `json:"created_by"`
	CreatedAt             time.Time                  `json:"created_at"`
	StartedAt             *time.Time                 `json:"started_at,omitempty"`
	CompletedAt           *time.Time                 `json:"completed_at,omitempty"`
	ExpiresAt             *time.Time                 `json:"expires_at,omitempty"`
	ErrorMessage          string                     `json:"error_message,omitempty"`
}

// CreateExportRequest is the request to create an export
type CreateExportRequest struct {
	ExportFormat   string                 `json:"export_format" binding:"required,oneof=csv json parquet"`
	FilterCriteria map[string]interface{} `json:"filter_criteria,omitempty"`
	IncludeErrors  bool                   `json:"include_errors"`
}

// ExportStatusResponse is the response for export status
type ExportStatusResponse struct {
	ID                  uuid.UUID `json:"id"`
	JobID               uuid.UUID `json:"job_id"`
	Status              string    `json:"status"`
	ExportFormat        string    `json:"export_format"`
	FileSize            int64     `json:"file_size"`
	RecordCount         int       `json:"record_count"`
	CreatedAt           time.Time `json:"created_at"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	ExpiresAt           *time.Time `json:"expires_at,omitempty"`
	PresignedURL        string    `json:"presigned_url,omitempty"`
	PresignedURLExpires *time.Time `json:"presigned_url_expires,omitempty"`
	IsDownloadable      bool      `json:"is_downloadable"`
	ErrorMessage        string    `json:"error_message,omitempty"`
}

// DownloadURLRequest is the request for a download URL
type DownloadURLRequest struct {
	ExpiryHours int `json:"expiry_hours" binding:"required,min=1,max=720"`
}

// DownloadURLResponse is the response with download URL
type DownloadURLResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
	Format    string    `json:"format"`
}

// ListExportsResponse lists exports for a job
type ListExportsResponse struct {
	Exports []*ExportSummary `json:"exports"`
	Total   int              `json:"total"`
}

// ExportSummary is a summary of an export
type ExportSummary struct {
	ID       uuid.UUID `json:"id"`
	Format   string    `json:"format"`
	Status   string    `json:"status"`
	FileSize int64     `json:"file_size"`
	Records  int       `json:"record_count"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// JobExportJSON is for database JSON storage
type JobExportJSON struct {
	ID          uuid.UUID `json:"id"`
	Status      string    `json:"status"`
	FileSize    int64     `json:"file_size"`
	RecordCount int       `json:"record_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// Scan implements sql.Scanner interface
func (j *JobExportJSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion failed")
	}
	return json.Unmarshal(bytes, j)
}

// Value implements driver.Valuer interface
func (j JobExportJSON) Value() (driver.Value, error) {
	return json.Marshal(j)
}
