package audit

import (
	"encoding/json"
	"time"
)

// KafkaEventEnvelope wraps all audit events for consistent transport
type KafkaEventEnvelope struct {
	EventID   string          `json:"eventId"`
	EventType string          `json:"eventType"`
	Version   string          `json:"version"`
	Timestamp time.Time       `json:"timestamp"`
	TenantID  string          `json:"tenantId"`
	Source    string          `json:"source"`
	Payload   json.RawMessage `json:"payload"`
}

// JobRunCompletedEvent is emitted when a scheduler job completes
// ...
type JobRunCompletedEvent struct {
	RunID             string          `json:"runId"`
	JobID             string          `json:"jobId"`
	DagID             string          `json:"dagId,omitempty"`
	TenantID          string          `json:"tenantId"`
	StartTS           time.Time       `json:"startTs"`
	EndTS             time.Time       `json:"endTs"`
	Status            string          `json:"status"`
	ErrorMessage      string          `json:"errorMessage,omitempty"`
	SemanticContext   json.RawMessage `json:"semanticContext"`
	ComplianceContext json.RawMessage `json:"complianceContext"`
	SLOContext        json.RawMessage `json:"sloContext"`
	AINarrative       json.RawMessage `json:"aiNarrative,omitempty"`
	Metadata          json.RawMessage `json:"metadata"`
	SemanticTerms     []string        `json:"semanticTerms"` // Added
}

// DAGRunCompletedEvent is emitted when a DAG completes
type DAGRunCompletedEvent struct {
	DagRunID      string          `json:"dagRunId"`
	DagID         string          `json:"dagId"`
	TenantID      string          `json:"tenantId"`
	StartTS       time.Time       `json:"startTs"`
	EndTS         time.Time       `json:"endTs"`
	Status        string          `json:"status"`
	CriticalPath  json.RawMessage `json:"criticalPath"`
	AIRootCause   json.RawMessage `json:"aiRootCause,omitempty"`
	Metadata      json.RawMessage `json:"metadata"`
	SemanticTerms []string        `json:"semanticTerms"` // Added
}

// ImpactedEntity represents an entity affected by a ChangeSet
type ImpactedEntity struct {
	ID         string `json:"id"`
	NodeID     string `json:"nodeId"`
	EntityType string `json:"entityType"` // SEMANTIC_TERM, JOB, DAG, BUSINESS_TERM, PAGE, API
}

// IncidentEvent represents a clustered operational incident in audit form
type IncidentEvent struct {
	IncidentID       string          `json:"incidentId"`
	TenantID         string          `json:"tenantId"`
	Status           string          `json:"status"` // OPEN, INVESTIGATING, RESOLVED, CLOSED
	Severity         string          `json:"severity"`
	Title            string          `json:"title"`
	Description      string          `json:"description"`
	DetectedAt       time.Time       `json:"detectedAt"`
	ResolvedAt       time.Time       `json:"resolvedAt,omitempty"`
	StartTS          time.Time       `json:"startTs"` // Incident start
	EndTS            time.Time       `json:"endTs"`   // Incident end
	EventCount       int             `json:"eventCount"`
	AffectedTerms    []string        `json:"affectedTerms"`
	CauseEventIDs    []string        `json:"causeEventIds"`
	RelatedJobRunIDs []string        `json:"relatedJobRunIds"`
	RelatedDAGRunIDs []string        `json:"relatedDagRunIds"`
	BlastRadius      string          `json:"blastRadius"`
	Metadata         json.RawMessage `json:"metadata"`
}

// ChangeSetCreatedEvent is emitted when a governance changeset is created
type ChangeSetCreatedEvent struct {
	ChangesetID      string           `json:"changesetId"`
	Title            string           `json:"title"`       // Added
	Description      string           `json:"description"` // Added
	Type             string           `json:"type"`
	Actor            string           `json:"actor"`
	Source           string           `json:"source"`   // Added
	TenantID         string           `json:"tenantId"` // may be empty for global changes
	CreatedAt        time.Time        `json:"createdAt"`
	PayloadOld       json.RawMessage  `json:"payloadOld"`
	PayloadNew       json.RawMessage  `json:"payloadNew"`
	SemanticImpact   json.RawMessage  `json:"semanticImpact"`
	ComplianceImpact json.RawMessage  `json:"complianceImpact"`
	TenantImpact     json.RawMessage  `json:"tenantImpact"`
	AISummary        json.RawMessage  `json:"aiSummary,omitempty"`
	AIRisk           json.RawMessage  `json:"aiRisk,omitempty"`
	Approvers        []string         `json:"approvers"`
	Status           string           `json:"status"`
	ImpactedEntities []ImpactedEntity `json:"impactedEntities"` // Added
}

// SemanticSnapshotEvent is emitted when semantic term state is captured
type SemanticSnapshotEvent struct {
	SnapshotID     string          `json:"snapshotId"`
	SemanticTermID string          `json:"semanticTermId"`
	Version        int             `json:"version"`
	Timestamp      time.Time       `json:"timestamp"`
	Definition     string          `json:"definition"`
	Snapshot       json.RawMessage `json:"snapshot"` // Added for full object state
	BusinessTermID string          `json:"businessTermId"`
	TenantID       string          `json:"tenantId"`         // may be empty for global terms
	Region         string          `json:"region,omitempty"` // added region for multi-region snapshots
	Compliance     json.RawMessage `json:"compliance"`
	Lineage        json.RawMessage `json:"lineage"`
	Metadata       json.RawMessage `json:"metadata"`
}

