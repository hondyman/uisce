-- Migration: 20261225_005_seed_platform_rbac_pages.up.sql
-- Seeds page definitions, route aliases, and updates menu nodes for Platform & RBAC pages

DO $$
DECLARE
    v_gold_copy_id UUID := '99e99e99-99e9-49e9-89e9-99e99e99e999';
    v_tenant RECORD;
BEGIN
    -- 1. Seed Core Page Definitions for gold_copy tenant
    INSERT INTO page_definitions (
        id, tenant_id, name, slug, description, layout, components,
        data_sources, version, is_core, status, tabs, presentation_events,
        filter_bar, app_model, menu_hidden, created_at, updated_at
    ) VALUES
    (
        '018f9d02-0001-7000-8000-000000000501',
        v_gold_copy_id,
        'Teams',
        'platform-teams',
        'Manage organizational teams, membership, and resource assignments',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "admin.TeamManager", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000502',
        v_gold_copy_id,
        'Users',
        'platform-users',
        'User accounts, profile management, and credentials',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "admin.UserManager", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000503',
        v_gold_copy_id,
        'User Roles',
        'platform-user-roles',
        'Assign and manage RBAC role bindings for users',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "admin.UserRoleAssignment", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000504',
        v_gold_copy_id,
        'User Tenants',
        'platform-user-tenants',
        'Multi-tenant memberships and tenant access assignment',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "admin.TenantUserAssignment", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000505',
        v_gold_copy_id,
        'Delegations',
        'platform-delegations',
        'Time-bound and approval-scoped user access delegations',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "admin.DelegationManager", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000506',
        v_gold_copy_id,
        'Field Permissions',
        'platform-field-permissions',
        'Fine-grained attribute and column-level permission matrix',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "admin.FieldPermissionEditor", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000507',
        v_gold_copy_id,
        'Roles & Permissions',
        'platform-roles',
        'Define RBAC roles, permission sets, and capability grants',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "admin.RoleManager", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000508',
        v_gold_copy_id,
        'IP Whitelist',
        'platform-ip-whitelist',
        'Configure network IP allowlists and CIDR ranges per tenant',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "admin.IPWhitelist", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000509',
        v_gold_copy_id,
        'Message Catalog',
        'platform-message-catalog',
        'Unified error code, localization, and message catalog editor',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "admin.MessageCatalog", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000510',
        v_gold_copy_id,
        'System Seeding',
        'platform-seeding',
        'Seed catalog metadata, demo business objects, and rule fixtures',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "admin.Seeding", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    )
    ON CONFLICT (tenant_id, slug) DO UPDATE
        SET name = EXCLUDED.name,
            description = EXCLUDED.description,
            layout = EXCLUDED.layout,
            components = EXCLUDED.components,
            filter_bar = EXCLUDED.filter_bar,
            app_model = EXCLUDED.app_model,
            status = 'published',
            updated_at = NOW();

    -- 2. Seed Route Aliases for all tenants
    FOR v_tenant IN SELECT tenant_id FROM tenants LOOP
        INSERT INTO route_aliases (tenant_id, path, page_key, created_at)
        VALUES
            (v_tenant.tenant_id, '/admin/rbac/teams', 'platform-teams', NOW()),
            (v_tenant.tenant_id, '/admin/rbac/users', 'platform-users', NOW()),
            (v_tenant.tenant_id, '/admin/rbac/user-roles', 'platform-user-roles', NOW()),
            (v_tenant.tenant_id, '/admin/rbac/user-tenants', 'platform-user-tenants', NOW()),
            (v_tenant.tenant_id, '/admin/rbac/delegations', 'platform-delegations', NOW()),
            (v_tenant.tenant_id, '/admin/rbac/field-permissions', 'platform-field-permissions', NOW()),
            (v_tenant.tenant_id, '/admin/rbac/roles', 'platform-roles', NOW()),
            (v_tenant.tenant_id, '/fabric/ip-whitelist', 'platform-ip-whitelist', NOW()),
            (v_tenant.tenant_id, '/admin/message-catalog', 'platform-message-catalog', NOW()),
            (v_tenant.tenant_id, '/admin/seeding', 'platform-seeding', NOW())
        ON CONFLICT (tenant_id, path) DO UPDATE
            SET page_key = EXCLUDED.page_key;
    END LOOP;

    -- 3. Repoint navigation_menu_nodes to page keys and clear target_route
    UPDATE navigation_menu_nodes SET target_page_key = 'platform-teams', target_route = NULL WHERE node_key = 'platform-organization-teams';
    UPDATE navigation_menu_nodes SET target_page_key = 'platform-users', target_route = NULL WHERE node_key = 'platform-organization-users';
    UPDATE navigation_menu_nodes SET target_page_key = 'platform-user-roles', target_route = NULL WHERE node_key = 'platform-organization-user-roles';
    UPDATE navigation_menu_nodes SET target_page_key = 'platform-user-tenants', target_route = NULL WHERE node_key = 'platform-organization-user-tenants';
    UPDATE navigation_menu_nodes SET target_page_key = 'platform-delegations', target_route = NULL WHERE node_key = 'platform-security-delegations';
    UPDATE navigation_menu_nodes SET target_page_key = 'platform-field-permissions', target_route = NULL WHERE node_key = 'platform-security-field-permissions';
    UPDATE navigation_menu_nodes SET target_page_key = 'platform-roles', target_route = NULL WHERE node_key = 'platform-security-roles-permissions';
    UPDATE navigation_menu_nodes SET target_page_key = 'platform-ip-whitelist', target_route = NULL WHERE node_key = 'platform-security-ip-whitelist';
    UPDATE navigation_menu_nodes SET target_page_key = 'platform-message-catalog', target_route = NULL WHERE node_key = 'system-message-catalog';
    UPDATE navigation_menu_nodes SET target_page_key = 'platform-seeding', target_route = NULL WHERE node_key = 'system-seeding';

END $$;
