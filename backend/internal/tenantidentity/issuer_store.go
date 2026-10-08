package tenantidentity

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// IssuerKey is the key under tenants.configuration where the realm issuer lives.
const IssuerKey = "identity_issuer"

// PgIssuerStore implements IssuerStore against tenants.configuration.
type PgIssuerStore struct {
	Pool *pgxpool.Pool
}

// SetIssuer upserts the issuer for one tenant. Keyed by tenant UUID, so a retry
// writes the same value again and an update overwrites the earlier one.
func (s *PgIssuerStore) SetIssuer(ctx context.Context, tenantID, issuer string) error {
	if s.Pool == nil {
		return errors.New("issuer store has no database pool")
	}
	tag, err := s.Pool.Exec(ctx, `
		UPDATE tenants
		SET configuration = jsonb_set(COALESCE(configuration, '{}'::jsonb), '{`+IssuerKey+`}', to_jsonb($2::text), true),
		    updated_at = now()
		WHERE id = $1::uuid`, tenantID, issuer)
	if err != nil {
		return errors.New("record issuer: the database rejected the write")
	}
	if tag.RowsAffected() != 1 {
		return errors.New("record issuer: tenant not found")
	}
	return nil
}
