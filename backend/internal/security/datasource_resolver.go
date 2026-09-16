package security

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	uisce_db "github.com/hondyman/uisce/backend/internal/db"
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
		JOIN public.tenants t ON ti.tenant_id = t.id
		WHERE tpd.id = $1
		  AND tpd.is_active = true
		  AND tp.is_active = true
		  AND ti.is_active = true
		LIMIT 1
	`

	// This is a security-boundary function used to resolve WHICH tenant a
	// caller-supplied datasource ID belongs to, before the caller's own
	// tenant membership is checked against that answer (see security.go's
	// ResolveContext: resolver.Resolve, then tenantAllowed). The resolution
	// itself is inherently cross-tenant — it doesn't know the tenant until
	// this query answers that — so it needs uisce_gold_copy_sync. The actual
	// access-control decision still happens in the caller, after this
	// returns, unaffected by the elevated role used only for this lookup.
	err := uisce_db.WithGoldCopySync(ctx, r.db.DB, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, query, datasourceID).Scan(
			&row.TenantID, &row.InstanceID, &row.ProductID, &row.DatasourceID, &row.AllowedRegions,
		)
	})
	if err != nil {
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
