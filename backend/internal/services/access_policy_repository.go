package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// AccessPolicyRepository defines persistence operations for access control policies.
type AccessPolicyRepository interface {
	List(ctx context.Context) ([]models.AccessControlPolicy, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.AccessControlPolicy, error)
	GetByPolicyID(ctx context.Context, policyID string) (*models.AccessControlPolicy, error)
	Create(ctx context.Context, policy *models.AccessControlPolicy) (*models.AccessControlPolicy, error)
	Update(ctx context.Context, policy *models.AccessControlPolicy) (*models.AccessControlPolicy, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type sqlAccessPolicyRepository struct {
	db *sqlx.DB
}

func newAccessPolicyRepository(db *sqlx.DB) AccessPolicyRepository {
	if db == nil || db.DB == nil {
		return nil
	}
	return &sqlAccessPolicyRepository{db: db}
}

func (r *sqlAccessPolicyRepository) List(ctx context.Context) ([]models.AccessControlPolicy, error) {
	const query = `
        SELECT id, policy_id, scope, role, permissions, duration_days, requires_certification,
               max_claims_per_user, approval_threshold, renewal_conditions, created_at, updated_at
          FROM access_control_policies
         ORDER BY policy_id ASC, created_at DESC
    `

	var rows []models.AccessControlPolicy
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, err
	}

	return rows, nil
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

	return nil, fmt.Errorf("access policy %s not found", policy.ID)
}

func (r *sqlAccessPolicyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM access_control_policies WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("access policy %s not found", id)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	// Handle lib/pq errors
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return strings.EqualFold(string(pqErr.Code), "23505")
	}
	return false
}

func ensurePolicyTimestamps(policy *models.AccessControlPolicy) {
	if policy == nil {
		return
	}
	now := time.Now().UTC()
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = now
	}
	if policy.UpdatedAt.IsZero() {
		policy.UpdatedAt = policy.CreatedAt
	}
}