// OrchestrationWorkflowEvent is emitted for Temporal workflow events
type OrchestrationWorkflowEvent struct {
	EventID           string          `json:"eventId"`
	WorkflowID        string          `json:"workflowId"`
	EventType         string          `json:"eventType"`
	TenantID          string          `json:"tenantId"` // may be empty for global workflows
	Timestamp         time.Time       `json:"timestamp"`
	Payload           json.RawMessage `json:"payload"`
	ComplianceContext json.RawMessage `json:"complianceContext,omitempty"`
	SemanticContext   json.RawMessage `json:"semanticContext,omitempty"`
}

// ComplianceViolationEvent is emitted when a compliance violation is detected
type ComplianceViolationEvent struct {
	ViolationID      string           `json:"violationId"`
	TenantID         string           `json:"tenantId"`
	JobRunID         string           `json:"jobRunId,omitempty"`
	ViolatedAt       time.Time        `json:"violatedAt"`
	ViolationType    string           `json:"violationType"`
	Severity         string           `json:"severity"`
	Status           string           `json:"status"` // Added
	PIIExposed       bool             `json:"piiExposed"`
	AffectedRecords  int64            `json:"affectedRecords"`
	ImpactedEntities []ImpactedEntity `json:"impactedEntities"` // Added
	ComplianceRefs   []string         `json:"complianceRefs"`   // Added
	Narrative        string           `json:"narrative"`
	Metadata         json.RawMessage  `json:"metadata"`
}

// AISuggestionEvent is emitted when an AI system generates a suggestion or narrative
type AISuggestionEvent struct {
	SuggestionID     string          `json:"suggestionId"`
	TenantID         string          `json:"tenantId"`
	Type             string          `json:"type"`           // "root_cause", "remediation", "impact_analysis", "narrative"
	SuggestionType   string          `json:"suggestionType"` // Alias for Type
	RelatedEventID   string          `json:"relatedEventId"`
	RelatedEventType string          `json:"relatedEventType"` // "incident", "compliance_event", "changeset"
	Narrative        string          `json:"narrative"`
	RootCause        string          `json:"rootCause"`        // Added
	BlastRadius      string          `json:"blastRadius"`      // Added
	RecommendedFix   string          `json:"recommendedFix"`   // Added
	ChangeSetSummary string          `json:"changeSetSummary"` // Added
	Confidence       float64         `json:"confidence"`
	GeneratedBy      string          `json:"generatedBy"` // "gemini", "claude", "o1"
	GeneratedAt      time.Time       `json:"generatedAt"`
	Context          json.RawMessage `json:"context,omitempty"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
}

// AIQueryExecutionEvent is emitted when an AI semantic query is compiled and run
type AIQueryExecutionEvent struct {
	QueryID        string          `json:"queryId"`
	TenantID       string          `json:"tenantId"`
	UserEmail      string          `json:"userEmail"`
	InputPrompt    string          `json:"inputPrompt"`
	GeneratedSQL   string          `json:"generatedSql"`
	IsMasked       bool            `json:"isMasked"`
	MaskedColumns  []string        `json:"maskedColumns,omitempty"`
	FunctionalRole string          `json:"functionalRole"`
	ExecutedAt     time.Time       `json:"executedAt"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

// Kafka Topics for audit events
const (
	TopicSchedulerJobRuns     = "audit.scheduler.job_runs"
	TopicSchedulerDAGRuns     = "audit.scheduler.dag_runs"
	TopicGovernanceChangeSets = "audit.governance.changesets"
	TopicSemanticSnapshots    = "audit.semantic.snapshots"
	TopicOrchestrationEvents  = "audit.orchestration.events"
	TopicComplianceViolations = "audit.compliance.violations"
	TopicAISuggestions        = "audit.ai.suggestions"
	TopicAIQueryAudits        = "audit.ai.queries"
)

// Event types for routing
const (
	EventTypeJobRunCompleted      = "JOB_RUN_COMPLETED"
	EventTypeDAGRunCompleted      = "DAG_RUN_COMPLETED"
	EventTypeChangeSetCreated     = "CHANGESET_CREATED"
	EventTypeChangeSetApproved    = "CHANGESET_APPROVED"
	EventTypeChangeSetApplied     = "CHANGESET_APPLIED"
	EventTypeSemanticSnapshot     = "SEMANTIC_SNAPSHOT"
	EventTypeWorkflowStarted      = "WORKFLOW_STARTED"
	EventTypeWorkflowCompleted    = "WORKFLOW_COMPLETED"
	EventTypeWorkflowFailed       = "WORKFLOW_FAILED"
	EventTypeComplianceViolation  = "COMPLIANCE_VIOLATION"
	EventTypeAINarrativeGenerated = "AI_NARRATIVE_GENERATED"
	EventTypeAIQueryExecuted      = "AI_QUERY_EXECUTED"
)
