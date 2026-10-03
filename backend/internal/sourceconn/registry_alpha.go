package sourceconn

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/dscreds"
	"github.com/hondyman/uisce/backend/internal/security"
)

// AlphaRegistry implements Registry over alpha. Ownership comes from the security resolver (the same
// answer request scoping uses); the row's config is then read INSIDE the owner's tenant transaction,
// so row-level security applies to it.
type AlphaRegistry struct {
	DB       *sql.DB
	Resolver interface {
		Resolve(ctx context.Context, datasourceID string) (*security.ResolvedDatasource, error)
	}
}

var _ Registry = (*AlphaRegistry)(nil)

func (r *AlphaRegistry) Source(ctx context.Context, tenantID, datasourceID string, p Policy) (Source, error) {
	if r == nil || r.DB == nil || r.Resolver == nil {
		return Source{}, errors.New("sourceconn: alpha registry is not configured")
	}
	if strings.TrimSpace(datasourceID) == "" {
		return Source{}, ErrNotFound
	}
	owner, err := r.Resolver.Resolve(ctx, datasourceID)
	if err != nil {
		if errors.Is(err, security.ErrDatasourceNotAvailable) {
			return Source{}, ErrNotFound
		}
		return Source{}, fmt.Errorf("sourceconn: resolve datasource: %w", err)
	}

	gold := false
	switch {
	case owner.TenantID == tenantID:
	case p == OwnerOrGoldCopy:
		// A different tenant's datasource is allowed only if its owner is the gold-copy tenant.
		// This is a cross-tenant read of one boolean, so it needs the gold-copy-sync role (as the
		// resolver's own lookup does); the config below is still read as the owner.
		err := db.WithGoldCopySync(ctx, r.DB, func(tx *sql.Tx) error {
			return tx.QueryRowContext(ctx, `SELECT COALESCE(gold_copy, false) FROM public.tenants WHERE id = $1`, owner.TenantID).Scan(&gold)
		})
		if err != nil {
			return Source{}, fmt.Errorf("sourceconn: read gold copy flag: %w", err)
		}
		if !gold {
			return Source{}, ErrNotAllowed
		}
	default:
		return Source{}, ErrNotAllowed
	}

	var config string
	err = db.WithTenantTransaction(ctx, r.DB, owner.TenantID, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, `SELECT COALESCE(config::text, '') FROM public.tenant_product_datasource WHERE id = $1`, owner.DatasourceID).Scan(&config)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Source{}, ErrNotFound
	}
	if err != nil {
		return Source{}, fmt.Errorf("sourceconn: read datasource config: %w", err)
	}
	if config == "" || config == "null" {
		return Source{}, fmt.Errorf("%w: datasource %s has no connection configuration", ErrBadConfig, datasourceID)
	}
	return Source{ID: owner.DatasourceID, TenantID: owner.TenantID, Config: []byte(config), GoldCopy: gold}, nil
}

// DSCreds adapts a dscreds.Resolver to Credentials: the credential is resolved by the datasource
// ROW's own tenant and id, never the caller's.
type DSCreds struct{ R *dscreds.Resolver }

func (d DSCreds) Hydrate(ctx context.Context, s Source) ([]byte, error) {
	r := d.R
	if r == nil {
		r = dscreds.Default()
	}
	return r.Hydrate(ctx, dscreds.KindDatasource, s.TenantID, s.ID, s.Config)
}
