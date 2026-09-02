-- Maps an IdP group claim (e.g. a Keycloak group path) to an internal
-- bp_roles.role_key, scoped per tenant. Lets a single-realm user (e.g. a
-- uisce-internal operator) hold different entitlement levels in different
-- tenants purely via group membership, without any tenant needing to know
-- our internal role_key vocabulary.
--
-- Unlike security.identity_profile_mappings (decorative — feeds only
-- FunctionalRole/ClearanceLevel for persona rendering, never gates access),
-- this table is read directly by GroupRoleResolver and its output is merged
-- into security.Context.Roles, which bp_field_permissions matches against.
CREATE TABLE IF NOT EXISTS security.idp_group_role_mappings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    idp_group_claim TEXT NOT NULL,
    role_key        TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, idp_group_claim, role_key)
);

CREATE INDEX IF NOT EXISTS idx_idp_group_role_mappings_lookup
    ON security.idp_group_role_mappings (tenant_id, idp_group_claim);
