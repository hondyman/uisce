package models

import (
	"time"
)

// AuditEventType represents the type of audit event
type AuditEventType string

const (
	// Authentication events
	EventLogin         AuditEventType = "login"
	EventLogout        AuditEventType = "logout"
	EventLoginFailed   AuditEventType = "login_failed"
	EventPasswordReset AuditEventType = "password_reset"

	// Data access events
	EventDataAccess AuditEventType = "data_access"
	EventDataExport AuditEventType = "data_export"
	EventDataModify AuditEventType = "data_modify"
	EventDataDelete AuditEventType = "data_delete"

	// Configuration events
	EventConfigChange AuditEventType = "config_change"
	EventBundleCreate AuditEventType = "bundle_create"
	EventBundleUpdate AuditEventType = "bundle_update"
	EventBundleDelete AuditEventType = "bundle_delete"

	// Calculation events
	EventCalculationRun AuditEventType = "calculation_run"
	EventModelExecute   AuditEventType = "model_execute"

	// System events
	EventSystemStart    AuditEventType = "system_start"
	EventSystemStop     AuditEventType = "system_stop"
	EventBackupStart    AuditEventType = "backup_start"
	EventBackupComplete AuditEventType = "backup_complete"

	// Compliance events
	EventComplianceCheck AuditEventType = "compliance_check"
	EventPolicyViolation AuditEventType = "policy_violation"
	EventAccessDenied    AuditEventType = "access_denied"
)

// AuditEventSeverity represents the severity level of an audit event
type AuditEventSeverity string

const (
	SeverityLow      AuditEventSeverity = "low"
	SeverityMedium   AuditEventSeverity = "medium"
	SeverityHigh     AuditEventSeverity = "high"
	SeverityCritical AuditEventSeverity = "critical"
)

// AuditEvent represents an audit trail entry
type AuditEvent struct {
	ID              string                 `json:"id" db:"id"`
	Timestamp       time.Time              `json:"timestamp" db:"timestamp"`
	EventType       AuditEventType         `json:"event_type" db:"event_type"`
	Severity        AuditEventSeverity     `json:"severity" db:"severity"`
	UserID          string                 `json:"user_id" db:"user_id"`
	TenantID        string                 `json:"tenant_id" db:"tenant_id"`
	SessionID       string                 `json:"session_id" db:"session_id"`
	ResourceID      string                 `json:"resource_id" db:"resource_id"`
	ResourceType    string                 `json:"resource_type" db:"resource_type"`
	Action          string                 `json:"action" db:"action"`
	IPAddress       string                 `json:"ip_address" db:"ip_address"`
	UserAgent       string                 `json:"user_agent" db:"user_agent"`
	RequestID       string                 `json:"request_id" db:"request_id"`
	Details         map[string]interface{} `json:"details" db:"details"`
	OldValues       map[string]interface{} `json:"old_values,omitempty" db:"old_values"`
	NewValues       map[string]interface{} `json:"new_values,omitempty" db:"new_values"`
	Success         bool                   `json:"success" db:"success"`
	ErrorMessage    string                 `json:"error_message,omitempty" db:"error_message"`
	ComplianceFlags []string               `json:"compliance_flags,omitempty" db:"compliance_flags"`
}

// AuditEventFilter represents filters for querying audit events
type AuditEventFilter struct {
	UserID       *string             `json:"user_id,omitempty"`
	TenantID     *string             `json:"tenant_id,omitempty"`
	EventType    *AuditEventType     `json:"event_type,omitempty"`
	Severity     *AuditEventSeverity `json:"severity,omitempty"`
	ResourceType *string             `json:"resource_type,omitempty"`
	ResourceID   *string             `json:"resource_id,omitempty"`
	StartTime    *time.Time          `json:"start_time,omitempty"`
	EndTime      *time.Time          `json:"end_time,omitempty"`
	IPAddress    *string             `json:"ip_address,omitempty"`
	Success      *bool               `json:"success,omitempty"`
	Limit        int                 `json:"limit,omitempty"`
	Offset       int                 `json:"offset,omitempty"`
}

// AuditSummary represents a summary of audit events for reporting
type AuditSummary struct {
	TotalEvents      int64                        `json:"total_events"`
	EventsByType     map[AuditEventType]int64     `json:"events_by_type"`
	EventsBySeverity map[AuditEventSeverity]int64 `json:"events_by_severity"`
	EventsByUser     map[string]int64             `json:"events_by_user"`
	RecentEvents     []AuditEvent                 `json:"recent_events"`
	TimeRange        AuditTimeRange               `json:"time_range"`
}

// AuditTimeRange represents a time range for audit queries
type AuditTimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ComplianceReport represents a compliance report
type ComplianceReport struct {
	ID              string         `json:"id"`
	ReportType      string         `json:"report_type"`
	TimeRange       AuditTimeRange `json:"time_range"`
	GeneratedAt     time.Time      `json:"generated_at"`
	GeneratedBy     string         `json:"generated_by"`
	Summary         AuditSummary   `json:"summary"`
	Violations      []AuditEvent   `json:"violations"`
	Recommendations []string       `json:"recommendations"`
	Status          string         `json:"status"`
}

// AuditRetentionPolicy represents data retention policies for audit logs
type AuditRetentionPolicy struct {
	EventType     AuditEventType `json:"event_type"`
	RetentionDays int            `json:"retention_days"`
	ArchiveAfter  int            `json:"archive_after_days"`
	DeleteAfter   int            `json:"delete_after_days"`
}

// AuditAlert represents an alert configuration for audit events
type AuditAlert struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	EventType   AuditEventType         `json:"event_type"`
	Severity    AuditEventSeverity     `json:"severity"`
	Conditions  map[string]interface{} `json:"conditions"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}
