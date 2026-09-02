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
		JOIN tenant_instance ti ON tp.datasource_id = ti.id
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

// ResolveBindingDatasource takes a Business Object binding's declared
// logical datasource slot (alpha_product_id + alpha_datasource_id — e.g.
// "ORM Connection", set once by gold copy on a core binding) and returns the
// CALLING TENANT's own tenant_product_datasource.id for that same slot. This
// is what a caller puts in secCtx.DatasourceID before Preview/Execute: the
// core BO/binding definition is shared across every tenant, but each tenant
// resolves to their own dedicated database/credentials, never gold copy's.
// Relies on the unique index added in
// 20260901000003_tenant_product_datasource_slot_unique.sql — at most one row
// can match, so this is unambiguous.
func (r *DBDatasourceResolver) ResolveBindingDatasource(ctx context.Context, tenantID, alphaProductID, alphaDatasourceID string) (string, error) {
	if r == nil || r.db == nil {
		return "", fmt.Errorf("database not configured")
	}
	if strings.TrimSpace(tenantID) == "" {
		return "", fmt.Errorf("tenant_id is required")
	}
	if strings.TrimSpace(alphaProductID) == "" || strings.TrimSpace(alphaDatasourceID) == "" {
		return "", fmt.Errorf("binding has no datasource slot configured")
	}

	var tpdID string
	query := `
		SELECT id FROM tenant_product_datasource
		WHERE tenant_id = $1
		  AND alpha_product_id = $2
		  AND alpha_datasource_id = $3
		  AND is_active = true
		LIMIT 1
	`
	if err := r.db.GetContext(ctx, &tpdID, query, tenantID, alphaProductID, alphaDatasourceID); err != nil {
		return "", fmt.Errorf("tenant %s has no datasource configured for this binding's product/datasource type: %w", tenantID, err)
	}
	return tpdID, nil
}
