-- Migration: 20261225_004_seed_compliance_and_platform_pages.up.sql
-- Seed route aliases and point navigation_menu_nodes to compliance-hub

DO $$
DECLARE
    v_gold_copy_id UUID := '99e99e99-99e9-49e9-89e9-99e99e99e999';
    v_tenant RECORD;
BEGIN
    -- For every tenant in the database, insert the route aliases for compliance-hub
    FOR v_tenant IN SELECT tenant_id FROM tenants LOOP
        INSERT INTO route_aliases (tenant_id, path, page_key, created_at)
        VALUES
            (v_tenant.tenant_id, '/governance/compliance', 'compliance-hub', NOW()),
            (v_tenant.tenant_id, '/compliance', 'compliance-hub', NOW()),
            (v_tenant.tenant_id, '/compliance/rules', 'compliance-hub', NOW()),
            (v_tenant.tenant_id, '/compliance/matrix', 'compliance-hub', NOW()),
            (v_tenant.tenant_id, '/compliance/activations', 'compliance-hub', NOW()),
            (v_tenant.tenant_id, '/compliance/blotter', 'compliance-hub', NOW()),
            (v_tenant.tenant_id, '/compliance/regulatory', 'compliance-hub', NOW()),
            (v_tenant.tenant_id, '/compliance/surveillance', 'compliance-hub', NOW())
        ON CONFLICT (tenant_id, path) DO UPDATE
            SET page_key = EXCLUDED.page_key;
    END LOOP;

    -- Update navigation_menu_nodes for Compliance
    UPDATE navigation_menu_nodes
    SET target_page_key = 'compliance-hub',
        target_route = NULL
    WHERE node_key = 'operations-governance-compliance';

END $$;
