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
	ClassificationParameterChange    CaseClassification = "PARAMETER_CHANGE"
	ClassificationSemanticChange     CaseClassification = "SEMANTIC_CHANGE"
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
	RuleID      uuid.UUID `json:"rule_id"`
	RuleCode    string    `json:"rule_code"`
	FromVersion int       `json:"from_version"`
	ToVersion   int       `json:"to_version"`
	ContentHash string    `json:"content_hash"`
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
	PublishedRuleVersions []PublishedRuleVersionEntry `json:"published_rule_versions,omitempty"`
	DueAt                 time.Time                   `json:"due_at"`
	CreatedBy             string                      `json:"created_by"`
	CreatedAt             time.Time                   `json:"created_at"`
	UpdatedAt             time.Time                   `json:"updated_at"`
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

// StewardTriageView represents structured data ready for Page Designer UI components
type StewardTriageView struct {
	CaseCode         string                 `json:"case_code"`
	Title            string                 `json:"title"`
	Source           CaseSource             `json:"source"`
	SourceReference  string                 `json:"source_reference"`
	Description      string                 `json:"description"`
	DueAt            time.Time              `json:"due_at"`
	Status           CaseStatus             `json:"status"`
	AffectedRules    []AffectedRuleSummary  `json:"affected_rules"`
	DiffViews        []RuleDiffView         `json:"diff_views"`
	AvailableSignals []string               `json:"available_signals"`
}

// AffectedRuleSummary provides rule context for the steward
type AffectedRuleSummary struct {
	RuleID         uuid.UUID              `json:"rule_id"`
	RuleCode       string                 `json:"rule_code"`
	Name           string                 `json:"name"`
	CurrentVersion int                    `json:"current_version"`
	LibraryStatus  string                 `json:"library_status"`
	Citation       string                 `json:"citation"`
	Thresholds     map[string]interface{} `json:"thresholds"`
}

// RuleDiffView provides structure-aware AST diff presentation for Page Designer
type RuleDiffView struct {
	RuleID             uuid.UUID              `json:"rule_id"`
	RuleCode           string                 `json:"rule_code"`
	Diff               *drift.ASTSemanticDiff `json:"diff"`
	OldThresholds      map[string]interface{} `json:"old_thresholds"`
	NewThresholds      map[string]interface{} `json:"new_thresholds"`
	OldCitation        string                 `json:"old_citation"`
	NewCitation        string                 `json:"new_citation"`
	HasBreakingChanges bool                   `json:"has_breaking_changes"`
}
