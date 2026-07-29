package contracts

import (
	"encoding/json"
	"time"
)

// ColumnChangeKind classifies a single column change
type ColumnChangeKind string

const (
	ColumnAdded             ColumnChangeKind = "ADDED"
	ColumnDropped           ColumnChangeKind = "DROPPED"
	ColumnRenamed           ColumnChangeKind = "RENAMED"
	ColumnTypeChanged       ColumnChangeKind = "TYPE_CHANGED"
	ColumnNullabilityChanged ColumnChangeKind = "NULLABILITY_CHANGED"
	ColumnDefaultChanged    ColumnChangeKind = "DEFAULT_CHANGED"
)

// BreakingChangeSeverity classifies a change as safe or blocking
type BreakingChangeSeverity string

const (
	SeveritySafe     BreakingChangeSeverity = "SAFE"
	SeverityCritical BreakingChangeSeverity = "CRITICAL"
)

// ViolationType categorizes what contract rule was broken
type ViolationType string

const (
	ViolationRequiredFieldDropped    ViolationType = "REQUIRED_FIELD_DROPPED"
	ViolationBusinessKeyAltered      ViolationType = "BUSINESS_KEY_ALTERED"
	ViolationSemanticIDDropped       ViolationType = "SEMANTIC_ID_DROPPED"
	ViolationTypeIncompatible        ViolationType = "TYPE_INCOMPATIBLE"
	ViolationPrimaryKeyAltered       ViolationType = "PRIMARY_KEY_ALTERED"
	ViolationNonNullableToNullable   ViolationType = "NON_NULLABLE_TO_NULLABLE"
	ViolationRequiredColumnAdded     ViolationType = "REQUIRED_COLUMN_ADDED"
)

// Violation describes a single broken contract rule
type Violation struct {
	Type        ViolationType          `json:"type"`
	Severity    BreakingChangeSeverity `json:"severity"`
	Column      string                 `json:"column,omitempty"`
	OldValue    interface{}            `json:"old_value,omitempty"`
	NewValue    interface{}            `json:"new_value,omitempty"`
	Description string                 `json:"description"`
}

// ColumnDiff describes a single column change in a proposed DDL diff
type ColumnDiff struct {
	ColumnName string           `json:"column_name"`
	ChangeKind ColumnChangeKind `json:"change_kind"`
	OldType   string           `json:"old_type,omitempty"`
	NewType   string           `json:"new_type,omitempty"`
	OldNull   *bool            `json:"old_nullable,omitempty"`
	NewNull   *bool            `json:"new_nullable,omitempty"`
	OldDefault string          `json:"old_default,omitempty"`
	NewDefault string          `json:"new_default,omitempty"`
}

// TableDiff describes the full proposed change for one table
type TableDiff struct {
	TableName    string        `json:"table_name"`
	DatasourceID string        `json:"datasource_id"`
	Columns      []ColumnDiff `json:"columns"`
}

// ContractValidationRequest is the input to Validate()
type ContractValidationRequest struct {
	TenantID     string       `json:"tenant_id"`
	DatasourceID string       `json:"datasource_id"`
	ProposedDiffs []TableDiff `json:"proposed_diffs"`
	Actor        string       `json:"actor,omitempty"`
}

// ValidationResult holds the outcome for a single TableDiff
type ValidationResult struct {
	TableName    string        `json:"table_name"`
	Severity    BreakingChangeSeverity `json:"severity"`
	Violations  []Violation   `json:"violations"`
	SafeToApply bool          `json:"safe_to_apply"`
}

// ContractValidationResponse is the output of Validate()
type ContractValidationResponse struct {
	RequestID         string            `json:"request_id"`
	TenantID          string            `json:"tenant_id"`
	Results           []ValidationResult `json:"results"`
	HasCritical       bool              `json:"has_critical"`
	AllSafe           bool              `json:"all_safe"`
	ViolationsCount   int               `json:"violations_count"`
	SafeCount         int               `json:"safe_count"`
	DownstreamBOs     []string          `json:"downstream_business_objects,omitempty"`
	TicketID          string            `json:"ticket_id,omitempty"`
	EvaluatedAt       time.Time         `json:"evaluated_at"`
}

// ViolationManifest is the structured output used by CI/CD to fail a build
type ViolationManifest struct {
	ExitCode     int               `json:"exit_code"`
	CriticalCount int               `json:"critical_count"`
	SafeCount    int               `json:"safe_count"`
	Violations   []Violation       `json:"violations"`
}

// SeverityToExitCode maps BreakingChangeSeverity to a CI/CD exit code
func SeverityToExitCode(sev BreakingChangeSeverity) int {
	switch sev {
	case SeverityCritical:
		return 1
	default:
		return 0
	}
}

// DownstreamBO describes a BO found during binding traversal
type DownstreamBO struct {
	BOName    string `json:"bo_name"`
	BOID      string `json:"bo_id"`
	FieldKey  string `json:"field_key,omitempty"`
	Reason    string `json:"reason"`
}

// ViolationRecord is the DB row type for public.data_contract_violations
type ViolationRecord struct {
	ViolationID  string          `db:"violation_id"`
	TenantID     string          `db:"tenant_id"`
	ContractID   string          `db:"contract_id"`
	Status       string          `db:"status"`
	Severity     string          `db:"severity"`
	TargetTable  string          `db:"target_table"`
	DatasourceID string          `db:"datasource_id"`
	ProposedDiff json.RawMessage `db:"proposed_diff"`
	Violations   json.RawMessage `db:"violations"`
	DetectedAt   time.Time       `db:"detected_at"`
	ReviewedBy   *string         `db:"reviewed_by"`
	ReviewedAt   *time.Time      `db:"reviewed_at"`
	TicketID     *string         `db:"ticket_id"`
	CreatedAt    time.Time       `db:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at"`
}

// ContractStatus values for ViolationRecord.Status
const (
	ContractStatusPending  = "PENDING_REVIEW"
	ContractStatusApproved = "APPROVED"
	ContractStatusBlocked  = "BLOCKED"
	ContractStatusTicketed = "TICKET_OPENED"
)
