package domain

import (
	"context"
	"errors"
	"time"
)

// Common errors
var (
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
	ErrInvalidRequest     = errors.New("invalid request")
)

// Core types for governance and access control

type Permission string

const (
	PermRead   Permission = "read"
	PermWrite  Permission = "write"
	PermUpdate Permission = "update"
	PermDelete Permission = "delete"
	PermAdmin  Permission = "admin"
)

type EvaluationRequest struct {
	UserID   string
	TenantID string
	AssetID  string
	Action   Permission
	Context  map[string]any
}

type EffectiveClaim struct {
	AssetID    string
	Permission Permission
	Scope      []string
	Source     string
	ExpiresAt  *time.Time
}

type PolicyRule struct {
	ID          string
	TenantID    string
	Name        string
	Description string
	Conditions  map[string]any
	Actions     []string
	Priority    int
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PolicyCheckResult struct {
	Allow   bool
	Reason  string
	Matched []map[string]any
	Scopes  []string
}

// Interfaces
type Evaluator interface {
	Evaluate(ctx context.Context, req EvaluationRequest) (bool, string, []EffectiveClaim, error)
}

type PolicyChecker interface {
	Check(ctx context.Context, req EvaluationRequest, claims []EffectiveClaim) (bool, string, []map[string]any, []string, error)
}

type PolicyRepo interface {
	ActivePolicies(ctx context.Context, tenantID string) ([]PolicyRule, error)
}

// Implementations
type SimpleEvaluator struct {
	service ClaimRepo
}

func NewSimpleEvaluator(service ClaimRepo) *SimpleEvaluator {
	return &SimpleEvaluator{service: service}
}

func (e *SimpleEvaluator) Evaluate(ctx context.Context, req EvaluationRequest) (bool, string, []EffectiveClaim, error) {
	if e.service == nil {
		return false, "no claim service configured", nil, ErrInvalidRequest
	}

	claims, err := e.service.EffectiveClaims(ctx, req.UserID, req.TenantID, req.AssetID)
	if err != nil {
		return false, "failed to resolve claims", nil, err
	}

	// Allow only if there exists an effective claim matching requested action for the asset
	for _, c := range claims {
		if c.AssetID == req.AssetID && c.Permission == req.Action {
			return true, "allowed by effective claim", claims, nil
		}
	}

	return false, "no matching effective claim", claims, nil
}

type SimplePolicyChecker struct {
	service interface{} // Will be AccessIntelligenceService
}

func NewSimplePolicyChecker(service interface{}) *SimplePolicyChecker {
	return &SimplePolicyChecker{service: service}
}

func (c *SimplePolicyChecker) Check(ctx context.Context, req EvaluationRequest, claims []EffectiveClaim) (bool, string, []map[string]any, []string, error) {
	// Baseline policy: allow if there's a matching claim
	for _, claim := range claims {
		if claim.Permission == req.Action {
			return true, "Allowed by effective claim", []map[string]any{{"policyId": "baseline_allow_read_or_action", "result": "pass"}}, claim.Scope, nil
		}
	}

	return false, "No effective claim for requested action", []map[string]any{{"policyId": "missing_claim", "result": "fail"}}, nil, nil
}

// ClaimRepo interface (assuming it's defined elsewhere)
type ClaimRepo interface {
	EffectiveClaims(ctx context.Context, userID, tenantID, assetID string) ([]EffectiveClaim, error)
}
