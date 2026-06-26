package scheduler_intelligence

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ============================================================================
// Scoping Concepts
// ============================================================================

type ScopeType string

const (
	ScopeGlobal ScopeType = "GLOBAL"
	ScopeTenant ScopeType = "TENANT"
)

type ActorType string

const (
	ActorTenantOps ActorType = "TENANT_OPS"
	ActorGlobalOps ActorType = "GLOBAL_OPS"
)

type TenantContext struct {
	Actor    ActorType
	TenantID *uuid.UUID
}

type TenantID = uuid.UUID

// ============================================================================
// Job Category
// ============================================================================

type JobCategory string

const (
	JobCategoryReport      JobCategory = "report"
	JobCategoryWorkflow    JobCategory = "workflow"
	JobCategoryIntegration JobCategory = "integration"
	JobCategoryAI          JobCategory = "ai"
	JobCategoryPreAgg      JobCategory = "preagg"
	JobCategoryCompliance  JobCategory = "compliance"
	JobCategoryDataQuality JobCategory = "data_quality"
	JobCategoryMigration   JobCategory = "migration"
)

// ============================================================================
// Schedule Type
// ============================================================================

type ScheduleType string

const (
	ScheduleTypeCron       ScheduleType = "cron"
	ScheduleTypeEvent      ScheduleType = "event"
	ScheduleTypePredictive ScheduleType = "predictive"
	ScheduleTypeManual     ScheduleType = "manual"
)

// ============================================================================
// Run Status
// ============================================================================

type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "cancelled"
	RunStatusPaused    RunStatus = "paused"
	RunStatusRetrying  RunStatus = "retrying"
)

// ============================================================================
// Trigger Type
// ============================================================================

type TriggerType string

const (
	TriggerTypeScheduled TriggerType = "scheduled"
	TriggerTypeManual    TriggerType = "manual"
	TriggerTypeEvent     TriggerType = "event"
	TriggerTypeAPI       TriggerType = "api"
)

// ============================================================================
// Retry Policy
// ============================================================================

type RetryPolicy struct {
	MaxAttempts            int     `json:"max_attempts"`
	InitialIntervalSeconds int     `json:"initial_interval_seconds"`
	BackoffCoefficient     float64 `json:"backoff_coefficient"`
	MaxIntervalSeconds     int     `json:"max_interval_seconds,omitempty"`
}

// DefaultRetryPolicy returns the default retry policy
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:            3,
		InitialIntervalSeconds: 60,
		BackoffCoefficient:     2.0,
		MaxIntervalSeconds:     3600,
	}
}

// ============================================================================
// Semantic Binding
// ============================================================================

// SemanticBinding links scheduler objects to the semantic layer
type SemanticBinding struct {
	BOIDs       []string `json:"bo_ids,omitempty"`
	APIIDs      []string `json:"api_ids,omitempty"`
	PageIDs     []string `json:"page_ids,omitempty"`
	WorkflowIDs []string `json:"workflow_ids,omitempty"`
	PreAggIDs   []string `json:"preagg_ids,omitempty"`
}

// ToIDList flattens all bound IDs into a single list
func (b SemanticBinding) ToIDList() []string {
	var ids []string
	ids = append(ids, b.BOIDs...)
	ids = append(ids, b.APIIDs...)
	ids = append(ids, b.PageIDs...)
	ids = append(ids, b.WorkflowIDs...)
	ids = append(ids, b.PreAggIDs...)
	return ids
}

// HasAny returns true if the binding has any IDs
func (b SemanticBinding) HasAny() bool {
	return len(b.BOIDs) > 0 || len(b.APIIDs) > 0 ||
		len(b.PageIDs) > 0 || len(b.WorkflowIDs) > 0 || len(b.PreAggIDs) > 0
}

// SemanticImpact represents the downstream effects of a scheduler change
type SemanticImpact struct {
	AffectedBOs       []string `json:"affected_bos"`
	AffectedAPIs      []string `json:"affected_apis"`
	AffectedPages     []string `json:"affected_pages"`
	AffectedWorkflows []string `json:"affected_workflows"`
	AffectedPreAggs   []string `json:"affected_preaggs"`

	DownstreamJobs []string `json:"downstream_jobs"`
	DownstreamDAGs []string `json:"downstream_dags"`
}

