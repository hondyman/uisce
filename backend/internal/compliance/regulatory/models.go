package regulatory

import (
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/compliance/drift"
)

type CaseSource string

const (
	SourceRegulatorPublication CaseSource = "REGULATOR_PUBLICATION"
	SourceESMAQA               CaseSource = "ESMA_QA"
	SourceScheduledReview      CaseSource = "SCHEDULED_REVIEW"
	SourceTenantSteward        CaseSource = "TENANT_STEWARD"
	SourceInternal             CaseSource = "INTERNAL"
	SourceClientIntake         CaseSource = "CLIENT_INTAKE"
)

type CaseClassification string

const (
	ClassificationNoImpact           CaseClassification = "NO_IMPACT"
	ClassificationInterpretationOnly CaseClassification = "INTERPRETATION_ONLY"
	ClassificationEditorial          CaseClassification = "EDITORIAL"
	ClassificationParameterChange    CaseClassification = "PARAMETER_CHANGE"
	ClassificationSemanticChange     CaseClassification = "SEMANTIC_CHANGE"
	ClassificationNewRuleRequired    CaseClassification = "NEW_RULE_REQUIRED"
)

type CaseStatus string

const (
	StatusIntaked            CaseStatus = "INTAKED"
	StatusTriaged            CaseStatus = "TRIAGED"
	StatusUnderReview        CaseStatus = "UNDER_REVIEW"
	StatusApprovedForPublish CaseStatus = "APPROVED_FOR_PUBLISH"
	StatusPublished          CaseStatus = "PUBLISHED"
	StatusClosedNoImpact     CaseStatus = "CLOSED_NO_IMPACT"
	StatusRejected           CaseStatus = "REJECTED"
	StatusEscalated          CaseStatus = "ESCALATED"
	StatusNewRuleBacklog     CaseStatus = "NEW_RULE_BACKLOG"
	StatusExpired            CaseStatus = "EXPIRED"
)

type CaseEventType string

const (
	EventCaseOpened       CaseEventType = "CASE_OPENED"
	EventTriaged          CaseEventType = "TRIAGED"
	EventReviewStarted    CaseEventType = "REVIEW_STARTED"
	EventStakeholderNoted CaseEventType = "STAKEHOLDER_NOTED"
	EventCorpusRun        CaseEventType = "CORPUS_RUN"
	EventApproved         CaseEventType = "APPROVED"
	EventPublished        CaseEventType = "PUBLISHED"
	EventClosed           CaseEventType = "CLOSED"
	EventEscalated        CaseEventType = "ESCALATED"
	EventNewRuleRouted    CaseEventType = "NEW_RULE_ROUTED"
	EventExpired          CaseEventType = "EXPIRED"
	EventRejected         CaseEventType = "REJECTED"
)

type NotificationKind string

const (
	NotificationInheritAdvance NotificationKind = "INHERIT_ADVANCE"
	NotificationDriftFlag       NotificationKind = "DRIFT_FLAG"
	NotificationStaleRegulation NotificationKind = "STALE_REGULATION"
	NotificationCaseEscalated   NotificationKind = "CASE_ESCALATED"
	NotificationSystem          NotificationKind = "SYSTEM"
)

// PublishedRuleVersionEntry tracks version bump per rule in published_rule_versions
type PublishedRuleVersionEntry struct {
	RuleID               uuid.UUID `json:"rule_id"`
	RuleCode             string    `json:"rule_code"`
	FromVersion          int       `json:"from_version"`
	ToVersion            int       `json:"to_version"`
	ApprovedContentHash  string    `json:"approved_content_hash"`
	PublishedContentHash string    `json:"published_content_hash"`
}

// RegulatoryChangeCase represents a regulatory intake case
type RegulatoryChangeCase struct {
	ID                    uuid.UUID                   `json:"id"`
	CaseCode              string                      `json:"case_code"`
	Source                CaseSource                  `json:"source"`
	SourceReference       string                      `json:"source_reference"`
	Title                 string                      `json:"title"`
	Description           string                      `json:"description"`
	AffectedRuleIDs       []uuid.UUID                 `json:"affected_rule_ids"`
	Classification        *CaseClassification        `json:"classification,omitempty"`
	TriageNotes           string                      `json:"triage_notes,omitempty"`
	TriagedBy             string                      `json:"triaged_by,omitempty"`
	TriagedAt             *time.Time                  `json:"triaged_at,omitempty"`
	Status                CaseStatus                  `json:"status"`
	IsEscalated           bool                        `json:"is_escalated"`
	EscalationCount       int                         `json:"escalation_count"`
	EscalatedAt           *time.Time                  `json:"escalated_at,omitempty"`
	LastEscalatedAt       *time.Time                  `json:"last_escalated_at,omitempty"`
	PublishedRuleVersions []PublishedRuleVersionEntry `json:"published_rule_versions,omitempty"`
	DueAt                 time.Time                   `json:"due_at"`
	CreatedBy             string                      `json:"created_by"`
	CreatedAt             time.Time                   `json:"created_at"`
	UpdatedAt             time.Time                   `json:"updated_at"`
}

