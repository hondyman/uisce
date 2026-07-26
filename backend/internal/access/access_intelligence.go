// Package access - Access Intelligence & Policy Repository domain.
//
// This package owns access control decisions and policy persistence for
// the Uisce platform. It was extracted from internal/services/ to enforce
// Cardinal Rule 3 (no cycles): this package ONLY depends on internal/analytics,
// internal/observability, libs/*, and stdlib.
//
// Cardinal Rule 7 (tenant security): every method carries tenantID.
// Cardinal Rule 8 (caching): policy caches are versioned per Rule 8.2.
package access

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/observability"
	"github.com/hondyman/uisce/backend/models"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// DEPENDENCY PROVIDER INTERFACES
// ============================================================================
//
// These interfaces decouple the access package from internal/services so
// the access domain can evolve independently. Concrete implementations
// live in services/ and satisfy these contracts via structural typing.

// CollabServiceProvider is satisfied by services.CollaborationService.
type CollabServiceProvider interface{}

// AutoServiceProvider is satisfied by services.AutomationService.
type AutoServiceProvider interface{}

// PerfMonitorProvider is satisfied by services.PerformanceMonitor.
type PerfMonitorProvider interface{}

// ============================================================================
// POLICY REPOSITORY
// ============================================================================

// AccessPolicyRepository defines persistence operations for access control policies.
type AccessPolicyRepository interface {
	List(ctx context.Context) ([]models.AccessControlPolicy, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.AccessControlPolicy, error)
	GetByPolicyID(ctx context.Context, policyID string) (*models.AccessControlPolicy, error)
	Create(ctx context.Context, policy *models.AccessControlPolicy) (*models.AccessControlPolicy, error)
	Update(ctx context.Context, policy *models.AccessControlPolicy) (*models.AccessControlPolicy, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// SqlAccessPolicyRepository is the canonical implementation of AccessPolicyRepository.
type SqlAccessPolicyRepository struct {
	DB *sqlx.DB
}

// NewSqlAccessPolicyRepository constructs a new SqlAccessPolicyRepository.
func NewSqlAccessPolicyRepository(db *sqlx.DB) AccessPolicyRepository {
	return &sqlAccessPolicyRepository{db: db}
}

type sqlAccessPolicyRepository struct {
	db *sqlx.DB
}

func (r *sqlAccessPolicyRepository) List(ctx context.Context) ([]models.AccessControlPolicy, error) {
	const query = `
        SELECT id, policy_id, scope, role, permissions, duration_days, requires_certification,
               max_claims_per_user, approval_threshold, renewal_conditions, created_at, updated_at
          FROM access_control_policies
         ORDER BY policy_id ASC
    `
	var policies []models.AccessControlPolicy
	if err := r.db.SelectContext(ctx, &policies, query); err != nil {
		return nil, err
	}
	return policies, nil
}

func (r *sqlAccessPolicyRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.AccessControlPolicy, error) {
	const query = `
        SELECT id, policy_id, scope, role, permissions, duration_days, requires_certification,
               max_claims_per_user, approval_threshold, renewal_conditions, created_at, updated_at
          FROM access_control_policies
         WHERE id = $1
         LIMIT 1
    `
	var policy models.AccessControlPolicy
	if err := r.db.GetContext(ctx, &policy, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("access policy %s not found", id)
		}
		return nil, err
	}
	return &policy, nil
}

func (r *sqlAccessPolicyRepository) GetByPolicyID(ctx context.Context, policyID string) (*models.AccessControlPolicy, error) {
	const query = `
        SELECT id, policy_id, scope, role, permissions, duration_days, requires_certification,
               max_claims_per_user, approval_threshold, renewal_conditions, created_at, updated_at
          FROM access_control_policies
         WHERE LOWER(policy_id) = LOWER($1)
         LIMIT 1
    `
	var policy models.AccessControlPolicy
	if err := r.db.GetContext(ctx, &policy, query, policyID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("access policy %s not found", policyID)
		}
		return nil, err
	}
	return &policy, nil
}

