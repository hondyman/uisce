package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// TenantStore defines operations for tenant management
type TenantStore interface {
	CreateTenant(ctx context.Context, req models.TenantCreateRequest) (*models.Tenant, error)
	GetTenantByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
	GetTenantByCode(ctx context.Context, code string) (*models.Tenant, error)
	ListTenants(ctx context.Context, limit int, offset int) ([]*models.Tenant, int, error)
	UpdateTenant(ctx context.Context, id uuid.UUID, req models.TenantUpdateRequest) (*models.Tenant, error)
	DeleteTenant(ctx context.Context, id uuid.UUID) error
	ValidateTenantIDs(ctx context.Context, ids []uuid.UUID) error
	SuspendTenant(ctx context.Context, id uuid.UUID) error
	UnsuspendTenant(ctx context.Context, id uuid.UUID) error
}

// tenantStoreImpl implements TenantStore
type tenantStoreImpl struct {
	db *sqlx.DB
}

// NewTenantStore creates a new tenant store
func NewTenantStore(db *sqlx.DB) TenantStore {
	return &tenantStoreImpl{db: db}
}

// CreateTenant creates a new tenant
func (s *tenantStoreImpl) CreateTenant(ctx context.Context, req models.TenantCreateRequest) (*models.Tenant, error) {
	if !models.ValidateTenantPlan(req.Plan) {
		return nil, fmt.Errorf("invalid plan: %s", req.Plan)
	}

	now := time.Now()
	query := `
		INSERT INTO tenants (id, name, code, region, plan, max_requests, window_seconds, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, name, code, region, plan, max_requests, window_seconds, is_suspended, created_at, updated_at
	`

	var tenant models.Tenant
	err := s.db.QueryRowxContext(ctx, query,
		req.ID, req.Name, req.Code, req.Region, req.Plan,
		req.MaxRequests, req.WindowSeconds, now, now,
	).StructScan(&tenant)

	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	return &tenant, nil
}

// GetTenantByID retrieves a tenant by ID
func (s *tenantStoreImpl) GetTenantByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	query := `
		SELECT id, name, code, region, plan, max_requests, window_seconds, is_suspended, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`

	var tenant models.Tenant
	err := s.db.QueryRowxContext(ctx, query, id).StructScan(&tenant)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	return &tenant, nil
}

// GetTenantByCode retrieves a tenant by code
func (s *tenantStoreImpl) GetTenantByCode(ctx context.Context, code string) (*models.Tenant, error) {
	query := `
		SELECT id, name, code, region, plan, max_requests, window_seconds, is_suspended, created_at, updated_at
		FROM tenants
		WHERE code = $1
	`

	var tenant models.Tenant
	err := s.db.QueryRowxContext(ctx, query, code).StructScan(&tenant)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get tenant by code: %w", err)
	}

	return &tenant, nil
}

// ListTenants lists all tenants with pagination
func (s *tenantStoreImpl) ListTenants(ctx context.Context, limit int, offset int) ([]*models.Tenant, int, error) {
	query := `
		SELECT id, name, code, region, plan, max_requests, window_seconds, is_suspended, created_at, updated_at
		FROM tenants
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := s.db.QueryxContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*models.Tenant
	for rows.Next() {
		var tenant models.Tenant
		if err := rows.StructScan(&tenant); err != nil {
			return nil, 0, fmt.Errorf("failed to scan tenant: %w", err)
		}
		tenants = append(tenants, &tenant)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM tenants`
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count tenants: %w", err)
	}

	return tenants, total, nil
}

// UpdateTenant updates a tenant
func (s *tenantStoreImpl) UpdateTenant(ctx context.Context, id uuid.UUID, req models.TenantUpdateRequest) (*models.Tenant, error) {
	query := `
		UPDATE tenants
		SET name = COALESCE($1, name),
		    region = COALESCE($2, region),
		    plan = COALESCE($3, plan),
		    max_requests = COALESCE($4, max_requests),
		    window_seconds = COALESCE($5, window_seconds),
		    is_suspended = COALESCE($6, is_suspended),
		    updated_at = now()
		WHERE id = $7
		RETURNING id, name, code, region, plan, max_requests, window_seconds, is_suspended, created_at, updated_at
	`

	var tenant models.Tenant
	err := s.db.QueryRowxContext(ctx, query,
		req.Name, req.Region, req.Plan, req.MaxRequests, req.WindowSeconds, req.IsSuspended, id,
	).StructScan(&tenant)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("failed to update tenant: %w", err)
	}

	return &tenant, nil
}

// DeleteTenant deletes a tenant
func (s *tenantStoreImpl) DeleteTenant(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM tenants WHERE id = $1`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant not found")
	}

	return nil
}

// ValidateTenantIDs validates that all provided tenant IDs exist
func (s *tenantStoreImpl) ValidateTenantIDs(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return fmt.Errorf("at least one tenant_id is required")
	}

	query := `SELECT COUNT(*) FROM tenants WHERE id = ANY($1)`
	var count int
	if err := s.db.QueryRowContext(ctx, query, pq.Array(ids)).Scan(&count); err != nil {
		return fmt.Errorf("failed to validate tenant_ids: %w", err)
	}

	if count != len(ids) {
		return fmt.Errorf("one or more tenant_ids are invalid or do not exist")
	}

	return nil
}

// SuspendTenant suspends a tenant
func (s *tenantStoreImpl) SuspendTenant(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE tenants SET is_suspended = true, updated_at = now() WHERE id = $1`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to suspend tenant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant not found")
	}

	return nil
}

// UnsuspendTenant unsuspends a tenant
func (s *tenantStoreImpl) UnsuspendTenant(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE tenants SET is_suspended = false, updated_at = now() WHERE id = $1`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to unsuspend tenant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant not found")
	}

	return nil
}
