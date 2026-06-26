package values

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ValueTheme struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type ValueSignalSourceType string

const (
	SourceTypeVendor   ValueSignalSourceType = "VENDOR"
	SourceTypeNews     ValueSignalSourceType = "NEWS"
	SourceTypeInternal ValueSignalSourceType = "INTERNAL"
)

type ValueSignalSource struct {
	ID               uuid.UUID             `json:"id" db:"id"`
	Name             string                `json:"name" db:"name"`
	Type             ValueSignalSourceType `json:"type" db:"type"`
	ReliabilityScore float64               `json:"reliability_score" db:"reliability_score"`
	CreatedAt        time.Time             `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at" db:"updated_at"`
}

type ValueSignalStatus string

const (
	SignalStatusActive      ValueSignalStatus = "ACTIVE"
	SignalStatusUnderReview ValueSignalStatus = "UNDER_REVIEW"
	SignalStatusExpired     ValueSignalStatus = "EXPIRED"
)

type EvidenceRef struct {
	URL     string `json:"url"`
	Date    string `json:"date"`
	Type    string `json:"type"`
	Summary string `json:"summary"`
}

type ValueSignal struct {
	ID           uuid.UUID         `json:"id" db:"id"`
	IssuerID     string            `json:"issuer_id" db:"issuer_id"`
	InstrumentID *string           `json:"instrument_id,omitempty" db:"instrument_id"`
	ThemeID      uuid.UUID         `json:"theme_id" db:"theme_id"`
	SourceID     uuid.UUID         `json:"source_id" db:"source_id"`
	Score        float64           `json:"score" db:"score"`
	Summary      string            `json:"summary" db:"summary"`
	EvidenceRefs []EvidenceRef     `json:"evidence_refs" db:"evidence_refs"` // Stored as JSONB
	Status       ValueSignalStatus `json:"status" db:"status"`
	Confidence   float64           `json:"confidence" db:"confidence"`
	ValidFrom    time.Time         `json:"valid_from" db:"valid_from"`
	ValidUntil   *time.Time        `json:"valid_until,omitempty" db:"valid_until"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
}

type StrategyTemplate struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	Name          string          `json:"name" db:"name"`
	Description   string          `json:"description" db:"description"`
	BasePolicyIDs json.RawMessage `json:"base_policy_ids" db:"base_policy_ids"` // JSONB list of IDs
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

type ClientValuesProfile struct {
	ID                 uuid.UUID       `json:"id" db:"id"`
	ClientID           string          `json:"client_id" db:"client_id"`
	StrategyTemplateID *uuid.UUID      `json:"strategy_template_id,omitempty" db:"strategy_template_id"`
	Preferences        json.RawMessage `json:"preferences" db:"preferences"` // JSONB
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"`
}

type ConstraintOperator string

const (
	OperatorExclude     ConstraintOperator = "EXCLUDE"
	OperatorUnderweight ConstraintOperator = "UNDERWEIGHT"
	OperatorOverweight  ConstraintOperator = "OVERWEIGHT"
	OperatorRequire     ConstraintOperator = "REQUIRE"
	OperatorCapExposure ConstraintOperator = "CAP_EXPOSURE"
)

type ConstraintSeverity string

const (
	SeverityLow      ConstraintSeverity = "LOW"
	SeverityMedium   ConstraintSeverity = "MEDIUM"
	SeverityHigh     ConstraintSeverity = "HIGH"
	SeverityCritical ConstraintSeverity = "CRITICAL"
)

type ConstraintScope struct {
	BenchmarkID  string `json:"benchmark_id,omitempty"`
	Region       string `json:"region,omitempty"`
	Sector       string `json:"sector,omitempty"`
	Issuer       string `json:"issuer,omitempty"`
	InstrumentID string `json:"instrument_id,omitempty"`
}

type Constraint struct {
	ID                    uuid.UUID          `json:"id" db:"id"`
	ClientValuesProfileID *uuid.UUID         `json:"client_values_profile_id,omitempty" db:"client_values_profile_id"`
	StrategyTemplateID    *uuid.UUID         `json:"strategy_template_id,omitempty" db:"strategy_template_id"`
	Name                  string             `json:"name" db:"name"`
	Scope                 ConstraintScope    `json:"scope" db:"scope"` // JSONB
	Operator              ConstraintOperator `json:"operator" db:"operator"`
	Condition             json.RawMessage    `json:"condition" db:"condition"` // JSONB
	Severity              ConstraintSeverity `json:"severity" db:"severity"`
	Priority              int                `json:"priority" db:"priority"`
	CreatedAt             time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time          `json:"updated_at" db:"updated_at"`
}
