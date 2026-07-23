package audit

import (
	"encoding/json"
	"time"
)

// ChangeSet represents a governance changeset
type ChangeSet struct {
	ID               string           `json:"id"`
	TenantID         string           `json:"tenantId"`
	Status           string           `json:"status"`
	ImpactedEntities []ImpactedEntity `json:"impactedEntities"`
}

// ChangeSetResponse is the response format for ChangeSet mutations
type ChangeSetResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// Aliases for compatibility with AI and GraphQL packages
// These map legacy names to the canonical Kafka event structs

type JobRunEvent = JobRunCompletedEvent
type ChangeSetEvent = ChangeSetCreatedEvent

// SchedulerJobRun represents a job run for reporting and AI analysis
type SchedulerJobRun struct {
	RunID             string          `json:"run_id"`
	JobID             string          `json:"job_id"`
	DagID             string          `json:"dag_id"`
	TenantID          string          `json:"tenant_id"`
	Status            string          `json:"status"`
	ErrorMessage      string          `json:"error_message"`
	StartTS           time.Time       `json:"start_ts"`
	EndTS             time.Time       `json:"end_ts"`
	SemanticContext   json.RawMessage `json:"semantic_context"`
	ComplianceContext json.RawMessage `json:"compliance_context"`
	SLOContext        json.RawMessage `json:"slo_context"`
	AINarrative       json.RawMessage `json:"ai_narrative"`
	Metadata          json.RawMessage `json:"metadata"`
	IngestTS          time.Time       `json:"ingest_ts"`
	SourceService     string          `json:"source_service"`
	SchemaVersion     string          `json:"schema_version"`
}

// SchedulerDAGRun represents a DAG run
type SchedulerDAGRun struct {
	DagRunID      string          `json:"dag_run_id"`
	DagID         string          `json:"dag_id"`
	TenantID      string          `json:"tenant_id"`
	Status        string          `json:"status"`
	StartTS       time.Time       `json:"start_ts"`
	EndTS         time.Time       `json:"end_ts"`
	CriticalPath  json.RawMessage `json:"critical_path"`
	AIRootCause   json.RawMessage `json:"ai_root_cause"`
	Metadata      json.RawMessage `json:"metadata"`
	IngestTS      time.Time       `json:"ingest_ts"`
	SourceService string          `json:"source_service"`
	SchemaVersion string          `json:"schema_version"`
}

// GovernanceChangeSet represents a changeset for reporting and AI analysis
type GovernanceChangeSet struct {
	ChangesetID      string          `json:"changeset_id"`
	TenantID         string          `json:"tenant_id"`
	Type             string          `json:"type"`
	Status           string          `json:"status"`
	Actor            string          `json:"actor"`
	CreatedAt        time.Time       `json:"created_at"`
	PayloadOld       json.RawMessage `json:"payload_old"`
	PayloadNew       json.RawMessage `json:"payload_new"`
	SemanticImpact   json.RawMessage `json:"semantic_impact"`
	ComplianceImpact json.RawMessage `json:"compliance_impact"`
	TenantImpact     json.RawMessage `json:"tenant_impact"`
	AIRisk           json.RawMessage `json:"ai_risk"`
	AISummary        json.RawMessage `json:"ai_summary"`
	Approvers        []string        `json:"approvers"`
	IngestTS         time.Time       `json:"ingest_ts"`
	SourceService    string          `json:"source_service"`
	SchemaVersion    string          `json:"schema_version"`
}

// SemanticSnapshot represents a point-in-time capture of a semantic term
type SemanticSnapshot struct {
	SnapshotID     string          `json:"snapshot_id"`
	SemanticTermID string          `json:"semantic_term_id"`
	BusinessTermID string          `json:"business_term_id"`
	TenantID       string          `json:"tenant_id"`
	Region         string          `json:"region,omitempty"`
	Version        int             `json:"version"`
	Definition     string          `json:"definition"`
	Timestamp      time.Time       `json:"timestamp"`
	Compliance     json.RawMessage `json:"compliance"`
	Lineage        json.RawMessage `json:"lineage"`
	Metadata       json.RawMessage `json:"metadata"`
	IngestTS       time.Time       `json:"ingest_ts"`
	SourceService  string          `json:"source_service"`
	SchemaVersion  string          `json:"schema_version"`
}

// ComplianceViolation represents a compliance violation for reporting
type ComplianceViolation struct {
	ViolationID     string          `json:"violation_id"`
	TenantID        string          `json:"tenant_id"`
	Severity        string          `json:"severity"`
	ViolationType   string          `json:"violation_type"`
	ComplianceRefs  []string        `json:"compliance_refs"`
	ViolatedAt      time.Time       `json:"violated_at"`
	RemediatedAt    time.Time       `json:"remediated_at"`
	PIIExposed      bool            `json:"pii_exposed"`
	AffectedRecords int64           `json:"affected_records"`
	Narrative       string          `json:"narrative"`
	JobRunID        string          `json:"job_run_id"`
	Metadata        json.RawMessage `json:"metadata"`
	IngestTS        time.Time       `json:"ingest_ts"`
	SourceService   string          `json:"source_service"`
	SchemaVersion   string          `json:"schema_version"`
}

// AIAuditSuggestion represents an AI-generated suggestion
type AIAuditSuggestion struct {
	SuggestionID     string          `json:"suggestion_id"`
	AuditRecordID    string          `json:"audit_record_id"` // RelatedEventID?
	RecordType       string          `json:"record_type"`     // RelatedEventType?
	TenantID         string          `json:"tenant_id"`
	Type             string          `json:"type"`
	Narrative        string          `json:"narrative"`
	RootCause        string          `json:"root_cause"`
	BlastRadius      string          `json:"blast_radius"`
	RecommendedFix   string          `json:"recommended_fix"`
	SuggestedActions []string        `json:"suggested_actions"`
	Confidence       float64         `json:"confidence"`
	GeneratedAt      time.Time       `json:"generated_at"`
	Timestamp        time.Time       `json:"timestamp"` // Alias for GeneratedAt?
	RelatedEventID   string          `json:"related_event_id"`
	RelatedEventType string          `json:"related_event_type"`
	Context          json.RawMessage `json:"context"`
	Metadata         json.RawMessage `json:"metadata"`
	IngestTS         time.Time       `json:"ingest_ts"`
	SourceService    string          `json:"source_service"`
	SchemaVersion    string          `json:"schema_version"`
}

const (
	JobStatusSuccess         = "SUCCESS"
	JobStatusFailed          = "FAILED"
	JobStatusComplianceBlock = "COMPLIANCE_BLOCK"

	ViolationSeverityLow    = "LOW"
	ViolationSeverityMedium = "MEDIUM"
	ViolationSeverityHigh   = "HIGH"

	ChangeSetStatusPending  = "PENDING"
	ChangeSetStatusApproved = "APPROVED"
	ChangeSetStatusApplied  = "APPLIED"
	ChangeSetStatusRejected = "REJECTED"
)
