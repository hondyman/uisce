package analytics

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// Rule inheritance.
//
// Rules authored in the gold-copy tenant are core: every tenant inherits them read-only. A tenant can
// add rules of its own, which are custom and apply to that tenant only. A tenant can never override,
// shadow or switch off a core rule: writes are tenant-scoped (a tenant only ever writes its own
// nodes), and a custom rule may not reuse a core rule's name or duplicate its conditions.

// goldCopyTenantID returns the gold-copy tenant's id, or "" if there is none.
func goldCopyTenantID(ctx context.Context, db sqlx.QueryerContext) (string, error) {
	// public.tenants is under RLS and shows a tenant only its own row, so it cannot be read to find the
	// gold-copy tenant; the SECURITY DEFINER function returns just that id (migration 20261024_006).
	var id sql.NullString
	err := sqlx.GetContext(ctx, db, &id, `SELECT public.uisce_gold_copy_tenant_id()::text`)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("resolve gold-copy tenant: %w", err)
	}
	return id.String, nil
}

// visibleTenants is the set of tenants whose rules and bindings a tenant sees: itself and the
// gold-copy tenant.
func visibleTenants(tenantID, gold string) []string {
	if gold == "" || gold == tenantID {
		return []string{tenantID}
	}
	return []string{tenantID, gold}
}

// originOf classifies a rule relative to the requesting tenant's view: authored in the gold-copy
// tenant means core, anything else custom.
func originOf(ruleTenantID, gold string) string {
	if gold != "" && ruleTenantID == gold {
		return models.ValidationRuleOriginCore
	}
	return models.ValidationRuleOriginCustom
}

// rejectCoreShadow refuses a tenant rule whose name is already taken by a core rule on the same BO.
// Rule nodes are keyed per tenant, so the database would accept the collision; the result would be two
// rules with one name and no way to tell which a violation refers to.
func (s *ValidationRuleService) rejectCoreShadow(ctx context.Context, tenantID, gold, boName, name string) error {
	if gold == "" || gold == tenantID {
		return nil
	}
	var n int
	if err := s.db.GetContext(ctx, &n, `
		SELECT count(*)
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'validation_rule'
		  AND n.tenant_id = $1::uuid
		  AND n.properties->>'bo_name' = $2
		  AND n.node_name = $3`, gold, boName, name); err != nil {
		return fmt.Errorf("check core rule names: %w", err)
	}
	if n > 0 {
		return fmt.Errorf("the rule name %q is used by a core rule on %s; core rules cannot be overridden, choose another name", name, boName)
	}
	return nil
}