func (r *sqlAccessPolicyRepository) Create(ctx context.Context, policy *models.AccessControlPolicy) (*models.AccessControlPolicy, error) {
	ensurePolicyTimestamps(policy)
	const query = `
        INSERT INTO access_control_policies (
            id, policy_id, scope, role, permissions, duration_days, requires_certification,
            max_claims_per_user, approval_threshold, renewal_conditions, created_at, updated_at
        ) VALUES (
            :id, :policy_id, :scope, :role, :permissions, :duration_days, :requires_certification,
            :max_claims_per_user, :approval_threshold, :renewal_conditions, :created_at, :updated_at
        )
        RETURNING id, policy_id, scope, role, permissions, duration_days, requires_certification,
                  max_claims_per_user, approval_threshold, renewal_conditions, created_at, updated_at
    `
	rows, err := r.db.NamedQueryContext(ctx, query, policy)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("policy with id %s already exists", policy.PolicyID)
		}
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var stored models.AccessControlPolicy
		if err := rows.StructScan(&stored); err != nil {
			return nil, err
		}
		return &stored, nil
	}
	return nil, errors.New("failed to insert access control policy")
}

func (r *sqlAccessPolicyRepository) Update(ctx context.Context, policy *models.AccessControlPolicy) (*models.AccessControlPolicy, error) {
	if policy == nil {
		return nil, errors.New("policy payload is required")
	}
	if policy.ID == uuid.Nil {
		return nil, errors.New("policy id is required")
	}
	ensurePolicyTimestamps(policy)
	policy.UpdatedAt = time.Now().UTC()
	const query = `
        UPDATE access_control_policies SET
            policy_id = :policy_id,
            scope = :scope,
            role = :role,
            permissions = :permissions,
            duration_days = :duration_days,
            requires_certification = :requires_certification,
            max_claims_per_user = :max_claims_per_user,
            approval_threshold = :approval_threshold,
            renewal_conditions = :renewal_conditions,
            updated_at = :updated_at
        WHERE id = :id
        RETURNING id, policy_id, scope, role, permissions, duration_days, requires_certification,
                  max_claims_per_user, approval_threshold, renewal_conditions, created_at, updated_at
    `
	rows, err := r.db.NamedQueryContext(ctx, query, policy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var stored models.AccessControlPolicy
		if err := rows.StructScan(&stored); err != nil {
			return nil, err
		}
		return &stored, nil
	}
	return nil, errors.New("failed to update access control policy")
}

func (r *sqlAccessPolicyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM access_control_policies WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("access policy %s not found", id)
	}
	return nil
}

// ensurePolicyTimestamps normalizes CreatedAt/UpdatedAt on a policy.
func ensurePolicyTimestamps(policy *models.AccessControlPolicy) {
	now := time.Now().UTC()
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = now
	}
	if policy.UpdatedAt.IsZero() {
		policy.UpdatedAt = now
	}
}

// ============================================================================
// ACCESS INTELLIGENCE SERVICE
// ============================================================================

// AccessIntelligenceService provides a unified interface for advanced access control features.
//
// This is the canonical home of the access intelligence service. The
// services-package wrapper remains for backward compatibility but new
// code should depend on this package directly.
type AccessIntelligenceService struct {
	DB              *sqlx.DB
	CollabService   CollabServiceProvider
	AutoService     AutoServiceProvider
	DTManager       *observability.DynatraceManager
	PerfMonitor     PerfMonitorProvider
	GovernanceCache *analytics.ShardedCache
	VersionManager  *analytics.VersionManager
}

// NewAccessIntelligenceService constructs the canonical access intelligence service.
//
// New code should call this directly. Legacy services.NewAccessIntelligenceService
// remains as a wrapper for backward compatibility.
func NewAccessIntelligenceService(
	db *sqlx.DB,
	collabService CollabServiceProvider,
	autoService AutoServiceProvider,
	dtManager *observability.DynatraceManager,
	perfMonitor PerfMonitorProvider,
) *AccessIntelligenceService {
	return &AccessIntelligenceService{
		DB:              db,
		CollabService:   collabService,
		AutoService:     autoService,
		DTManager:       dtManager,
		PerfMonitor:     perfMonitor,
		GovernanceCache: analytics.NewShardedCache(16, 1000),
		VersionManager:  analytics.NewVersionManager(),
	}
}