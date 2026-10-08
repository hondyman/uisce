package tenantnetwork

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PgAllowlistStore implements AllowlistStore against the live schema, which is
// the schema CI restores from backend/db/snapshots/schema-snapshot.sql.
//
// On that schema each entry is owned by one tenant (entries.tenant_id, unique
// per tenant and address), and the middleware enforces through assignments.
// Replacing a tenant's list therefore touches only that tenant's rows. Forced
// row-level security applies to the write, so the tenant context is set inside
// the same transaction.
//
// Note: migration 000018 describes a different, global-entry model. That
// migration does not match the running schema, so it must not be used as the
// reference for this store.
type PgAllowlistStore struct {
	Pool *pgxpool.Pool
}

// ReplaceTenantAllowlist replaces one tenant's allowlist atomically. Any error
// rolls the whole change back, so the previous list stays in place.
func (s *PgAllowlistStore) ReplaceTenantAllowlist(ctx context.Context, tenantID string, cidrs []string) (err error) {
	if s.Pool == nil {
		return errors.New("allowlist store has no database pool")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return errors.New("begin allowlist transaction")
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(context.Background())
		}
	}()

	// Scope the row-level security context to this tenant for this transaction.
	if _, err = tx.Exec(ctx, `SELECT set_config('uisce.current_tenant', $1, true)`, tenantID); err != nil {
		return errors.New("set tenant context")
	}
	if _, err = tx.Exec(ctx, `DELETE FROM tenant_ip_whitelist_assignments WHERE tenant_id = $1`, tenantID); err != nil {
		return errors.New("clear current allowlist")
	}
	if _, err = tx.Exec(ctx, `DELETE FROM tenant_ip_whitelist_entries WHERE tenant_id = $1`, tenantID); err != nil {
		return errors.New("clear current allowlist entries")
	}
	for _, cidr := range cidrs {
		var entryID string
		if err = tx.QueryRow(ctx, `
			INSERT INTO tenant_ip_whitelist_entries (tenant_id, ip_address, created_at, updated_at)
			VALUES ($1::uuid, $2, now(), now())
			ON CONFLICT (tenant_id, ip_address) DO UPDATE SET updated_at = now()
			RETURNING id::text`, tenantID, cidr).Scan(&entryID); err != nil {
			return errors.New("write allowlist entry")
		}
		// The assignment is written in the same transaction as the entry, so an
		// entry is never committed without its tenant assignment.
		if _, err = tx.Exec(ctx, `
			INSERT INTO tenant_ip_whitelist_assignments (whitelist_id, tenant_id, created_at)
			VALUES ($1::uuid, $2::uuid, now())
			ON CONFLICT DO NOTHING`, entryID, tenantID); err != nil {
			return errors.New("assign allowlist entry to tenant")
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return errors.New("commit allowlist transaction")
	}
	return nil
}
