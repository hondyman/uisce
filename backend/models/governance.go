package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// NotificationRoutingRule defines how a notification is routed.
type NotificationRoutingRule struct {
	ID           uuid.UUID       `db:"id" json:"id"`
	RuleID       string          `db:"rule_id" json:"rule_id"`             // e.g., "certification_change_alert"
	Trigger      string          `db:"trigger" json:"trigger"`             // e.g., "certification_updated"
	Scope        string          `db:"scope" json:"scope"`                 // e.g., "asset", "domain"
	AssetType    string          `db:"asset_type" json:"asset_type"`       // e.g., "metric", "view"
	RoutingLogic json.RawMessage `db:"routing_logic" json:"routing_logic"` // JSONB of RoutingLogic
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at" json:"updated_at"`
	UpdatedBy    string          `db:"updated_by" json:"updated_by"`
}

// RoutingLogic specifies who to notify and when to escalate.
type RoutingLogic struct {
	Notify     []string              `json:"notify"`  // e.g., ["asset_owner", "domain_steward"]
	Exclude    []string              `json:"exclude"` // e.g., ["users_without_claim"]
	SuppressIf *SuppressionCondition `json:"suppress_if,omitempty"`
	EscalateIf *EscalationCondition  `json:"escalate_if,omitempty"`
	EscalateTo []string              `json:"escalate_to,omitempty"` // e.g., ["governance_reviewers"]
}

// SuppressionCondition defines conditions for suppressing a notification.
type SuppressionCondition struct {
	AssetCertified *bool    `json:"asset_certified,omitempty"`
	ChangeType     []string `json:"change_type,omitempty"`
	RiskScoreLte   *int     `json:"risk_score_lte,omitempty"` // Less than or equal to
}

// EscalationCondition defines conditions for escalating a notification.
type EscalationCondition struct {
	AssetCertified *bool    `json:"asset_certified,omitempty"`
	ChangeType     []string `json:"change_type,omitempty"` // e.g., ["revocation", "definition_change"]
	RiskFlags      []string `json:"risk_flags,omitempty"`
	RiskScoreGte   *int     `json:"risk_score_gte,omitempty"` // Greater than or equal to
}

// SemanticChangeEvent represents a change event to be evaluated by the alert engine.
type SemanticChangeEvent struct {
	AssetID          uuid.UUID       `json:"asset_id"`
	AssetType        string          `json:"asset_type"`
	ChangeType       string          `json:"change_type"` // e.g., "claim_grant", "certification_revoked"
	UserID           string          `json:"user_id"`
	AssetSensitivity string          `json:"asset_sensitivity"` // "low", "medium", "high"
	Details          json.RawMessage `json:"details"`           // e.g., {"permission": "read"}
}

// ClaimSuggestion represents a system-generated suggestion to grant a claim.
type ClaimSuggestion struct {
	ID                  uuid.UUID       `db:"id" json:"id"`
	UserID              string          `db:"user_id" json:"user_id"`
	ModelID             uuid.UUID       `db:"model_id" json:"model_id"`
	SuggestedPermission string          `db:"suggested_permission" json:"suggested_permission"`
	Reason              string          `db:"reason" json:"reason"`     // e.g., "High query frequency"
	Evidence            json.RawMessage `db:"evidence" json:"evidence"` // e.g., {"query_count": 12, "last_queried": "..."}
	Status              string          `db:"status" json:"status"`     // 'new', 'dismissed', 'granted'
	CreatedAt           time.Time       `db:"created_at" json:"created_at"`
}