// ImpactedObject represents an object affected by a semantic change
type ImpactedObject struct {
	ID   string `json:"id"`
	Type string `json:"type"` // BO, API, PAGE, WORKFLOW, PREAGG, JOB, DAG
}

// DriftStatus represents the drift state of a semantic object
type DriftStatus struct {
	SemanticID string `json:"semantic_id"`
	Severity   string `json:"severity"` // LOW, MEDIUM, HIGH
	Message    string `json:"message"`
}

// SemanticClient defines the interface for interacting with the semantic graph
type SemanticClient interface {
	// ResolveBindings takes reference IDs (names or paths) and returns normalized UUID bindings
	ResolveBindings(ctx context.Context, refIDs []string) (SemanticBinding, error)

	// GetImpactedObjects returns downstream objects affected by a change to semantic IDs
	GetImpactedObjects(ctx context.Context, semanticIDs []string) ([]ImpactedObject, error)

	// GetDriftStatus returns the current drift state for given semantic IDs
	GetDriftStatus(ctx context.Context, semanticIDs []string) ([]DriftStatus, error)

	// GetBOsByPhysicalTable returns Business Object IDs associated with a physical table name
	GetBOsByPhysicalTable(ctx context.Context, tableName string) ([]string, error)
}

// ============================================================================
// Event Trigger
// ============================================================================

type EventTrigger struct {
	EventType  string                 `json:"event_type"`
	Source     string                 `json:"source,omitempty"`
	Conditions map[string]interface{} `json:"conditions,omitempty"`
}

// ============================================================================
// Blackout Window
// ============================================================================

type BlackoutWindow struct {
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
	Reason string    `json:"reason,omitempty"`
}

// ============================================================================
// Compliance
// ============================================================================

type Compliance struct {
	PII         bool   `json:"pii"`
	Residency   string `json:"residency"`   // US, EU, GLOBAL
	Sensitivity string `json:"sensitivity"` // LOW, MEDIUM, HIGH
}

// Value implements driver.Valuer
func (c Compliance) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implements sql.Scanner
func (c *Compliance) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &c)
}

// ============================================================================
// Job - Scheduled Job Definition
// ============================================================================

type Job struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	Scope        ScopeType  `db:"scope" json:"scope"`
	TenantID     *uuid.UUID `db:"tenant_id" json:"tenant_id,omitempty"`
	ParentJobID  *uuid.UUID `db:"parent_job_id" json:"parent_job_id,omitempty"`
	DatasourceID *uuid.UUID `db:"datasource_id" json:"datasource_id,omitempty"`
	Name         string     `db:"name" json:"name"`
	Description  string     `db:"description" json:"description,omitempty"`
	Category     string     `db:"category" json:"category"`
	JobType      string     `db:"job_type" json:"job_type"`

	// Parameters
	Parameters json.RawMessage `db:"parameters" json:"parameters"`

	// Compliance Risk
	ComplianceRiskScore float64         `db:"compliance_risk_score" json:"compliance_risk_score"`
	ComplianceRiskLevel string          `db:"compliance_risk_level" json:"compliance_risk_level"`
	SemanticBindings    SemanticBinding `db:"semantic_bindings" json:"semantic_bindings"`

	// Scheduling
	ScheduleType   string          `db:"schedule_type" json:"schedule_type"`
	CronExpression *string         `db:"cron_expression" json:"cron_expression,omitempty"`
	EventTrigger   json.RawMessage `db:"event_trigger" json:"event_trigger,omitempty"`
	Timezone       string          `db:"timezone" json:"timezone"`

	// Calendars & Constraints
	CalendarIDs     []uuid.UUID     `db:"calendar_ids" json:"calendar_ids"`
	BlackoutWindows json.RawMessage `db:"blackout_windows" json:"blackout_windows"`
	Constraints     json.RawMessage `db:"constraints" json:"constraints"`

	// Execution Config
	RetryPolicy    json.RawMessage `db:"retry_policy" json:"retry_policy"`
	TimeoutSeconds int             `db:"timeout_seconds" json:"timeout_seconds"`
	Priority       int             `db:"priority" json:"priority"`

	// Risk & Compliance
	RiskScore        int             `db:"risk_score" json:"risk_score"`
	SLOCritical      bool            `db:"slo_critical" json:"slo_critical"`
	ComplianceTags   []string        `db:"compliance_tags" json:"compliance_tags"`
	PIIExposureLevel string          `db:"pii_exposure_level" json:"pii_exposure_level"` // Deprecated in favor of Compliance.Sensitivity
	ResidencyRules   json.RawMessage `db:"residency_rules" json:"residency_rules"`       // Deprecated in favor of Compliance.Residency
	Compliance       Compliance      `db:"compliance" json:"compliance"`

	// Governance
	ChangeSetID *uuid.UUID `db:"changeset_id" json:"changeset_id,omitempty"`
	ApprovedBy  *uuid.UUID `db:"approved_by" json:"approved_by,omitempty"`
	ApprovedAt  *time.Time `db:"approved_at" json:"approved_at,omitempty"`

	// Status
	IsActive  bool       `db:"is_active" json:"is_active"`
	LastRunAt *time.Time `db:"last_run_at" json:"last_run_at,omitempty"`
	NextRunAt *time.Time `db:"next_run_at" json:"next_run_at,omitempty"`

	// Audit
	CreatedBy *uuid.UUID `db:"created_by" json:"created_by,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
}

// ============================================================================
// DAG - Directed Acyclic Graph
// ============================================================================

type DAGNode struct {
	ID         string                 `json:"id"`
	JobID      uuid.UUID              `json:"job_id"`
	Conditions map[string]interface{} `json:"conditions,omitempty"`
	Position   *Position              `json:"position,omitempty"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type DAGEdge struct {
	FromNodeID string                 `json:"from_node_id"`
	ToNodeID   string                 `json:"to_node_id"`
	Type       string                 `json:"type,omitempty"` // success, completion, any
	Conditions map[string]interface{} `json:"conditions,omitempty"`
}

