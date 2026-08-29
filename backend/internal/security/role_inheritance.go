package security

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// GoldCopyTenantID returns the id of the tenant flagged as the gold-copy
// (template) tenant, mirroring public.get_gold_copy_tenant_id() used by the
// business_objects core/custom overlay.
func GoldCopyTenantID(ctx context.Context, db *sql.DB) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.QueryRowContext(ctx, `SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1`).Scan(&id)
	if err == sql.ErrNoRows {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to resolve gold copy tenant: %w", err)
	}
	return id, nil
}

// Permission is the resolved (role, effective) permission pair returned by
// ResolveEffectivePermissions.
type Permission struct {
	ID             string `json:"id" db:"id"`
	PermissionKey  string `json:"permission_key" db:"permission_key"`
	PermissionName string `json:"permission_name" db:"permission_name"`
	ResourceType   string `json:"resource_type" db:"resource_type"`
	Action         string `json:"action" db:"action"`
	SourceRoleID   string `json:"source_role_id" db:"source_role_id"`
	Inherited      bool   `json:"inherited" db:"inherited"`
}

// ResolveEffectivePermissions walks a role's parent_role_id chain up to its
// gold-copy ancestor and unions bp_role_permissions across every level.
// Permissions are additive grants (bp_role_permissions has no deny state),
// so a tenant role automatically gets everything its gold-copy ancestor
// grants plus anything locally added, without any row ever being copied.
func ResolveEffectivePermissions(ctx context.Context, db *sql.DB, roleID string) ([]Permission, error) {
	const maxDepth = 20 // guards against an accidental parent_role_id cycle

	seen := make(map[string]bool)
	permsByKey := make(map[string]Permission)
	currentID := roleID

	for depth := 0; depth < maxDepth && currentID != "" && !seen[currentID]; depth++ {
		seen[currentID] = true

		rows, err := db.QueryContext(ctx, `
			SELECT p.id, p.permission_key, p.permission_name, p.resource_type, p.action
			FROM bp_role_permissions rp
			JOIN bp_permissions p ON p.id = rp.permission_id
			WHERE rp.role_id = $1
		`, currentID)
		if err != nil {
			return nil, fmt.Errorf("failed to load permissions for role %s: %w", currentID, err)
		}
		for rows.Next() {
			var p Permission
			if err := rows.Scan(&p.ID, &p.PermissionKey, &p.PermissionName, &p.ResourceType, &p.Action); err != nil {
				rows.Close()
				return nil, err
			}
			p.SourceRoleID = currentID
			p.Inherited = currentID != roleID
			// First writer (closest to the requested role) wins so a local
			// grant's provenance is preferred over an inherited one with the
			// same key.
			if _, exists := permsByKey[p.PermissionKey]; !exists {
				permsByKey[p.PermissionKey] = p
			}
		}
		rows.Close()

		var parentID sql.NullString
		if err := db.QueryRowContext(ctx, `SELECT parent_role_id::text FROM bp_roles WHERE id = $1`, currentID).Scan(&parentID); err != nil {
			if err == sql.ErrNoRows {
				break
			}
			return nil, fmt.Errorf("failed to walk parent_role_id for role %s: %w", currentID, err)
		}
		if !parentID.Valid {
			break
		}
		currentID = parentID.String
	}

	out := make([]Permission, 0, len(permsByKey))
	for _, p := range permsByKey {
		out = append(out, p)
	}
	return out, nil
}

// ComponentEntitlement mirrors studio.tenant_component_entitlements plus the
// level it was resolved from, for display in the Entitlement Matrix.
type ComponentEntitlement struct {
	NodePath        string `json:"node_path"`
	EntitlementType string `json:"entitlement_type"`
	OverrideState   string `json:"override_state"`
	ConditionDsl    string `json:"condition_dsl,omitempty"`
	ResolvedTenant  string `json:"resolved_tenant,omitempty"` // "" (empty) means gold-copy baseline
	Inherited       bool   `json:"inherited"`
}

