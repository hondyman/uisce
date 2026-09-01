package security

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// IssuerConfig is what a tenant's IDP registration resolves to: where to
// fetch signing keys, and which tenant(s) trust tokens from this issuer.
// TenantIDs has more than one entry only for a deliberately cross-tenant
// issuer (a professional-services or super-admin realm) — the normal case
// for a tenant's own Keycloak realm is a single entry. A tenant can also
// have more than one IssuerConfig registered (post-M&A, a second realm),
// which is why resolution is keyed by issuer, not by tenant.
type IssuerConfig struct {
	Issuer    string // must match the token's "iss" claim exactly
	JWKSURI   string
	TenantIDs []string
}

// IssuerRegistry resolves and enumerates trusted token issuers (Keycloak
// realms or any OIDC IDP) registered against semantic.tenant_identity_providers.
type IssuerRegistry interface {
	// ResolveIssuer returns the trust configuration for a token's "iss"
	// claim, or (nil, nil) if that issuer isn't registered at all.
	ResolveIssuer(ctx context.Context, issuer string) (*IssuerConfig, error)
	// ListActiveIssuers returns every active, registered issuer — used to
	// fetch and refresh the full set of trusted signing keys.
	ListActiveIssuers(ctx context.Context) ([]IssuerConfig, error)
}

// DBIssuerRegistry implements IssuerRegistry against
// semantic.tenant_identity_providers / tenant_identity_provider_grants.
type DBIssuerRegistry struct {
	db *sqlx.DB
}

// NewDBIssuerRegistry creates a new registry.
func NewDBIssuerRegistry(db *sqlx.DB) *DBIssuerRegistry {
	return &DBIssuerRegistry{db: db}
}

// ResolveIssuer looks up the trust configuration for a token's "iss" claim.
func (r *DBIssuerRegistry) ResolveIssuer(ctx context.Context, issuer string) (*IssuerConfig, error) {
	var idpID, jwksURI string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, jwks_uri FROM semantic.tenant_identity_providers
		WHERE issuer = $1 AND is_active = TRUE
	`, issuer).Scan(&idpID, &jwksURI)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("resolving issuer %q: %w", issuer, err)
	}

	tenantIDs, err := r.grantedTenants(ctx, idpID)
	if err != nil {
		return nil, err
	}

	return &IssuerConfig{Issuer: issuer, JWKSURI: jwksURI, TenantIDs: tenantIDs}, nil
}

// ListActiveIssuers returns every active issuer registration, for the JWKS
// key-refresh loop.
func (r *DBIssuerRegistry) ListActiveIssuers(ctx context.Context) ([]IssuerConfig, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, issuer, jwks_uri FROM semantic.tenant_identity_providers
		WHERE is_active = TRUE
	`)
	if err != nil {
		return nil, fmt.Errorf("listing active issuers: %w", err)
	}
	defer rows.Close()

	type row struct{ id, issuer, jwksURI string }
	var idps []row
	for rows.Next() {
		var rr row
		if err := rows.Scan(&rr.id, &rr.issuer, &rr.jwksURI); err != nil {
			continue
		}
		idps = append(idps, rr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	configs := make([]IssuerConfig, 0, len(idps))
	for _, idp := range idps {
		tenantIDs, err := r.grantedTenants(ctx, idp.id)
		if err != nil {
			return nil, err
		}
		configs = append(configs, IssuerConfig{Issuer: idp.issuer, JWKSURI: idp.jwksURI, TenantIDs: tenantIDs})
	}
	return configs, nil
}

func (r *DBIssuerRegistry) grantedTenants(ctx context.Context, idpID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tenant_id::text FROM semantic.tenant_identity_provider_grants
		WHERE idp_id = $1
	`, idpID)
	if err != nil {
		return nil, fmt.Errorf("loading tenant grants for idp %q: %w", idpID, err)
	}
	defer rows.Close()

	var tenantIDs []string
	for rows.Next() {
		var tid string
		if err := rows.Scan(&tid); err != nil {
			continue
		}
		tenantIDs = append(tenantIDs, tid)
	}
	return tenantIDs, rows.Err()
}