type DAG struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	Scope       ScopeType  `db:"scope" json:"scope"`
	TenantID    *uuid.UUID `db:"tenant_id" json:"tenant_id,omitempty"`
	ParentDAGID *uuid.UUID `db:"parent_dag_id" json:"parent_dag_id,omitempty"`
	Name        string     `db:"name" json:"name"`
	Description string     `db:"description" json:"description,omitempty"`
	Category    *string    `db:"category" json:"category,omitempty"`

	// Graph Structure
	Nodes json.RawMessage `db:"nodes" json:"nodes"`
	Edges json.RawMessage `db:"edges" json:"edges"`

	// Semantic
	SemanticBindings SemanticBinding `db:"semantic_bindings" json:"semantic_bindings"`

	// Scheduling
	ScheduleType   *string     `db:"schedule_type" json:"schedule_type,omitempty"`
	CronExpression *string     `db:"cron_expression" json:"cron_expression,omitempty"`
	CalendarIDs    []uuid.UUID `db:"calendar_ids" json:"calendar_ids"`
	Timezone       string      `db:"timezone" json:"timezone"`

	// Execution Config
	MaxParallelJobs int  `db:"max_parallel_jobs" json:"max_parallel_jobs"`
	FailFast        bool `db:"fail_fast" json:"fail_fast"`
	TimeoutSeconds  int  `db:"timeout_seconds" json:"timeout_seconds"`

	// Risk & Governance
	RiskScore   int        `db:"risk_score" json:"risk_score"`
	SLOCritical bool       `db:"slo_critical" json:"slo_critical"`
	ChangeSetID *uuid.UUID `db:"changeset_id" json:"changeset_id,omitempty"`

	// Status
	IsActive  bool       `db:"is_active" json:"is_active"`
	LastRunAt *time.Time `db:"last_run_at" json:"last_run_at,omitempty"`
	NextRunAt *time.Time `db:"next_run_at" json:"next_run_at,omitempty"`

	// Audit
	CreatedBy *uuid.UUID `db:"created_by" json:"created_by,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
}

// ============================================================================
// DAG Run
// ============================================================================

type DAGRun struct {
	ID       uuid.UUID `db:"id" json:"id"`
	DAGID    uuid.UUID `db:"dag_id" json:"dag_id"`
	TenantID uuid.UUID `db:"tenant_id" json:"tenant_id"`

	// Temporal
	TemporalWorkflowID string `db:"temporal_workflow_id" json:"temporal_workflow_id,omitempty"`
	TemporalRunID      string `db:"temporal_run_id" json:"temporal_run_id,omitempty"`

	// Status
	Status      string     `db:"status" json:"status"`
	TriggerType string     `db:"trigger_type" json:"trigger_type"`
	TriggeredBy *uuid.UUID `db:"triggered_by" json:"triggered_by,omitempty"`

	// Timing
	ScheduledAt *time.Time `db:"scheduled_at" json:"scheduled_at,omitempty"`
	StartedAt   *time.Time `db:"started_at" json:"started_at,omitempty"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	DurationMS  *int       `db:"duration_ms" json:"duration_ms,omitempty"`

	// Results
	CompletedJobs int            `db:"completed_jobs" json:"completed_jobs"`
	FailedJobs    int            `db:"failed_jobs" json:"failed_jobs"`
	SkippedJobs   int            `db:"skipped_jobs" json:"skipped_jobs"`
	ErrorMessage  sql.NullString `db:"error_message" json:"error_message,omitempty"`

	// Metadata
	Metadata  json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}