// ResolveEffectiveEntitlements walks security.security_profiles.parent_profile_id
// from the given (tenantID, profileKey) up to the root gold-copy profile. For
// each node_path, the nearest non-INHERIT_BASELINE row found while walking
// child -> parent wins; a node_path with no row at any level is absent from
// the result (denied by default, matching the existing fail-closed behavior).
func ResolveEffectiveEntitlements(ctx context.Context, db *sql.DB, tenantID uuid.UUID, profileKey string) ([]ComponentEntitlement, error) {
	const maxDepth = 20

	type levelKey struct {
		tenantID   *uuid.UUID
		profileKey string
	}

	seen := make(map[uuid.UUID]bool)
	resolved := make(map[string]ComponentEntitlement) // keyed by entitlement_type + "|" + node_path

	var curTenant *uuid.UUID = &tenantID
	curKey := profileKey
	isRoot := false

	for depth := 0; depth < maxDepth; depth++ {
		var tenantFilter interface{}
		if curTenant == nil {
			tenantFilter = nil
		} else {
			tenantFilter = *curTenant
		}

		rows, err := db.QueryContext(ctx, `
			SELECT entitlement_type::text, node_path, override_state::text, COALESCE(condition_dsl, '')
			FROM studio.tenant_component_entitlements
			WHERE target_profile_key = $1 AND tenant_id IS NOT DISTINCT FROM $2
		`, curKey, tenantFilter)
		if err != nil {
			return nil, fmt.Errorf("failed to load entitlements for profile %s: %w", curKey, err)
		}
		for rows.Next() {
			var e ComponentEntitlement
			if err := rows.Scan(&e.EntitlementType, &e.NodePath, &e.OverrideState, &e.ConditionDsl); err != nil {
				rows.Close()
				return nil, err
			}
			k := e.EntitlementType + "|" + e.NodePath
			if _, exists := resolved[k]; exists {
				continue // a nearer (more tenant-specific) level already resolved this node_path
			}
			if e.OverrideState == "INHERIT_BASELINE" {
				continue // explicitly defers to the parent level; keep walking up
			}
			if curTenant != nil {
				e.ResolvedTenant = curTenant.String()
			}
			e.Inherited = depth > 0
			resolved[k] = e
		}
		rows.Close()

		if isRoot {
			break
		}

		// Walk to the parent profile: prefer the tenant-scoped profile row,
		// falling back to the gold-copy blueprint with the same key.
		var parentProfileID sql.NullString
		var profileTenantID uuid.NullUUID
		err = db.QueryRowContext(ctx, `
			SELECT parent_profile_id::text, tenant_id
			FROM security.security_profiles
			WHERE profile_key = $1 AND tenant_id IS NOT DISTINCT FROM $2
		`, curKey, tenantFilter).Scan(&parentProfileID, &profileTenantID)
		if err == sql.ErrNoRows || !parentProfileID.Valid {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to walk parent_profile_id for %s: %w", curKey, err)
		}

		var parentTenantID uuid.NullUUID
		var parentKey string
		if err := db.QueryRowContext(ctx, `
			SELECT tenant_id, profile_key FROM security.security_profiles WHERE profile_id = $1
		`, parentProfileID.String).Scan(&parentTenantID, &parentKey); err != nil {
			if err == sql.ErrNoRows {
				break
			}
			return nil, fmt.Errorf("failed to load parent profile %s: %w", parentProfileID.String, err)
		}

		if seen[uuid.MustParse(parentProfileID.String)] {
			break // cycle guard
		}
		seen[uuid.MustParse(parentProfileID.String)] = true

		curKey = parentKey
		if parentTenantID.Valid {
			t := parentTenantID.UUID
			curTenant = &t
		} else {
			curTenant = nil
			isRoot = true
		}
	}

	out := make([]ComponentEntitlement, 0, len(resolved))
	for _, e := range resolved {
		out = append(out, e)
	}
	return out, nil
}