// RegulatoryDraftRule stores durable, content-addressed proposed rule changes bound to approval
type RegulatoryDraftRule struct {
	ID                            uuid.UUID              `json:"id"`
	CaseID                        uuid.UUID              `json:"case_id"`
	RuleID                        uuid.UUID              `json:"rule_id"`
	ProposedAST                   map[string]interface{} `json:"proposed_ast"`
	ProposedParameterThresholds map[string]interface{} `json:"proposed_parameter_thresholds"`
	ProposedCitation            string                 `json:"proposed_citation"`
	ProposedContentHash         string                 `json:"proposed_content_hash"`
	CorpusResults                 *drift.CorpusRunResult `json:"corpus_results,omitempty"`
	IsApproved                    bool                   `json:"is_approved"`
	ApprovedBy                    string                 `json:"approved_by,omitempty"`
	ApprovedAt                    *time.Time             `json:"approved_at,omitempty"`
	CreatedAt                     time.Time              `json:"created_at"`
	UpdatedAt                     time.Time              `json:"updated_at"`
}

// RegulatoryCaseEvent represents an immutable case event audit record
type RegulatoryCaseEvent struct {
	ID        uuid.UUID              `json:"id"`
	CaseID    uuid.UUID              `json:"case_id"`
	EventType CaseEventType          `json:"event_type"`
	Actor     string                 `json:"actor"`
	Payload   map[string]interface{} `json:"payload"`
	CreatedAt time.Time              `json:"created_at"`
}

// ComplianceNotification represents an in-app blotter alert
type ComplianceNotification struct {
	ID        uuid.UUID              `json:"id"`
	TenantID  uuid.UUID              `json:"tenant_id"`
	Kind      NotificationKind       `json:"kind"`
	Title     string                 `json:"title"`
	Payload   map[string]interface{} `json:"payload"`
	IsRead    bool                   `json:"is_read"`
	CreatedAt time.Time              `json:"created_at"`
}

// UnaddressedRegulatoryCase models operational SLA metrics from compliance.v_unaddressed_regulatory_cases
type UnaddressedRegulatoryCase struct {
	CaseID                uuid.UUID                   `json:"case_id"`
	CaseCode              string                      `json:"case_code"`
	Title                 string                      `json:"title"`
	Source                string                      `json:"source"`
	SourceReference       string                      `json:"source_reference"`
	Status                string                      `json:"status"`
	Classification        string                      `json:"classification"`
	IsEscalated           bool                        `json:"is_escalated"`
	EscalationCount       int                         `json:"escalation_count"`
	DueAt                 time.Time                   `json:"due_at"`
	CreatedAt             time.Time                   `json:"created_at"`
	TriagedAt             *time.Time                  `json:"triaged_at,omitempty"`
	PublishedRuleVersions []PublishedRuleVersionEntry `json:"published_rule_versions,omitempty"`
	OverdueSeconds        int64                       `json:"overdue_seconds"`
	IsOverdue             bool                        `json:"is_overdue"`
	IntakeToTriageSeconds int64                       `json:"intake_to_triage_seconds"`
}

// StewardTriageView represents structured data ready for Page Designer UI components
type StewardTriageView struct {
	CaseCode         string                `json:"case_code"`
	Title            string                `json:"title"`
	Source           CaseSource            `json:"source"`
	SourceReference  string                `json:"source_reference"`
	Description      string                `json:"description"`
	DueAt            time.Time             `json:"due_at"`
	Status           CaseStatus            `json:"status"`
	Classification   *CaseClassification   `json:"classification,omitempty"`
	IsEscalated      bool                  `json:"is_escalated"`
	EscalationCount  int                   `json:"escalation_count"`
	AffectedRules    []AffectedRuleSummary `json:"affected_rules"`
	DiffViews        []RuleDiffView        `json:"diff_views"`
	AvailableSignals []string              `json:"available_signals"`
}

// AffectedRuleSummary provides rule context for the steward
type AffectedRuleSummary struct {
	RuleID             uuid.UUID              `json:"rule_id"`
	RuleCode           string                 `json:"rule_code"`
	Name               string                 `json:"name"`
	CurrentVersion     int                    `json:"current_version"`
	LibraryStatus      string                 `json:"library_status"`
	Citation           string                 `json:"citation"`
	Thresholds         map[string]interface{} `json:"thresholds"`
	ProposedThresholds map[string]interface{} `json:"proposed_thresholds,omitempty"`
	ProposedCitation   string                 `json:"proposed_citation,omitempty"`
}

// RuleDiffView provides structure-aware AST diff presentation for Page Designer
type RuleDiffView struct {
	RuleID              uuid.UUID              `json:"rule_id"`
	RuleCode            string                 `json:"rule_code"`
	RuleName            string                 `json:"rule_name"`
	Diff                *drift.ASTSemanticDiff `json:"diff"`
	CurrentAST          map[string]interface{} `json:"current_ast"`
	ProposedAST         map[string]interface{} `json:"proposed_ast"`
	CurrentThresholds   map[string]interface{} `json:"current_thresholds"`
	ProposedThresholds  map[string]interface{} `json:"proposed_thresholds"`
	CurrentCitation     string                 `json:"current_citation"`
	ProposedCitation    string                 `json:"proposed_citation"`
	ProposedContentHash string                 `json:"proposed_content_hash"`
	CorpusPassed        bool                   `json:"corpus_passed"`
	HasBreakingChanges  bool                   `json:"has_breaking_changes"`
}
