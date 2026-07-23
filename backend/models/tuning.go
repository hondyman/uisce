package models

import (
	"time"

	"github.com/google/uuid"
)

// RuleTuningStatus represents the current tuning status of a single rule.
type RuleTuningStatus struct {
	RuleConfig
	Overrides     []RuleModelConfig `json:"overrides"`
	ApprovalRate  float64           `json:"approval_rate"`
	RejectionRate float64           `json:"rejection_rate"`
}

// RuleConfig represents the global configuration for a single tuning rule.
type RuleConfig struct {
	RuleID            string    `db:"rule_id" json:"rule_id"`
	Name              string    `db:"name" json:"name"`
	Description       string    `db:"description" json:"description"`
	Enabled           bool      `db:"enabled" json:"enabled"`
	Aggressiveness    float64   `db:"aggressiveness" json:"aggressiveness"`
	AutoAcceptDefault bool      `db:"auto_accept_default" json:"auto_accept_default"`
	LastTuned         time.Time `db:"last_tuned" json:"last_tuned"`
	UpdatedBy         string    `db:"updated_by" json:"updated_by"`
}

// RuleModelConfig represents a model-specific override for a tuning rule.
type RuleModelConfig struct {
	RuleID         string    `db:"rule_id" json:"rule_id"`
	ModelName      string    `db:"model_name" json:"model_name"`
	Enabled        bool      `db:"enabled" json:"enabled"`
	Aggressiveness *float64  `db:"aggressiveness" json:"aggressiveness,omitempty"`
	LastTuned      time.Time `db:"last_tuned" json:"last_tuned"`
	UpdatedBy      string    `db:"updated_by" json:"updated_by"`
}

// RuleConfigChangelog represents a single entry in the changelog for a rule.
type RuleConfigChangelog struct {
	ID                uuid.UUID `db:"id" json:"id"`
	RuleID            string    `db:"rule_id" json:"rule_id"`
	OldAggressiveness *float64  `db:"old_aggressiveness" json:"old_aggressiveness,omitempty"`
	NewAggressiveness *float64  `db:"new_aggressiveness" json:"new_aggressiveness,omitempty"`
	OldAutoAccept     *bool     `db:"old_auto_accept" json:"old_auto_accept,omitempty"`
	NewAutoAccept     *bool     `db:"new_auto_accept" json:"new_auto_accept,omitempty"`
	Scope             string    `db:"scope" json:"scope"`
	Reason            string    `db:"reason" json:"reason"`
	TriggeredBy       string    `db:"triggered_by" json:"triggered_by"`
	TriggeredAt       time.Time `db:"triggered_at" json:"triggered_at"`
}

// TuningThresholds holds the thresholds for generating tuning proposals.
type TuningThresholds struct {
	RejectRateDisable     float64 `json:"reject_rate_disable"`
	ApproveRateAutoAccept float64 `json:"approve_rate_auto_accept"`
}

// TuningProposal represents a single tuning suggestion for a rule.
type TuningProposal struct {
	RuleID                 string              `json:"rule_id"`
	Scope                  string              `json:"scope"`
	CurrentAggressiveness  float64             `json:"current_aggressiveness"`
	ProposedAggressiveness float64             `json:"proposed_aggressiveness"`
	CurrentAutoAccept      bool                `json:"current_auto_accept"`
	ProposedAutoAccept     bool                `json:"proposed_auto_accept"`
	Reason                 string              `json:"reason"`
	Metrics                ProposalMetrics     `json:"metrics"`
	ImpactPreview          *ImpactPreview      `json:"impact_preview,omitempty"`
	SideBySide             []SideBySidePreview `json:"side_by_side,omitempty"`
}

// ProposalMetrics holds the metrics that support a tuning proposal.
type ProposalMetrics struct {
	ApprovalRate  float64 `json:"approval_rate"`
	RejectionRate float64 `json:"rejection_rate"`
	TotalChanges  int     `json:"total_changes"`
}

// TuningSimulationRequest is the request body for the tuning simulation endpoint.
type TuningSimulationRequest struct {
	LookbackDays int               `json:"lookback_days"`
	RuleIDs      []string          `json:"rule_ids"`
	Scope        string            `json:"scope"`
	Thresholds   *TuningThresholds `json:"thresholds,omitempty"`
	WithPreview  bool              `json:"with_preview"`
}

// TuningSimulationResponse is the response from the tuning simulation endpoint.
type TuningSimulationResponse struct {
	SimulationID string            `json:"simulation_id"`
	Proposals    []*TuningProposal `json:"proposals"`
}

// ImpactPreview estimates the impact of a rule change.
type ImpactPreview struct {
	ModelsAffected []string `json:"models_affected"`
	FieldsAffected int      `json:"fields_affected"`
}

// SideBySidePreview shows a comparison of the old and new behavior for a rule.
type SideBySidePreview struct {
	Model       string             `json:"model"`
	BeforeYAML  string             `json:"before_yaml"`
	AfterYAML   string             `json:"after_yaml"`
	Annotations []ChangeAnnotation `json:"annotations,omitempty"`
}
