package ops

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of operational event
type EventType string

const (
	EventAlert          EventType = "alert"
	EventFingerprint    EventType = "fingerprint"
	EventTenantHealth   EventType = "tenant_health"
	EventEndpointHealth EventType = "endpoint_health"
	EventLatencyAnomaly EventType = "latency_anomaly"
	EventIncidentOpened EventType = "incident_opened"
	EventIncidentClosed EventType = "incident_closed"
	EventActionExecuted EventType = "action_executed"
)

// Severity represents event severity level
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Event represents a single operational event in the timeline
type Event struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	IncidentID    *uuid.UUID      `json:"incident_id,omitempty" db:"incident_id"`
	EventType     EventType       `json:"event_type" db:"event_type"`
	Scope         string          `json:"scope" db:"scope"` // "global" | "tenant" | "endpoint" | "region"
	TenantID      *uuid.UUID      `json:"tenant_id,omitempty" db:"tenant_id"`
	EndpointPath  *string         `json:"endpoint_path,omitempty" db:"endpoint_path"`
	Region        *string         `json:"region,omitempty" db:"region"`
	FingerprintID *uuid.UUID      `json:"fingerprint_id,omitempty" db:"fingerprint_id"`
	AlertID       *uuid.UUID      `json:"alert_id,omitempty" db:"alert_id"`
	Severity      Severity        `json:"severity" db:"severity"`
	Title         string          `json:"title" db:"title"`
	Details       json.RawMessage `json:"details" db:"details"`
	OccurredAt    time.Time       `json:"occurred_at" db:"occurred_at"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
}

// Incident represents a grouped set of related events
type Incident struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	Region    *string    `json:"region,omitempty" db:"region"` // Geographic region (us-east-1, eu-west-1, etc.)
	Status    string     `json:"status" db:"status"`           // "open" | "closed"
	Severity  Severity   `json:"severity" db:"severity"`
	Title     string     `json:"title" db:"title"`
	Summary   *string    `json:"summary,omitempty" db:"summary"`
	RootCause *string    `json:"root_cause,omitempty" db:"root_cause"`
	StartedAt time.Time  `json:"started_at" db:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty" db:"ended_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	Events    []Event    `json:"events,omitempty" db:"-"` // populated by GetIncident
}

// IncidentResponse wraps incident with related events for API responses
type IncidentResponse struct {
	Incident *Incident `json:"incident"`
	Events   []Event   `json:"events"`
}

// TimelineResponse wraps events for API responses
type TimelineResponse struct {
	Events []Event `json:"events"`
	Total  int     `json:"total"`
}
