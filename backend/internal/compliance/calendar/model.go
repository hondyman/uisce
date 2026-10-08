package calendar

import (
	"time"

	"github.com/google/uuid"
)

// ComplianceCalendarEvent represents a single compliance deadline, filing, or review item.
type ComplianceCalendarEvent struct {
	ID               uuid.UUID      `json:"id"`
	TenantID         uuid.UUID      `json:"tenant_id"`
	EventCode        string         `json:"event_code"`
	Title            string         `json:"title"`
	Description      string         `json:"description"`
	Jurisdiction     string         `json:"jurisdiction"` // US, UK, EU, GLOBAL
	Regulation       string         `json:"regulation"`   // SEC Form 13F, Rule 8.3, EMIR, etc.
	DeadlineType     string         `json:"deadline_type"` // STATUTORY_FILING, MANDATE_ATTESTATION, PORTFOLIO_REVIEW, DISCLOSURE_CUTOFF, REGULATORY_CHANGE
	DueDate          string         `json:"due_date"`     // YYYY-MM-DD
	CutoffTime       string         `json:"cutoff_time"`  // 17:30 EST, 12:00 BST, etc.
	CutoffTimestamp  *time.Time     `json:"cutoff_timestamp,omitempty"`
	DaysRemaining    int            `json:"days_remaining"` // Business days relative to reference date (can be negative if overdue)
	Status           string         `json:"status"`         // UPCOMING, DUE_SOON, OVERDUE, FILED, EXEMPT
	Severity         string         `json:"severity"`       // CRITICAL, HIGH, MEDIUM, LOW
	LegalCitation    string         `json:"legal_citation,omitempty"`
	AffectedAccounts []string       `json:"affected_accounts,omitempty"`
	RuleIDs          []string       `json:"rule_ids,omitempty"`
	Source           string         `json:"source"` // STATUTORY_SCHEDULE, REGULATORY_CASE, SURVEILLANCE_BREACH
	SourceID         string         `json:"source_id,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

// CalendarFilter defines query filters for the compliance calendar.
type CalendarFilter struct {
	TenantID     uuid.UUID
	FromDate     time.Time
	ToDate       time.Time
	Jurisdiction string
	Regulation   string
	DeadlineType string
	Status       string
	ReferenceDay time.Time
}
