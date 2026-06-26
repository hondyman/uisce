package reports

import (
	"time"

	"github.com/google/uuid"
)

// ReportTemplate represents a report template configuration
type ReportTemplate struct {
	ID              uuid.UUID              `json:"id"`
	TenantID        uuid.UUID              `json:"tenant_id"`
	TemplateName    string                 `json:"template_name"`
	Description     string                 `json:"description"`
	Category        string                 `json:"category"`
	SemanticViewIDs []uuid.UUID            `json:"semantic_view_ids"`
	LayoutConfig    map[string]interface{} `json:"layout_config"`
	ParameterSchema map[string]interface{} `json:"parameter_schema"`
	IsActive        bool                   `json:"is_active"`
	IsPublic        bool                   `json:"is_public"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	CreatedBy       string                 `json:"created_by,omitempty"`
	Version         int                    `json:"version"`
}

// ReportExecution represents a report generation job
type ReportExecution struct {
	ID              uuid.UUID              `json:"id"`
	TenantID        uuid.UUID              `json:"tenant_id"`
	TemplateID      uuid.UUID              `json:"template_id"`
	HouseholdID     *uuid.UUID             `json:"household_id,omitempty"`
	Parameters      map[string]interface{} `json:"parameters"`
	Status          string                 `json:"status"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	OutputURL       string                 `json:"output_url,omitempty"`
	OutputSizeBytes int                    `json:"output_size_bytes,omitempty"`
	ExecutionTimeMS int                    `json:"execution_time_ms,omitempty"`
	RowsProcessed   int                    `json:"rows_processed,omitempty"`
	WorkflowID      string                 `json:"workflow_id,omitempty"`
	RunID           string                 `json:"run_id,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	CreatedBy       string                 `json:"created_by,omitempty"`
}
