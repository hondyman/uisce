package metadata

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// IsGoldCopyTenant reports whether tenantID is the gold-copy tenant
// (public.uisce_gold_copy_tenant_id()). Anything the gold-copy tenant
// creates is core metadata that every other tenant inherits read-only, so
// create paths use this - never a client-supplied isCore flag - to decide a
// new business object's / binding's is_core. False when no gold copy is
// configured.
func IsGoldCopyTenant(ctx context.Context, q sqlx.QueryerContext, tenantID string) (bool, error) {
	var gold bool
	err := sqlx.GetContext(ctx, q, &gold, `SELECT COALESCE($1::uuid = public.uisce_gold_copy_tenant_id(), false)`, tenantID)
	return gold, err
}
