package nba

import (
	"time"

	"github.com/google/uuid"
)

type DetectedSignal struct {
	SignalID   uuid.UUID              `json:"signal_id"`
	ClientID   uuid.UUID              `json:"client_id"`
	SignalType string                 `json:"signal_type"`
	Category   string                 `json:"category"`
	Strength   float64                `json:"strength"`
	DetectedAt time.Time              `json:"detected_at"`
	RawData    map[string]interface{} `json:"raw_data"`
	ClientTier string                 `json:"client_tier"` // 'VIP', 'HIGH_NET_WORTH', 'STANDARD'
	ExpiryAt   *time.Time             `json:"expiry_at,omitempty"`
}

type NextBestAction struct {
	ActionID           uuid.UUID              `json:"action_id"`
	ClientID           uuid.UUID              `json:"client_id"`
	ClientName         string                 `json:"client_name"`
	ActionType         string                 `json:"action_type"`
	ActionName         string                 `json:"action_name"`
	Confidence         float64                `json:"confidence"`
	UrgencyScore       float64                `json:"urgency_score"`
	ExpectedValue      float64                `json:"expected_value"`
	SuccessProbability float64                `json:"success_probability"`
	TriggerSignal      string                 `json:"trigger_signal"`
	Reasoning          string                 `json:"reasoning"`
	RecommendedChannel string                 `json:"recommended_channel"`
	DurationMinutes    int                    `json:"estimated_duration_minutes"`
	TemplateContent    map[string]interface{} `json:"template_content"`
}

type ActionOutcome struct {
	OutcomeID         uuid.UUID  `json:"outcome_id"`
	ActionID          uuid.UUID  `json:"action_id"`
	ClientID          uuid.UUID  `json:"client_id"`
	AdvisorID         uuid.UUID  `json:"advisor_id"`
	TriggerSignalType string     `json:"trigger_signal_type"`
	RecommendedAt     time.Time  `json:"recommended_at"`
	ExecutedAt        *time.Time `json:"executed_at"`
	CompletedAt       *time.Time `json:"completed_at"`
	ExecutionChannel  string     `json:"execution_channel"`
	ClientResponded   bool       `json:"client_responded"`
	ActionSuccessful  bool       `json:"action_successful"`
	RevenueGenerated  float64    `json:"revenue_generated"`
	AdvisorFeedback   string     `json:"advisor_feedback"`
	AdvisorRating     int        `json:"advisor_rating"`
}
