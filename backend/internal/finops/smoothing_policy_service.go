package finops

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// WorkloadSmoothingPolicy mirrors the finops.workload_smoothing_policies table row.
type WorkloadSmoothingPolicy struct {
	PolicyID                    uuid.UUID `db:"policy_id"                      json:"policyId"`
	TenantID                    uuid.UUID `db:"tenant_id"                      json:"tenantId"`
	PolicyName                  string    `db:"policy_name"                    json:"policyName"`
	IsActive                    bool      `db:"is_active"                      json:"isActive"`
	OffPeakCron                 string    `db:"off_peak_cron"                  json:"offPeakCron"`
	PrewarmThresholdMultiplier  float64   `db:"prewarm_threshold_multiplier"   json:"prewarmThresholdMultiplier"`
	EnableBurstDeferral         bool      `db:"enable_burst_deferral"          json:"enableBurstDeferral"`
	MaxDeferralMinutes          int       `db:"max_deferral_minutes"           json:"maxDeferralMinutes"`
	MinPeakProbabilityToPrewarm float64   `db:"min_peak_probability_to_prewarm" json:"minPeakProbabilityToPrewarm"`
	CreatedAt                   time.Time `db:"created_at"                     json:"createdAt"`
	UpdatedAt                   time.Time `db:"updated_at"                     json:"updatedAt"`
}

// defaultPolicy is the safe fallback used when no tenant policy has been configured.
var defaultPolicy = WorkloadSmoothingPolicy{
	PolicyName:                  "DEFAULT",
	IsActive:                    true,
	OffPeakCron:                 "0 2 * * *",
	PrewarmThresholdMultiplier:  2.50,
	EnableBurstDeferral:         true,
	MaxDeferralMinutes:          180,
	MinPeakProbabilityToPrewarm: 0.700,
}

// SmoothingPolicyService manages CRUD operations for workload smoothing policies.
// It implements the Config-Before-Code principle (Rule 1): all runtime thresholds
// are read from finops.workload_smoothing_policies rather than hard-coded in Go.
type SmoothingPolicyService struct {
	db *sqlx.DB
}

// NewSmoothingPolicyService constructs a SmoothingPolicyService.
func NewSmoothingPolicyService(db *sqlx.DB) *SmoothingPolicyService {
	return &SmoothingPolicyService{db: db}
}

// GetActivePolicy retrieves the active smoothing policy for a tenant.
// If no policy exists, the built-in defaultPolicy is returned so that callers
// always receive a valid configuration without defensive nil-checks.
func (s *SmoothingPolicyService) GetActivePolicy(
	ctx context.Context,
	tenantID uuid.UUID,
) (*WorkloadSmoothingPolicy, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	if s.db == nil {
		p := defaultPolicy
		p.TenantID = tenantID
		return &p, nil
	}

	var policy WorkloadSmoothingPolicy
	err := s.db.GetContext(ctx, &policy, `
		SELECT policy_id, tenant_id, policy_name, is_active,
		       off_peak_cron, prewarm_threshold_multiplier,
		       enable_burst_deferral, max_deferral_minutes,
		       min_peak_probability_to_prewarm, created_at, updated_at
		FROM finops.workload_smoothing_policies
		WHERE tenant_id = $1
		  AND is_active = TRUE
		ORDER BY created_at DESC
		LIMIT 1;
	`, tenantID)
	if err != nil {
		// No row → return the safe default rather than propagating a not-found error.
		p := defaultPolicy
		p.TenantID = tenantID
		return &p, nil
	}

	return &policy, nil
}

// UpsertPolicy creates or updates a smoothing policy for a tenant.
// Uses the (tenant_id, policy_name) unique constraint for idempotent upsert.
func (s *SmoothingPolicyService) UpsertPolicy(
	ctx context.Context,
	policy *WorkloadSmoothingPolicy,
) (*WorkloadSmoothingPolicy, error) {
	if policy.TenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}
	if policy.PolicyName == "" {
		return nil, fmt.Errorf("policy_name must not be empty")
	}
	if policy.MinPeakProbabilityToPrewarm < 0 || policy.MinPeakProbabilityToPrewarm > 1 {
		return nil, fmt.Errorf("min_peak_probability_to_prewarm must be between 0.000 and 1.000")
	}
	if policy.MaxDeferralMinutes < 0 {
		return nil, fmt.Errorf("max_deferral_minutes must be >= 0")
	}

	if s.db == nil {
		policy.PolicyID = uuid.New()
		policy.CreatedAt = time.Now()
		policy.UpdatedAt = time.Now()
		return policy, nil
	}

	var result WorkloadSmoothingPolicy
	err := s.db.GetContext(ctx, &result, `
		INSERT INTO finops.workload_smoothing_policies (
			tenant_id, policy_name, is_active,
			off_peak_cron, prewarm_threshold_multiplier,
			enable_burst_deferral, max_deferral_minutes,
			min_peak_probability_to_prewarm
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant_id, policy_name)
		DO UPDATE SET
			is_active                       = EXCLUDED.is_active,
			off_peak_cron                   = EXCLUDED.off_peak_cron,
			prewarm_threshold_multiplier    = EXCLUDED.prewarm_threshold_multiplier,
			enable_burst_deferral           = EXCLUDED.enable_burst_deferral,
			max_deferral_minutes            = EXCLUDED.max_deferral_minutes,
			min_peak_probability_to_prewarm = EXCLUDED.min_peak_probability_to_prewarm,
			updated_at                      = NOW()
		RETURNING policy_id, tenant_id, policy_name, is_active,
		          off_peak_cron, prewarm_threshold_multiplier,
		          enable_burst_deferral, max_deferral_minutes,
		          min_peak_probability_to_prewarm, created_at, updated_at;
	`,
		policy.TenantID,
		policy.PolicyName,
		policy.IsActive,
		policy.OffPeakCron,
		policy.PrewarmThresholdMultiplier,
		policy.EnableBurstDeferral,
		policy.MaxDeferralMinutes,
		policy.MinPeakProbabilityToPrewarm,
	)
	if err != nil {
		return nil, fmt.Errorf("failed upserting smoothing policy: %w", err)
	}

	return &result, nil
}

// ListPolicies returns all policies (active and inactive) for a tenant.
func (s *SmoothingPolicyService) ListPolicies(
	ctx context.Context,
	tenantID uuid.UUID,
) ([]WorkloadSmoothingPolicy, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}
	if s.db == nil {
		return nil, nil
	}

	var policies []WorkloadSmoothingPolicy
	err := s.db.SelectContext(ctx, &policies, `
		SELECT policy_id, tenant_id, policy_name, is_active,
		       off_peak_cron, prewarm_threshold_multiplier,
		       enable_burst_deferral, max_deferral_minutes,
		       min_peak_probability_to_prewarm, created_at, updated_at
		FROM finops.workload_smoothing_policies
		WHERE tenant_id = $1
		ORDER BY created_at DESC;
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed listing smoothing policies: %w", err)
	}
	return policies, nil
}
