-- Migration 20261225_003_seed_catalog_and_platform_pages.up.sql
-- Seeds core Page Studio pages for Node Types, Edge Types, Abbreviations, and repoints navigation menu nodes.

INSERT INTO public.page_definitions (
    id, tenant_id, name, slug, description, layout, tabs, components, data_sources,
    presentation_events, filter_bar, app_model, version, is_core, status, updated_at
) VALUES (
    '018f9d02-0001-7000-8000-000000000701',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'Node Types',
    'catalog-node-types',
    'Metadata structures and node types in the catalog graph',
    $layout${"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["controls_row", "grid", "create_dialog"], "style": {"gap": "16px"}}, "controls_row": {"id": "controls_row", "type": "Row", "children": ["search", "create_btn"], "style": {"alignItems": "center", "gap": "12px"}}}}$layout$::jsonb,
    '[]'::jsonb,
    $comp${"header": {"id": "header", "type": "PageHeader", "props": {"icon": "schema", "title": "Node Types", "subtitle": "Metadata structures and node types in the catalog graph"}}, "search": {"id": "search", "type": "SearchInput", "props": {"variable": "search", "placeholder": "Search node types..."}, "style": {"flex": "1 1 300px"}}, "create_btn": {"id": "create_btn", "type": "ActionButton", "props": {"label": "New Node Type", "variant": "contained", "onClick": [{"kind": "setVariable", "name": "createOpen", "value": true}]}, "style": {"flex": "0 0 auto"}}, "grid": {"id": "grid", "type": "DataGrid", "props": {"query": "nodeTypes", "rowKey": "id", "enableSearch": true, "searchVariable": "search", "searchFields": ["catalog_type_name", "description", "type"], "columns": [{"id": "name", "header": "Catalog Type Name", "cell": {"kind": "text", "value": "{{row.catalog_type_name}}"}}, {"id": "type", "header": "Origin", "cell": {"kind": "chip", "value": "{{row.type}}", "colorMap": {"core": "primary", "custom": "success"}}}, {"id": "desc", "header": "Description", "cell": {"kind": "text", "value": "{{row.description}}"}}, {"id": "status", "header": "Status", "cell": {"kind": "chip", "value": "{{row.is_active}}", "labelKey": "{{row.is_active ? \"Active\" : \"Inactive\"}}", "colorMap": {"true": "success", "false": "default"}}}, {"id": "act", "header": "", "cell": {"kind": "actions", "buttons": [{"label": "Delete", "color": "error", "visibleWhen": {"type": "condition", "field": "row.type", "operator": "equals", "value": "custom"}, "onClick": [{"kind": "runOperation", "operation": "catalog.deleteNodeType", "params": {"id": "{{row.id}}"}, "successMessage": "Node type deleted"}]}]}, "align": "right"}]}}, "create_dialog": {"id": "create_dialog", "type": "Dialog", "props": {"title": "Create Custom Node Type", "openWhen": {"type": "condition", "field": "vars.createOpen", "operator": "is_true"}, "onClose": [{"kind": "setVariable", "name": "createOpen", "value": false}]}, "children": ["create_form"]}, "create_form": {"id": "create_form", "type": "Form", "props": {"variable": "createForm", "fields": [{"name": "catalog_type_name", "label": "Node Type Name", "kind": "text", "required": true}, {"name": "description", "label": "Description", "kind": "multiline", "required": false}], "submitLabel": "Create", "onSubmit": [{"kind": "runOperation", "operation": "catalog.createNodeType", "params": {"catalog_type_name": "{{form.catalog_type_name}}", "description": "{{form.description}}"}, "successMessage": "Node type created successfully", "onSuccess": [{"kind": "setVariable", "name": "createOpen", "value": false}]}]}}}$comp$::jsonb,
    '[]'::jsonb,
    '[]'::jsonb,
    $fb${"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"alignItems": "center", "width": "100%"}}}}$fb$::jsonb,
    $app${"chrome": "none", "surface": {"maxWidth": 1400, "padding": 3}, "variables": [{"name": "search", "default": "", "url": true}, {"name": "createOpen", "default": false}, {"name": "createForm", "default": {"catalog_type_name": "", "description": ""}}], "queries": [{"id": "nodeTypes", "operation": "catalog.listNodeTypes"}]}$app$::jsonb,
    1,
    true,
    'published',
    NOW()
)
ON CONFLICT (tenant_id, slug) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    layout = EXCLUDED.layout,
    tabs = EXCLUDED.tabs,
    components = EXCLUDED.components,
    data_sources = EXCLUDED.data_sources,
    presentation_events = EXCLUDED.presentation_events,
    filter_bar = EXCLUDED.filter_bar,
    app_model = EXCLUDED.app_model,
    version = EXCLUDED.version,
    is_core = EXCLUDED.is_core,
    status = EXCLUDED.status,
    updated_at = NOW();

-- Repoint navigation menu node from target_route to target_page_key
UPDATE public.navigation_menu_nodes
SET target_page_key = 'catalog-node-types', target_route = NULL
WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND (node_key = 'catalog-config-node-types' OR target_route = '/catalog/node-types');

-- Register route alias
INSERT INTO public.route_aliases (tenant_id, path, page_key)
VALUES ('99e99e99-99e9-49e9-89e9-99e99e99e999', '/catalog/node-types', 'catalog-node-types')
ON CONFLICT (tenant_id, path) DO UPDATE SET page_key = EXCLUDED.page_key;

INSERT INTO public.page_definitions (
    id, tenant_id, name, slug, description, layout, tabs, components, data_sources,
    presentation_events, filter_bar, app_model, version, is_core, status, updated_at
) VALUES (
    '018f9d02-0001-7000-8000-000000000702',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'Edge Types',
    'catalog-edge-types',
    'Relationship types and connections in the catalog graph',
    $layout${"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["controls_row", "grid"], "style": {"gap": "16px"}}, "controls_row": {"id": "controls_row", "type": "Row", "children": ["search"], "style": {"alignItems": "center", "gap": "12px"}}}}$layout$::jsonb,
    '[]'::jsonb,
    $comp${"header": {"id": "header", "type": "PageHeader", "props": {"icon": "accountTree", "title": "Edge Types", "subtitle": "Relationship types and connections in the catalog graph"}}, "search": {"id": "search", "type": "SearchInput", "props": {"variable": "search", "placeholder": "Search edge types..."}, "style": {"flex": "1 1 300px"}}, "grid": {"id": "grid", "type": "DataGrid", "props": {"query": "edgeTypes", "rowKey": "id", "enableSearch": true, "searchVariable": "search", "searchFields": ["edge_type_name", "description", "type"], "columns": [{"id": "name", "header": "Edge Type Name", "cell": {"kind": "text", "value": "{{row.edge_type_name}}"}}, {"id": "type", "header": "Origin", "cell": {"kind": "chip", "value": "{{row.type}}", "colorMap": {"core": "primary", "custom": "success"}}}, {"id": "desc", "header": "Description", "cell": {"kind": "text", "value": "{{row.description}}"}}, {"id": "status", "header": "Status", "cell": {"kind": "chip", "value": "{{row.is_active}}", "labelKey": "{{row.is_active ? \"Active\" : \"Inactive\"}}", "colorMap": {"true": "success", "false": "default"}}}]}}}$comp$::jsonb,
    '[]'::jsonb,
    '[]'::jsonb,
    $fb${"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"alignItems": "center", "width": "100%"}}}}$fb$::jsonb,
    $app${"chrome": "none", "surface": {"maxWidth": 1400, "padding": 3}, "variables": [{"name": "search", "default": "", "url": true}], "queries": [{"id": "edgeTypes", "operation": "catalog.listEdgeTypes"}]}$app$::jsonb,
    1,
    true,
    'published',
    NOW()
)
ON CONFLICT (tenant_id, slug) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    layout = EXCLUDED.layout,
    tabs = EXCLUDED.tabs,
    components = EXCLUDED.components,
    data_sources = EXCLUDED.data_sources,
    presentation_events = EXCLUDED.presentation_events,
    filter_bar = EXCLUDED.filter_bar,
    app_model = EXCLUDED.app_model,
    version = EXCLUDED.version,
    is_core = EXCLUDED.is_core,
    status = EXCLUDED.status,
    updated_at = NOW();

-- Repoint navigation menu node from target_route to target_page_key
UPDATE public.navigation_menu_nodes
SET target_page_key = 'catalog-edge-types', target_route = NULL
WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND (node_key = 'catalog-config-edge-types' OR target_route = '/catalog/edge-types');

-- Register route alias
INSERT INTO public.route_aliases (tenant_id, path, page_key)
VALUES ('99e99e99-99e9-49e9-89e9-99e99e99e999', '/catalog/edge-types', 'catalog-edge-types')
ON CONFLICT (tenant_id, path) DO UPDATE SET page_key = EXCLUDED.page_key;

INSERT INTO public.page_definitions (
    id, tenant_id, name, slug, description, layout, tabs, components, data_sources,
    presentation_events, filter_bar, app_model, version, is_core, status, updated_at
) VALUES (
    '018f9d02-0001-7000-8000-000000000703',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'Abbreviations',
    'core-abbreviations',
    'Standard business abbreviations and their full semantic expansions',
    $layout${"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["controls_row", "grid"], "style": {"gap": "16px"}}, "controls_row": {"id": "controls_row", "type": "Row", "children": ["search"], "style": {"alignItems": "center", "gap": "12px"}}}}$layout$::jsonb,
    '[]'::jsonb,
    $comp${"header": {"id": "header", "type": "PageHeader", "props": {"icon": "textFields", "title": "Abbreviations", "subtitle": "Standard business abbreviations and their full semantic expansions"}}, "search": {"id": "search", "type": "SearchInput", "props": {"variable": "search", "placeholder": "Search abbreviations..."}, "style": {"flex": "1 1 300px"}}, "grid": {"id": "grid", "type": "DataGrid", "props": {"query": "abbreviations", "rowKey": "id", "enableSearch": true, "searchVariable": "search", "searchFields": ["abbreviation", "expansion", "domain"], "columns": [{"id": "abbrev", "header": "Abbreviation", "cell": {"kind": "text", "value": "{{row.abbreviation}}"}}, {"id": "expansion", "header": "Expansion", "cell": {"kind": "text", "value": "{{row.expansion}}"}}, {"id": "domain", "header": "Domain", "cell": {"kind": "chip", "value": "{{row.domain}}"}}, {"id": "status", "header": "Status", "cell": {"kind": "chip", "value": "{{row.is_active}}", "labelKey": "{{row.is_active ? \"Active\" : \"Inactive\"}}", "colorMap": {"true": "success", "false": "default"}}}]}}}$comp$::jsonb,
    '[]'::jsonb,
    '[]'::jsonb,
    $fb${"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"alignItems": "center", "width": "100%"}}}}$fb$::jsonb,
    $app${"chrome": "none", "surface": {"maxWidth": 1400, "padding": 3}, "variables": [{"name": "search", "default": "", "url": true}], "queries": [{"id": "abbreviations", "operation": "catalog.listAbbreviations"}]}$app$::jsonb,
    1,
    true,
    'published',
    NOW()
)
ON CONFLICT (tenant_id, slug) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    layout = EXCLUDED.layout,
    tabs = EXCLUDED.tabs,
    components = EXCLUDED.components,
    data_sources = EXCLUDED.data_sources,
    presentation_events = EXCLUDED.presentation_events,
    filter_bar = EXCLUDED.filter_bar,
    app_model = EXCLUDED.app_model,
    version = EXCLUDED.version,
    is_core = EXCLUDED.is_core,
    status = EXCLUDED.status,
    updated_at = NOW();

-- Repoint navigation menu node from target_route to target_page_key
UPDATE public.navigation_menu_nodes
SET target_page_key = 'core-abbreviations', target_route = NULL
WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND (node_key = 'catalog-glossary-abbreviations' OR target_route = '/core/abbreviations');

-- Register route alias
INSERT INTO public.route_aliases (tenant_id, path, page_key)
VALUES ('99e99e99-99e9-49e9-89e9-99e99e99e999', '/core/abbreviations', 'core-abbreviations')
ON CONFLICT (tenant_id, path) DO UPDATE SET page_key = EXCLUDED.page_key;

