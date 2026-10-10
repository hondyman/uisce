package provisioning

import (
	"context"
	"database/sql"
	"errors"
)

// SQLCatalog is the Catalog over the control database (alpha).
type SQLCatalog struct {
	DB *sql.DB
}

// Regions are the active regions with the cluster each one runs on, if it has an active one.
func (c SQLCatalog) Regions(ctx context.Context) ([]Region, error) {
	if c.DB == nil {
		return nil, errors.New("no control database")
	}
	rows, err := c.DB.QueryContext(ctx, `
		SELECT rc.region_code, rc.region_name, COALESCE(pc.host, ''), COALESCE(pc.port, 0)
		FROM public.region_config rc
		LEFT JOIN public.region_postgres_cluster pc ON pc.region_code = rc.region_code AND pc.is_active
		WHERE rc.is_active
		ORDER BY rc.region_code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Region
	for rows.Next() {
		var r Region
		if err := rows.Scan(&r.Code, &r.Name, &r.ClusterHost, &r.ClusterPort); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Products are the active products that have a code.
func (c SQLCatalog) Products(ctx context.Context) ([]Product, error) {
	if c.DB == nil {
		return nil, errors.New("no control database")
	}
	rows, err := c.DB.QueryContext(ctx, `
		SELECT product_code, COALESCE(product_name, product_code)
		FROM public.alpha_product
		WHERE is_active AND status = 'active' AND product_code IS NOT NULL AND product_code <> ''
		ORDER BY product_code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.Code, &p.Name); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// DatabaseNameTaken is true when a database of this name exists on the control cluster, or any
// datasource already names it. A database on another cluster is checked again by the worker before
// it creates anything.
func (c SQLCatalog) DatabaseNameTaken(ctx context.Context, name string) (bool, error) {
	if c.DB == nil {
		return false, errors.New("no control database")
	}
	var taken bool
	err := c.DB.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)
		    OR EXISTS (SELECT 1 FROM public.tenant_product_datasource WHERE config ->> 'database' = $1)`, name).Scan(&taken)
	return taken, err
}
