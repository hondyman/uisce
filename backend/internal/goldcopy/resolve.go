// Package goldcopy resolves the platform gold-copy tenant.
//
// # Platform standard (gold-copy widen decision)
//
// Shared rows are those owned by the tenant where public.tenants.gold_copy = true.
// Surfaces must NOT use the MCP transitional nil-UUID OR
// ('00000000-0000-0000-0000-000000000000') as the gold signal.
//
// Pages additionally require is_core = true when admitting gold-tenant rows
// (matches PageStudioHandler). BO/catalog reads admit gold-tenant rows by
// tenant_id alone (matches ListBusinessObjectsLegacy's gold visibility intent,
// expressed as tenant_id = gold rather than EXISTS).
package goldcopy

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ResolveTenantID returns the gold-copy tenant id, or uuid.Nil if none exists.
func ResolveTenantID(ctx context.Context, db *sqlx.DB) uuid.UUID {
	if db == nil {
		return uuid.Nil
	}
	var id uuid.UUID
	_ = db.GetContext(ctx, &id, `SELECT id FROM (SELECT public.uisce_gold_copy_tenant_id() AS id) g WHERE id IS NOT NULL`)
	return id
}