// ============================================================================
// Job Run
// ============================================================================

type JobRun struct {
	ID       uuid.UUID  `db:"id" json:"id"`
	JobID    uuid.UUID  `db:"job_id" json:"job_id"`
	DAGRunID *uuid.UUID `db:"dag_run_id" json:"dag_run_id,omitempty"`
	TenantID uuid.UUID  `db:"tenant_id" json:"tenant_id"`

	// Temporal
	TemporalWorkflowID string `db:"temporal_workflow_id" json:"temporal_workflow_id,omitempty"`
	TemporalRunID      string `db:"temporal_run_id" json:"temporal_run_id,omitempty"`
	TaskQueue          string `db:"task_queue" json:"task_queue,omitempty"`

	// Status
	Status        string     `db:"status" json:"status"`
	AttemptNumber int        `db:"attempt_number" json:"attempt_number"`
	TriggerType   string     `db:"trigger_type" json:"trigger_type"`
	TriggeredBy   *uuid.UUID `db:"triggered_by" json:"triggered_by,omitempty"`
	ExecutionMode string     `db:"execution_mode" json:"execution_mode"` // NORMAL, DRY_RUN, CANARY

	// Timing
	ScheduledAt *time.Time `db:"scheduled_at" json:"scheduled_at,omitempty"`
	StartedAt   *time.Time `db:"started_at" json:"started_at,omitempty"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	DurationMS  *int       `db:"duration_ms" json:"duration_ms,omitempty"`

	// Results
	Result       json.RawMessage `db:"result" json:"result,omitempty"`
	ErrorMessage sql.NullString  `db:"error_message" json:"error_message,omitempty"`
	ErrorDetails json.RawMessage `db:"error_details" json:"error_details,omitempty"`

	// SLO
	SLOTargetMS *int `db:"slo_target_ms" json:"slo_target_ms,omitempty"`
	SLOBreached bool `db:"slo_breached" json:"slo_breached"`

	// Metadata
	InputParameters  json.RawMessage `db:"input_parameters" json:"input_parameters,omitempty"`
	OutputProperties json.RawMessage `db:"output_properties" json:"output_properties,omitempty"`
	SemanticBindings SemanticBinding `db:"semantic_bindings" json:"semantic_bindings"`
	LogsURL          sql.NullString  `db:"logs_url" json:"logs_url,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// GovernancePolicy represents a rule for change set reviews
type GovernancePolicy struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	TenantID    *uuid.UUID      `json:"tenant_id,omitempty" db:"tenant_id"`
	Scope       ScopeType       `json:"scope" db:"scope"`
	Name        string          `json:"name" db:"name"`
	Description string          `json:"description" db:"description"`
	Rules       json.RawMessage `json:"rules" db:"rules"`
	IsActive    bool            `json:"is_active" db:"is_active"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// AI Suggestion
// ============================================================================

type AISuggestion struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	TenantID       uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	SuggestionType string     `db:"suggestion_type" json:"suggestion_type"`
	TargetType     *string    `db:"target_type" json:"target_type,omitempty"`
	TargetID       *uuid.UUID `db:"target_id" json:"target_id,omitempty"`

	// Content
	Title           string      `db:"title" json:"title"`
	Description     string      `db:"description" json:"description,omitempty"`
	ImpactSummary   string      `db:"impact_summary" json:"impact_summary,omitempty"`
	RiskLevel       string      `db:"risk_level" json:"risk_level"`
	AffectedTenants []uuid.UUID `db:"affected_tenants" json:"affected_tenants"`

	// Proposed Changes
	ProposedChanges json.RawMessage `db:"proposed_changes" json:"proposed_changes"`

	// Status
	Status          string     `db:"status" json:"status"`
	DismissedReason *string    `db:"dismissed_reason" json:"dismissed_reason,omitempty"`
	SnoozedUntil    *time.Time `db:"snoozed_until" json:"snoozed_until,omitempty"`

	// Governance
	ChangeSetID *uuid.UUID `db:"changeset_id" json:"changeset_id,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// ============================================================================
// Governance - ChangeSets
// ============================================================================

type ChangeSetType string

const (
	ChangeSetTypeJobCreate   ChangeSetType = "scheduler.job.create"
	ChangeSetTypeJobUpdate   ChangeSetType = "scheduler.job.update"
	ChangeSetTypeJobDelete   ChangeSetType = "scheduler.job.delete"
	ChangeSetTypeDAGCreate   ChangeSetType = "scheduler.dag.create"
	ChangeSetTypeDAGUpdate   ChangeSetType = "scheduler.dag.update"
	ChangeSetTypeDAGDelete   ChangeSetType = "scheduler.dag.delete"
	ChangeSetTypeCalendarMod ChangeSetType = "scheduler.calendar.modify"
)

type ChangeSetStatus string

const (
	ChangeSetStatusDraft      ChangeSetStatus = "draft"
	ChangeSetStatusPending    ChangeSetStatus = "pending_review"
	ChangeSetStatusApproved   ChangeSetStatus = "approved"
	ChangeSetStatusRejected   ChangeSetStatus = "rejected"
	ChangeSetStatusApplied    ChangeSetStatus = "applied"
	ChangeSetStatusRolledBack ChangeSetStatus = "rolled_back"
)

type ImpactAnalysis struct {
	AffectedJobs    []string `json:"affected_jobs"`
	AffectedDAGs    []string `json:"affected_dags"`
	AffectedTenants []string `json:"affected_tenants"`
	BlastRadius     int      `json:"blast_radius"`
	SLOImpact       bool     `json:"slo_impact"`
	PIIExposure     string   `json:"pii_exposure"` // none, low, medium, high
	ComplianceTags  []string `json:"compliance_tags"`
}

type AIReview struct {
	Summary        string   `json:"summary"`
	RiskScore      float64  `json:"risk_score"`
	FlaggedIssues  []string `json:"flagged_issues"`
	Recommendation string   `json:"recommendation"`
}

type SchedulerChangeSet struct {
	ID               uuid.UUID       `db:"id" json:"id"`
	TenantID         *uuid.UUID      `db:"tenant_id" json:"tenant_id,omitempty"`
	Scope            ScopeType       `db:"scope" json:"scope"`
	Type             ChangeSetType   `db:"type" json:"type"`
	Title            string          `db:"title" json:"title"`
	Description      string          `db:"description,omitempty"`
	Author           string          `db:"author" json:"author"`
	Status           ChangeSetStatus `db:"status" json:"status"`
	TargetType       string          `db:"target_type" json:"target_type"`
	TargetID         *uuid.UUID      `db:"target_id" json:"target_id,omitempty"`
	Diff             json.RawMessage `db:"diff" json:"diff"`
	ImpactAnalysis   json.RawMessage `db:"impact_analysis" json:"impact_analysis"`
	AIReview         json.RawMessage `db:"ai_review" json:"ai_review"`
	RiskScore        float64         `db:"risk_score" json:"risk_score"`
	Tags             pq.StringArray  `db:"tags" json:"tags"`
	Metadata         json.RawMessage `db:"metadata" json:"metadata"`
	SemanticBindings SemanticBinding `db:"semantic_bindings" json:"semantic_bindings"`
	CreatedAt        time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at" json:"updated_at"`
}

type ChangeSetApproval struct {
	ID           uuid.UUID `db:"id" json:"id"`
	ChangeSetID  uuid.UUID `db:"changeset_id" json:"changeset_id"`
	ApproverID   string    `db:"approver_id" json:"approver_id"`
	ApproverRole string    `db:"approver_role" json:"approver_role"`
	Decision     string    `db:"decision" json:"decision"`
	Comment      string    `db:"comment,omitempty" json:"comment,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// ============================================================================
// Request/Response Types
// ============================================================================

type CreateJobRequest struct {
	Name           string                 `json:"name" validate:"required"`
	Description    string                 `json:"description"`
	Category       string                 `json:"category" validate:"required"`
	JobType        string                 `json:"job_type" validate:"required"`
	Parameters     map[string]interface{} `json:"parameters"`
	SemanticSpec   SemanticBinding        `json:"semantic_spec,omitempty"`
	ScheduleType   string                 `json:"schedule_type" validate:"required"`
	CronExpression string                 `json:"cron_expression"`
	Timezone       string                 `json:"timezone"`
	CalendarIDs    []string               `json:"calendar_ids"`
	TimeoutSeconds int                    `json:"timeout_seconds"`
	Priority       int                    `json:"priority"`
	RetryPolicy    *RetryPolicy           `json:"retry_policy"`
	SLOCritical    bool                   `json:"slo_critical"`
	ComplianceTags []string               `json:"compliance_tags"`
}

type UpdateJobRequest struct {
	Name           *string                `json:"name"`
	Description    *string                `json:"description"`
	Category       *string                `json:"category"`
	Parameters     map[string]interface{} `json:"parameters"`
	SemanticSpec   *SemanticBinding       `json:"semantic_spec,omitempty"`
	ScheduleType   *string                `json:"schedule_type"`
	CronExpression *string                `json:"cron_expression"`
	Timezone       *string                `json:"timezone"`
	CalendarIDs    []string               `json:"calendar_ids"`
	TimeoutSeconds *int                   `json:"timeout_seconds"`
	Priority       *int                   `json:"priority"`
	IsActive       *bool                  `json:"is_active"`
}

type CreateDAGRequest struct {
	Name            string          `json:"name" validate:"required"`
	Description     string          `json:"description"`
	Category        string          `json:"category"`
	SemanticSpec    SemanticBinding `json:"semantic_spec,omitempty"`
	Nodes           []DAGNode       `json:"nodes" validate:"required"`
	Edges           []DAGEdge       `json:"edges"`
	ScheduleType    string          `json:"schedule_type"`
	CronExpression  string          `json:"cron_expression"`
	CalendarIDs     []string        `json:"calendar_ids"`
	MaxParallelJobs int             `json:"max_parallel_jobs"`
	FailFast        bool            `json:"fail_fast"`
	TimeoutSeconds  int             `json:"timeout_seconds"`
}

type UpdateDAGRequest struct {
	Name            *string          `json:"name"`
	Description     *string          `json:"description"`
	Category        *string          `json:"category"`
	SemanticSpec    *SemanticBinding `json:"semantic_spec,omitempty"`
	ScheduleType    *string          `json:"schedule_type"`
	CronExpression  *string          `json:"cron_expression"`
	Timezone        *string          `json:"timezone"`
	MaxParallelJobs *int             `json:"max_parallel_jobs"`
	FailFast        *bool            `json:"fail_fast"`
	TimeoutSeconds  *int             `json:"timeout_seconds"`
	Nodes           []DAGNode        `json:"nodes"`
	Edges           []DAGEdge        `json:"edges"`
}

type TriggerJobRequest struct {
	Parameters map[string]interface{} `json:"parameters"`
}

type JobListFilters struct {
	TenantID     string
	DatasourceID string
	Scope        string
	Category     string
	Status       string
	IsActive     *bool
	SLOCritical  *bool
	Limit        int
	Offset       int
}

type JobRunListFilters struct {
	JobID    string
	DAGRunID string
	TenantID string
	Status   string
	Limit    int
	Offset   int
}
