-- Capability-driven UI menu ABAC policies.
-- The frontend remains role-agnostic: it only renders menus whose capability
-- the backend explicitly grants via /api/auth/me/entitlements.

-- Make abac_policies nullable so global (tenant-agnostic) menu policies can be
-- stored.  Some environments still have the original NOT NULL constraints.
ALTER TABLE abac_policies ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE abac_policies ALTER COLUMN datasource_id DROP NOT NULL;

-- 1. Ensure Alice's federated identity profile maps to a standard tenant role.
--    The mapping is keyed by the IdP client (azp) and group claim so it works
--    for any IdP vendor.
DO $$
DECLARE
    invesco_tenant_id uuid := 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11';
BEGIN
    INSERT INTO security.identity_profile_mappings (
        mapping_id, tenant_id, idp_client_id, idp_group_claim, functional_role, clearance_level
    ) VALUES (
        gen_random_uuid(),
        invesco_tenant_id,
        'investco-sso-client',
        'GG-Uisce-Users',
        'platform_analyst',
        'L1'
    )
    ON CONFLICT (idp_client_id, idp_group_claim) DO UPDATE
    SET functional_role = EXCLUDED.functional_role,
        clearance_level = EXCLUDED.clearance_level,
        tenant_id = EXCLUDED.tenant_id;
END $$;

-- 2. Base UI menus: every authenticated tenant user sees these.
INSERT INTO abac_policies (
    id, tenant_id, name, description, effect, priority, enabled,
    subject_rules, action_rules, resource_rules, environment_rules
) VALUES (
    gen_random_uuid(),
    NULL,
    'Base UI Menus',
    'Standard tenant-scoped menus available to all platform users',
    'allow',
    100,
    true,
    '{"roles": ["platform_analyst", "tenant_admin", "global_admin", "professional_services"]}'::jsonb,
    '{"action": "view"}'::jsonb,
    '{"type": "ui_menu", "name": ["catalog", "glossary", "discovery", "lineage", "build", "models", "rules", "quality", "studio", "api-studio", "page-studio", "workflow-studio", "operations", "scheduler", "workflows", "governance", "intelligence", "optimization", "observability", "ai-copilot", "consume", "reports", "analytics", "dashboards", "calendar"]}'::jsonb,
    '{}'::jsonb
) ON CONFLICT DO NOTHING;

-- 3. Operator / Organization menus: only global admins, professional services,
--    and tenant administrators may see platform administration menus.
INSERT INTO abac_policies (
    id, tenant_id, name, description, effect, priority, enabled,
    subject_rules, action_rules, resource_rules, environment_rules
) VALUES (
    gen_random_uuid(),
    NULL,
    'Operator UI Menus',
    'Organization, security, and system administration menus',
    'allow',
    100,
    true,
    '{"roles": ["tenant_admin", "global_admin", "professional_services"]}'::jsonb,
    '{"action": "view"}'::jsonb,
    '{"type": "ui_menu", "name": ["platform", "organization", "security", "system"]}'::jsonb,
    '{}'::jsonb
) ON CONFLICT DO NOTHING;
