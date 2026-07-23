package scheduler

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// SQLTenantRepository implements TenantRepository
type SQLTenantRepository struct {
	db *sqlx.DB
}

// NewSQLTenantRepository creates a new repo
func NewSQLTenantRepository(db *sqlx.DB) *SQLTenantRepository {
	return &SQLTenantRepository{db: db}
}

// ListActiveTenants queries the auth.tenants table for active tenants
func (r *SQLTenantRepository) ListActiveTenants(ctx context.Context) ([]Tenant, error) {
	var tenants []Tenant
	err := r.db.SelectContext(ctx, &tenants, "SELECT id, name FROM auth.tenants WHERE is_deleted = false")
	return tenants, err
}
