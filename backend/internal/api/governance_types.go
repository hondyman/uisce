package api

import "github.com/hondyman/semlayer/backend/internal/domain"

// AccessEvaluationRequest represents an access evaluation request
type AccessEvaluationRequest struct {
	UserID   string                 `json:"user_id" binding:"required"`
	TenantID string                 `json:"tenant_id" binding:"required"`
	AssetID  string                 `json:"asset_id" binding:"required"`
	Action   string                 `json:"action" binding:"required,oneof=read write update delete"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

// AccessEvaluationResponse represents the response from access evaluation
type AccessEvaluationResponse struct {
	Allowed         bool                     `json:"allowed"`
	Reason          string                   `json:"reason"`
	Claims          []domain.EffectiveClaim  `json:"claims,omitempty"`
	Scopes          []string                 `json:"scopes,omitempty"`
	MatchedPolicies []map[string]interface{} `json:"matched_policies,omitempty"`
}

// PolicyValidationRequest represents a policy validation request
type PolicyValidationRequest struct {
	UserID   string                 `json:"user_id" binding:"required"`
	TenantID string                 `json:"tenant_id" binding:"required"`
	AssetID  string                 `json:"asset_id" binding:"required"`
	Action   string                 `json:"action" binding:"required"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

// PolicyValidationResponse represents the response from policy validation
type PolicyValidationResponse struct {
	Valid       bool     `json:"valid"`
	Violations  []string `json:"violations,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
}
