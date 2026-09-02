package security

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// GroupRoleResolver resolves an authenticated principal's IdP group
// memberships into internal bp_roles.role_key values, scoped to a single
// tenant. The same group claim can resolve to different role_keys in
// different tenants (security.idp_group_role_mappings is keyed on both),
// which is what lets one identity hold e.g. read-only access in several
// tenants and full CRUD in others, driven entirely by group membership.
type GroupRoleResolver struct {
	db *sqlx.DB
}

// NewGroupRoleResolver creates a resolver backed by security.idp_group_role_mappings.
func NewGroupRoleResolver(db *sqlx.DB) *GroupRoleResolver {
	return &GroupRoleResolver{db: db}
}

// ResolveRoles returns the distinct role_keys granted for tenantID to any of
// the given idpGroups, and provisions the bp_user_roles row(s) that
// EntitlementsService still requires as its join anchor — so a group match
// alone is sufficient; nobody has to hand-insert a bp_user_roles row per
// user per tenant. Provisioning is idempotent (ON CONFLICT DO NOTHING) and
// runs synchronously on the request path, so it costs a no-op upsert per
// distinct (user, tenant, role) combination once already provisioned.
//
// Returns an empty (non-nil) slice, not an error, when there are no groups
// or no matches — group-based mapping is additive on top of any literal
// role claims the token already carries.
func (r *GroupRoleResolver) ResolveRoles(ctx context.Context, userID, tenantID string, idpGroups []string) ([]string, error) {
	if r == nil || r.db == nil || tenantID == "" || len(idpGroups) == 0 {
		return []string{}, nil
	}

	var roleKeys []string
	err := r.db.SelectContext(ctx, &roleKeys, `
		SELECT DISTINCT role_key
		FROM security.idp_group_role_mappings
		WHERE tenant_id = $1 AND idp_group_claim = ANY($2)
	`, tenantID, pq.Array(idpGroups))
	if err != nil {
		return nil, fmt.Errorf("resolving group roles for tenant %q: %w", tenantID, err)
	}
	if roleKeys == nil {
		roleKeys = []string{}
	}

	if userID != "" && len(roleKeys) > 0 {
		// NOTE: bp_user_roles' unique constraint includes the nullable
		// tenant_instance_id column, and Postgres treats NULL as distinct from
		// NULL in uniqueness checks — so ON CONFLICT on that composite key
		// never fires here and would silently duplicate rows on every
		// request. Guard with NOT EXISTS instead.
		if _, err := r.db.ExecContext(ctx, `
			UPDATE bp_user_roles ur
			SET is_active = true
			FROM bp_roles r
			WHERE ur.user_id = $1 AND ur.tenant_id = $2 AND ur.tenant_instance_id IS NULL
			  AND ur.role_key = r.role_key AND r.role_key = ANY($3) AND r.is_active = true
			  AND ur.is_active = false
		`, userID, tenantID, pq.Array(roleKeys)); err != nil {
			return nil, fmt.Errorf("reactivating group-derived user roles for tenant %q: %w", tenantID, err)
		}
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO bp_user_roles (user_id, role_id, tenant_id, role_key, is_active)
			SELECT $1, r.id, $2, r.role_key, true
			FROM bp_roles r
			WHERE r.role_key = ANY($3) AND r.is_active = true
			  AND NOT EXISTS (
			      SELECT 1 FROM bp_user_roles ur
			      WHERE ur.user_id = $1 AND ur.tenant_id = $2 AND ur.role_key = r.role_key
			        AND ur.tenant_instance_id IS NULL
			  )
		`, userID, tenantID, pq.Array(roleKeys)); err != nil {
			return nil, fmt.Errorf("provisioning group-derived user roles for tenant %q: %w", tenantID, err)
		}
	}

	return roleKeys, nil
}
