package security

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

type DBDatasourceResolver struct {
	db *sqlx.DB
}

func NewDBDatasourceResolver(db *sqlx.DB) *DBDatasourceResolver {
	return &DBDatasourceResolver{db: db}
}

func (r *DBDatasourceResolver) Resolve(ctx context.Context, datasourceID string) (*ResolvedDatasource, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("database not configured")
	}
	if strings.TrimSpace(datasourceID) == "" {
		return nil, fmt.Errorf("datasource_id is required")
	}

	var row struct {
		TenantID       string         `db:"tenant_id"`
		InstanceID     string         `db:"instance_id"`
		ProductID      string         `db:"product_id"`
		DatasourceID   string         `db:"datasource_id"`
		AllowedRegions sql.NullString `db:"allowed_regions"`
	}

	query := `
		SELECT ti.tenant_id as tenant_id,
		       ti.id as instance_id,
		       tp.id as product_id,
		       tpd.id as datasource_id,
		       t.allowed_regions::text as allowed_regions
		FROM tenant_product_datasource tpd
		JOIN tenant_product tp ON tpd.tenant_product_id = tp.id
		JOIN tenant_instance ti ON tp.tenant_instance_id = ti.id
		JOIN tenants t ON ti.tenant_id = t.id
		WHERE tpd.id = $1
		  AND tpd.is_active = true
		  AND tp.is_active = true
		  AND ti.is_active = true
		LIMIT 1
	`

	if err := r.db.GetContext(ctx, &row, query, datasourceID); err != nil {
		return nil, fmt.Errorf("datasource not found: %w", err)
	}

	return &ResolvedDatasource{
		TenantID:       row.TenantID,
		InstanceID:     row.InstanceID,
		ProductID:      row.ProductID,
		DatasourceID:   row.DatasourceID,
		AllowedRegions: parseAllowedRegions(row.AllowedRegions),
	}, nil
}
