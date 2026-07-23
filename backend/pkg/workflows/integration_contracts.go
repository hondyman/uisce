package workflows

// ============================================================================
// Unified Integration Microservice Contract
// ============================================================================

// IntegrationSystem represents supported external systems
type IntegrationSystem string

const (
	SystemSalesforce IntegrationSystem = "Salesforce"
	SystemServiceNow IntegrationSystem = "ServiceNow"
	SystemJira       IntegrationSystem = "Jira"
)

// IntegrationAction represents normalized actions across systems
type IntegrationAction string

const (
	ActionCreateCase     IntegrationAction = "create_case"     // Salesforce Case, ServiceNow Incident, Jira Issue
	ActionCreateIncident IntegrationAction = "create_incident" // Specific to ServiceNow
	ActionCreateIssue    IntegrationAction = "create_issue"    // Specific to Jira
	ActionUpdate         IntegrationAction = "update"
	ActionAddComment     IntegrationAction = "add_comment"
	ActionClose          IntegrationAction = "close"
)

// ExternalTaskRequest matches the Unified API Contract (POST /external-tasks)
type ExternalTaskRequest struct {
	System   IntegrationSystem    `json:"system"`
	Action   IntegrationAction    `json:"action"`
	Payload  ExternalTaskPayload  `json:"payload"`
	Callback ExternalTaskCallback `json:"callback,omitempty"`
}

// ExternalTaskPayload captures the business data for the task
type ExternalTaskPayload struct {
	ClientID string                 `json:"client_id,omitempty"`
	Summary  string                 `json:"summary"`
	Details  map[string]interface{} `json:"details,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Priority, assignment group, etc.
}

// ExternalTaskCallback defines where to report back results
type ExternalTaskCallback struct {
	WorkflowID string `json:"workflow_id"`
	StepID     string `json:"step_id"`
}

// ExternalTaskResponse represents the immediate API response
type ExternalTaskResponse struct {
	ExternalID string `json:"external_id"`
	Status     string `json:"status"` // "created", "pending", "failed"
}

// ReportRequest defines the contract for the Report Generation Service
type ReportRequest struct {
	ReportType string                 `json:"report_type"` // e.g., "ModelChangeSummary"
	Format     string                 `json:"format"`      // "pdf", "html"
	Context    map[string]interface{} `json:"context"`     // client, proposal, trades, etc.
}

// ReportResponse defines the output of a report generation
type ReportResponse struct {
	ReportID  string `json:"report_id"`
	ReportURL string `json:"url"`
}