// ClaimBundle is a reusable template of claims for a specific role.
type ClaimBundle struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	CreatedBy   string    `db:"created_by" json:"created_by"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// ClaimBundleItem is a single permission within a ClaimBundle.
type ClaimBundleItem struct {
	ID         uuid.UUID `db:"id" json:"id"`
	BundleID   uuid.UUID `db:"bundle_id" json:"bundle_id"`
	ModelID    uuid.UUID `db:"model_id" json:"model_id"`
	Permission string    `db:"permission" json:"permission"`
}

// GovernanceHeatmapDataPoint represents aggregated governance metrics for a domain.
type GovernanceHeatmapDataPoint struct {
	Domain                 string  `json:"domain"`
	CertifiedModelPercent  float64 `json:"certified_model_percent"`
	ClaimDensity           int     `json:"claim_density"` // Total number of active claims
	RiskyClaimCount        int     `json:"risky_claim_count"`
	UnresolvedRequestCount int     `json:"unresolved_request_count"`
	ClaimDriftCount        int     `json:"claim_drift_count"`
}

// ClaimConflict represents an identified conflict between two or more claims.
type ClaimConflict struct {
	ID               uuid.UUID       `db:"id" json:"id"`
	UserID           string          `db:"user_id" json:"user_id"`
	ModelID          uuid.UUID       `db:"model_id" json:"model_id"`
	ConflictType     string          `db:"conflict_type" json:"conflict_type"` // 'overlap', 'contradiction'
	Details          json.RawMessage `db:"details" json:"details"`             // JSON of { conflicting_claims: [claim1, claim2], description: "..." }
	DetectedAt       time.Time       `db:"detected_at" json:"detected_at"`
	Status           string          `db:"status" json:"status"` // 'new', 'resolved'
	ResolutionAction *string         `db:"resolution_action" json:"resolution_action,omitempty"`
}

// AccessDecisionLog records every decision made by the evaluation engine.
type AccessDecisionLog struct {
	ID          uuid.UUID `db:"id" json:"id"`
	UserID      string    `db:"user_id" json:"user_id"`
	AssetID     string    `db:"asset_id" json:"asset_id"`
	Action      string    `db:"action" json:"action"`
	Decision    string    `db:"decision" json:"decision"`
	Reason      string    `db:"reason" json:"reason"`
	EvaluatedAt time.Time `db:"evaluated_at" json:"evaluated_at"`
}

// AccessDecisionTrace stores the detailed context for a single access decision.
type AccessDecisionTrace struct {
	ID              uuid.UUID       `db:"id" json:"id"`
	DecisionLogID   uuid.UUID       `db:"decision_log_id" json:"decision_log_id"`
	UserID          string          `db:"user_id" json:"user_id"`
	AssetID         string          `db:"asset_id" json:"asset_id"`
	Action          string          `db:"action" json:"action"`
	Decision        string          `db:"decision" json:"decision"`
	EvaluatedClaims json.RawMessage `db:"evaluated_claims" json:"evaluated_claims"` // JSONB of claims considered
	MatchedPolicies json.RawMessage `db:"matched_policies" json:"matched_policies"` // JSONB of policies triggered
	TenantScope     string          `db:"tenant_scope" json:"tenant_scope"`
	Reason          string          `db:"reason" json:"reason"` // Human-readable summary
	EvaluatedAt     time.Time       `db:"evaluated_at" json:"evaluated_at"`
}

// GovernanceHealthScore represents the composite health score.
type GovernanceHealthScore struct {
	Score             float64 `json:"score"`
	CertifiedCoverage float64 `json:"certified_coverage"`
	ClaimAlignment    float64 `json:"claim_alignment"`
	UsageCoverage     float64 `json:"usage_coverage"`
	RiskExposure      float64 `json:"risk_exposure"`
}

// GovernanceCockpitSnapshot aggregates data for the main governance dashboard.
type GovernanceCockpitSnapshot struct {
	ID                    uuid.UUID             `json:"id"`
	TenantID              string                `json:"tenant_id"`
	Timestamp             time.Time             `json:"timestamp"`
	HealthScore           GovernanceHealthScore `json:"health_score"`
	ActiveClaimsCount     int                   `json:"active_claims_count"`
	ConflictCount         int                   `json:"conflict_count"`
	DriftCount            int                   `json:"drift_count"`
	TenantIsolationStatus string                `json:"tenant_isolation_status"` // e.g., "healthy", "at_risk"
	RecentDecisions       []AccessDecisionTrace `json:"recent_decisions"`
	PolicyCount           int                   `json:"policy_count"`
	SimulationCount       int                   `json:"simulation_count"`
	SuppressedAlertCount  int                   `json:"suppressed_alert_count"`
	EscalatedAlertCount   int                   `json:"escalated_alert_count"`
	AutomationStatus      string                `json:"automation_status"` // e.g., "running", "paused"
	AutoResolvedCount     int                   `json:"auto_resolved_count"`
}

// AutomationPolicy defines a rule for the automation engine.
type AutomationPolicy struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	PolicyID    string          `db:"policy_id" json:"policy_id"` // e.g., "auto_expire_drifted_claims"
	Description string          `db:"description" json:"description"`
	Trigger     string          `db:"trigger" json:"trigger"`           // e.g., "claim_drift_detected"
	Conditions  json.RawMessage `db:"conditions" json:"conditions"`     // JSONB of conditions, e.g., {"inactive_days": 60}
	Action      string          `db:"action" json:"action"`             // e.g., "auto_expire"
	ActionParam json.RawMessage `db:"action_param" json:"action_param"` // JSONB of action parameters
	IsEnabled   bool            `db:"is_enabled" json:"is_enabled"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
}

