package aso

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ASOPolicyStore manages ASO policies with core→tenant inheritance
type ASOPolicyStore interface {
	// GetPolicy retrieves the policy for an env/tenant, falling back to core if no tenant override
	GetPolicy(ctx context.Context, env string, tenantID *uuid.UUID) (*ASOPolicy, error)

	// ListPolicies returns all policies for an environment
	ListPolicies(ctx context.Context, env string) ([]ASOPolicy, error)

	// ListAllPolicies returns all policies across all environments
	ListAllPolicies(ctx context.Context) ([]ASOPolicy, error)

	// UpsertPolicy creates or updates a policy
	UpsertPolicy(ctx context.Context, policy *ASOPolicy) error

	// DeletePolicy removes a tenant override policy (cannot delete core)
	DeletePolicy(ctx context.Context, id uuid.UUID) error

	// GetCorePolicy returns the core policy for an environment
	GetCorePolicy(ctx context.Context, env string) (*ASOPolicy, error)
}

// asoPolicyStore implements ASOPolicyStore
type asoPolicyStore struct {
	db *sqlx.DB
}

// NewASOPolicyStore creates a new policy store
func NewASOPolicyStore(db *sqlx.DB) ASOPolicyStore {
	return &asoPolicyStore{db: db}
}

// GetPolicy retrieves policy with core→tenant inheritance
func (s *asoPolicyStore) GetPolicy(ctx context.Context, env string, tenantID *uuid.UUID) (*ASOPolicy, error) {
	// First try tenant-specific policy
	if tenantID != nil {
		policy, err := s.getByEnvTenant(ctx, env, *tenantID)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to get tenant policy: %w", err)
		}
		if policy != nil {
			return policy, nil
		}
	}

	// Fall back to core policy
	return s.GetCorePolicy(ctx, env)
}

// GetCorePolicy returns the core (tenant_id = NULL) policy for an environment
func (s *asoPolicyStore) GetCorePolicy(ctx context.Context, env string) (*ASOPolicy, error) {
	var policy ASOPolicy
	err := s.db.GetContext(ctx, &policy, `
		SELECT * FROM semantic.aso_policy
		WHERE env = $1 AND tenant_id IS NULL
	`, env)

	if err == sql.ErrNoRows {
		// Return disabled default policy
		return &ASOPolicy{
			Env:                   env,
			Enabled:               false,
			Mode:                  ASOModAdvisory,
			MaxNewPreAggsPerDay:   3,
			MaxChangesPerDay:      10,
			MinScoreForNewPreAgg:  1.0,
			HotPathThresholdMs:    1000,
			LookbackWindowSeconds: 604800, // 7 days
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get core policy: %w", err)
	}

	return &policy, nil
}

// getByEnvTenant retrieves a specific tenant's policy
func (s *asoPolicyStore) getByEnvTenant(ctx context.Context, env string, tenantID uuid.UUID) (*ASOPolicy, error) {
	var policy ASOPolicy
	err := s.db.GetContext(ctx, &policy, `
		SELECT * FROM semantic.aso_policy
		WHERE env = $1 AND tenant_id = $2
	`, env, tenantID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &policy, nil
}

// ListPolicies returns all policies for an environment
func (s *asoPolicyStore) ListPolicies(ctx context.Context, env string) ([]ASOPolicy, error) {
	var policies []ASOPolicy
	err := s.db.SelectContext(ctx, &policies, `
		SELECT * FROM semantic.aso_policy
		WHERE env = $1
		ORDER BY tenant_id NULLS FIRST
	`, env)

	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}

	return policies, nil
}

// ListAllPolicies returns all policies across all environments
func (s *asoPolicyStore) ListAllPolicies(ctx context.Context) ([]ASOPolicy, error) {
	var policies []ASOPolicy
	err := s.db.SelectContext(ctx, &policies, `
		SELECT * FROM semantic.aso_policy
		ORDER BY env, tenant_id NULLS FIRST
	`)

	if err != nil {
		return nil, fmt.Errorf("failed to list all policies: %w", err)
	}

	return policies, nil
}

// UpsertPolicy creates or updates a policy
func (s *asoPolicyStore) UpsertPolicy(ctx context.Context, policy *ASOPolicy) error {
	query := `
		INSERT INTO semantic.aso_policy (
			env, tenant_id, enabled, mode,
			max_new_preaggs_per_day, max_changes_per_day,
			min_score_for_new_preagg, min_usage_for_retirement,
			hot_path_threshold_ms, lookback_window_seconds,
			prewarm_enabled, prewarm_lead_time_minutes,
			created_by, updated_by
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9, $10,
			$11, $12, $13, $13
		)
		ON CONFLICT (env, tenant_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			mode = EXCLUDED.mode,
			max_new_preaggs_per_day = EXCLUDED.max_new_preaggs_per_day,
			max_changes_per_day = EXCLUDED.max_changes_per_day,
			min_score_for_new_preagg = EXCLUDED.min_score_for_new_preagg,
			min_usage_for_retirement = EXCLUDED.min_usage_for_retirement,
			hot_path_threshold_ms = EXCLUDED.hot_path_threshold_ms,
			lookback_window_seconds = EXCLUDED.lookback_window_seconds,
			prewarm_enabled = EXCLUDED.prewarm_enabled,
			prewarm_lead_time_minutes = EXCLUDED.prewarm_lead_time_minutes,
			updated_by = EXCLUDED.updated_by,
			updated_at = now()
		RETURNING id
	`

	err := s.db.GetContext(ctx, &policy.ID, query,
		policy.Env, policy.TenantID, policy.Enabled, policy.Mode,
		policy.MaxNewPreAggsPerDay, policy.MaxChangesPerDay,
		policy.MinScoreForNewPreAgg, policy.MinUsageForRetirement,
		policy.HotPathThresholdMs, policy.LookbackWindowSeconds,
		policy.PrewarmEnabled, policy.PrewarmLeadTimeMinutes,
		policy.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert policy: %w", err)
	}

	return nil
}

// DeletePolicy removes a tenant override policy (cannot delete core)
func (s *asoPolicyStore) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM semantic.aso_policy
		WHERE id = $1 AND tenant_id IS NOT NULL
	`, id)

	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("policy not found or cannot delete core policy")
	}

	return nil
}

// ============================================================================
// Policy Helpers
// ============================================================================

// CanAutoApply checks if the policy allows automatic application
func (p *ASOPolicy) CanAutoApply() bool {
	return p.Enabled && p.Mode == ASOModeAutoApply
}

// CanAutoTune checks if the policy allows automatic tuning
func (p *ASOPolicy) CanAutoTune() bool {
	return p.Enabled && (p.Mode == ASOModeAutoTune || p.Mode == ASOModeAutoApply)
}

// IsAdvisoryOnly checks if the policy only allows recommendations
func (p *ASOPolicy) IsAdvisoryOnly() bool {
	return !p.Enabled || p.Mode == ASOModAdvisory
}

// GetEffectivePolicy returns the effective policy for a tenant,
// merging core with tenant overrides
func GetEffectivePolicy(core, tenant *ASOPolicy) *ASOPolicy {
	if tenant == nil {
		return core
	}
	// Tenant policy takes precedence
	return tenant
}
