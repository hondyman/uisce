-- Migration: 20261225_006_seed_catalog_and_glossary_pages.up.sql
-- Seeds page definitions, route aliases, and updates menu nodes for Catalog & Glossary screens

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
        '018f9d02-0001-7000-8000-000000000511',
        v_gold_copy_id,
        'API Inventory',
        'catalog-api-inventory',
        'Catalog and inspect endpoints, payloads, schema contracts, and authentication',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "catalog.ApiInventory", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000512',
        v_gold_copy_id,
        'Glossary & Semantic Terms',
        'catalog-glossary',
        'Explore business glossary, semantic terms, graph relationships, and classifications',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "catalog.GlossaryExplorer", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000513',
        v_gold_copy_id,
        'Business Terms',
        'catalog-business-terms',
        'Governed business taxonomy and terminology explorer',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "catalog.BusinessTerms", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000514',
        v_gold_copy_id,
        'Custom Fields',
        'catalog-custom-fields',
        'Manage dynamic custom attributes and entity extensions',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "catalog.CustomFields", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000515',
        v_gold_copy_id,
        'Data Domains',
        'core-domains',
        'Business data domains, ownership boundaries, and semantic scope',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "catalog.DomainsManagement", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000516',
        v_gold_copy_id,
        'Schema Explorer',
        'catalog-schema-explorer',
        'Deep introspection of underlying database tables, views, and columns',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "catalog.SchemaExplorer", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000517',
        v_gold_copy_id,
        'Semantic Mapper',
        'core-semantic-mapper',
        'Map physical columns and upstream assets to canonical business concepts',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "catalog.SemanticMapper", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000518',
        v_gold_copy_id,
        'AI Term Suggestions',
        'catalog-ai-suggestions',
        'AI-assisted terminology recommendations, match confidence, and review workflow',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "catalog.AISuggestions", "inputs": {}}}}'::jsonb,
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
            (v_tenant.tenant_id, '/catalog/api-inventory', 'catalog-api-inventory', NOW()),
            (v_tenant.tenant_id, '/core/glossary', 'catalog-glossary', NOW()),
            (v_tenant.tenant_id, '/core/semantic-terms', 'catalog-glossary', NOW()),
            (v_tenant.tenant_id, '/catalog/semantic-terms', 'catalog-glossary', NOW()),
            (v_tenant.tenant_id, '/core/business-terms', 'catalog-business-terms', NOW()),
            (v_tenant.tenant_id, '/catalog/business-terms', 'catalog-business-terms', NOW()),
            (v_tenant.tenant_id, '/catalog/custom-fields', 'catalog-custom-fields', NOW()),
            (v_tenant.tenant_id, '/core/domains', 'core-domains', NOW()),
            (v_tenant.tenant_id, '/schema-explorer', 'catalog-schema-explorer', NOW()),
            (v_tenant.tenant_id, '/core/semantic-mapper', 'core-semantic-mapper', NOW()),
            (v_tenant.tenant_id, '/catalog/ai-suggestions', 'catalog-ai-suggestions', NOW())
        ON CONFLICT (tenant_id, path) DO UPDATE
            SET page_key = EXCLUDED.page_key;
    END LOOP;

    -- 3. Repoint navigation_menu_nodes to page keys and clear target_route
    UPDATE navigation_menu_nodes SET target_page_key = 'catalog-api-inventory', target_route = NULL WHERE node_key = 'catalog-glossary-api-inventory';
    UPDATE navigation_menu_nodes SET target_page_key = 'catalog-glossary', target_route = NULL WHERE node_key = 'catalog-glossary-semantic-terms';
    UPDATE navigation_menu_nodes SET target_page_key = 'catalog-business-terms', target_route = NULL WHERE node_key = 'catalog-glossary-business-terms';
    UPDATE navigation_menu_nodes SET target_page_key = 'catalog-custom-fields', target_route = NULL WHERE node_key = 'build-data-manage-custom-fields';
    UPDATE navigation_menu_nodes SET target_page_key = 'core-domains', target_route = NULL WHERE node_key = 'catalog-glossary-data-domains';
    UPDATE navigation_menu_nodes SET target_page_key = 'catalog-schema-explorer', target_route = NULL WHERE node_key = 'catalog-glossary-datasource-explorer';
    UPDATE navigation_menu_nodes SET target_page_key = 'core-semantic-mapper', target_route = NULL WHERE node_key = 'catalog-config-semantic-mapper';
    UPDATE navigation_menu_nodes SET target_page_key = 'catalog-ai-suggestions', target_route = NULL WHERE node_key = 'catalog-config-ai-term-suggestions';

END $$;