// AutomationLog records an action taken by the automation engine.
type AutomationLog struct {
	ID         uuid.UUID       `db:"id" json:"id"`
	Timestamp  time.Time       `db:"timestamp" json:"timestamp"`
	PolicyID   string          `db:"policy_id" json:"policy_id"`
	Action     string          `db:"action" json:"action"` // e.g., "claim_expired", "conflict_resolved"
	TargetType string          `db:"target_type" json:"target_type"`
	TargetID   string          `db:"target_id" json:"target_id"`
	Details    json.RawMessage `db:"details" json:"details"` // JSONB of details, e.g., {"reason": "inactive for 90 days"}
	Status     string          `db:"status" json:"status"`   // e.g., "success", "failed", "undone"
	UndoneBy   *string         `db:"undone_by" json:"undone_by,omitempty"`
	UndoneAt   *time.Time      `db:"undone_at" json:"undone_at,omitempty"`
}

// ProposedClaim is a simplified claim for simulation or evaluation purposes.
type ProposedClaim struct {
	ModelID    uuid.UUID `json:"model_id"`
	Permission string    `json:"permission"`
}

// GuardrailRule defines a proactive check before a claim is requested or granted.
type GuardrailRule struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	RuleID      string          `db:"rule_id" json:"rule_id"` // e.g., "certified_update_requires_approval"
	Description string          `db:"description" json:"description"`
	Trigger     string          `db:"trigger" json:"trigger"`       // e.g., "claim_request", "claim_grant"
	Conditions  json.RawMessage `db:"conditions" json:"conditions"` // JSONB of GuardrailConditions
	Actions     pq.StringArray  `db:"actions" json:"actions"`       // e.g., ["block", "escalate_to_steward", "require_justification"]
	IsEnabled   bool            `db:"is_enabled" json:"is_enabled"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
}

// GuardrailConditions defines the conditions for a rule to match.
type GuardrailConditions struct {
	AssetCertified *bool    `json:"asset_certified,omitempty"`
	Permission     []string `json:"permission,omitempty"` // e.g., ["update", "delete"]
	AssetDomain    []string `json:"asset_domain,omitempty"`
	UserRole       []string `json:"user_role,omitempty"`
}

// GuardrailViolation records an instance where a proposed action was flagged by a guardrail.
type GuardrailViolation struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	Timestamp      time.Time       `db:"timestamp" json:"timestamp"`
	UserID         string          `db:"user_id" json:"user_id"`
	ProposedClaim  json.RawMessage `db:"proposed_claim" json:"proposed_claim"` // JSONB of the proposed claim
	ViolatedRuleID string          `db:"violated_rule_id" json:"violated_rule_id"`
	ActionTaken    string          `db:"action_taken" json:"action_taken"` // e.g., "blocked", "escalated"
	RiskScore      int             `db:"risk_score" json:"risk_score"`
	Details        string          `db:"details" json:"details"`
}

// EvaluateGuardrailRequest is the payload for the guardrail evaluation endpoint.
type EvaluateGuardrailRequest struct {
	UserID        string        `json:"user_id" binding:"required"`
	ProposedClaim ProposedClaim `json:"proposed_claim" binding:"required"`
}

// EvaluateGuardrailResponse is the result of a guardrail evaluation.
type EvaluateGuardrailResponse struct {
	Decision     string   `json:"decision"` // "allow", "block", "escalate"
	Reason       string   `json:"reason"`
	RiskScore    int      `json:"risk_score"`
	ViolatedRule *string  `json:"violated_rule,omitempty"`
	NextSteps    []string `json:"next_steps,omitempty"` // e.g., ["Contact data steward for approval"]
}
