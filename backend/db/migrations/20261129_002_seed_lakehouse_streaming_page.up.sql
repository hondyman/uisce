-- Migration 20261129_002_seed_lakehouse_streaming_page.up.sql
-- Seeds the core Page Studio page definition and navigation menu node for Lakehouse & CDC Stream Ingestion

INSERT INTO public.page_definitions (
    id, tenant_id, name, slug, description, layout, tabs, components, data_sources,
    presentation_events, filter_bar, app_model, version, is_core, status, updated_at
) VALUES (
    '018f9d02-0001-7000-8000-000000000200',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'Lakehouse & CDC Stream Ingestion',
    'lakehouse-streaming',
    'Apache Iceberg REST catalog management, Debezium CDC ingestion pipelines, Layer 2 tenant assertion, and streaming gatekeeper inspection.',
    $layout${"root": "telemetry_root", "nodes": {"telemetry_root": {"id": "telemetry_root", "type": "Column", "children": ["gatekeeper_widget"], "style": {"gap": "16px"}}}}$layout$::jsonb,
    $tabs$[{"id": "telemetry", "label": "Ingestion & Gatekeeper", "layout": {"root": "telemetry_root", "nodes": {"telemetry_root": {"id": "telemetry_root", "type": "Column", "children": ["gatekeeper_widget"], "style": {"gap": "16px"}}}}}, {"id": "catalog", "label": "Iceberg & Lakekeeper Catalog", "layout": {"root": "catalog_root", "nodes": {"catalog_root": {"id": "catalog_root", "type": "Column", "children": ["catalog_widget"], "style": {"gap": "16px"}}}}}, {"id": "dlq", "label": "Dead-Letter Queue (DLQ)", "layout": {"root": "dlq_root", "nodes": {"dlq_root": {"id": "dlq_root", "type": "Column", "children": ["dlq_widget"], "style": {"gap": "16px"}}}}}] $tabs$::jsonb,
    $comp${"hdr": {"id": "hdr", "type": "PageHeader", "props": {"icon": "stream", "title": "Lakehouse & CDC Stream Ingestion", "subtitle": "Apache Iceberg Lakekeeper REST catalog, Debezium CDC gatekeeper, Layer 2 Tenant Assertion, and DLQ telemetry"}, "style": {"flex": "1 1 320px"}}, "gatekeeper_widget": {"id": "gatekeeper_widget", "type": "lakehouse.StreamingGatekeeperMonitor", "props": {}}, "catalog_widget": {"id": "catalog_widget", "type": "lakehouse.LakekeeperCatalogManager", "props": {}}, "dlq_widget": {"id": "dlq_widget", "type": "lakehouse.DLQInspectorPanel", "props": {}}}$comp$::jsonb,
    $ds$[]$ds$::jsonb,
    $pe$[]$pe$::jsonb,
    $fb${"root": "filter_root", "nodes": {"filter_root": {"id": "filter_root", "type": "Column", "children": ["header_row"], "style": {"gap": "12px"}}, "header_row": {"id": "header_row", "type": "Row", "children": ["hdr"], "style": {"alignItems": "center", "gap": "8px", "flexWrap": "wrap"}}}}$fb$::jsonb,
    $app${"chrome": "none", "surface": {"maxWidth": 1400, "padding": 3}, "tabVariable": "tab", "variables": [{"name": "tab", "default": "telemetry", "url": true}], "queries": [{"id": "lakehouse_overview", "operation": "lakehouse.overview", "params": {}}]}$app$::jsonb,
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

-- Add to Master Data / Governance navigation menu
INSERT INTO public.navigation_menu_nodes (
    id, tenant_id, parent_id, node_key, label, target_page_key, display_order, required_entitlement
) VALUES (
    'e81a3d02-0001-7000-8000-000000000200',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '4ef1676b-e9aa-4012-9e44-bba9091e962d',
    'lakehouse-streaming',
    'Lakehouse & CDC Stream Ingestion',
    'lakehouse-streaming',
    15,
    'BASE_USER'
)
ON CONFLICT (tenant_id, node_key) DO UPDATE SET
    label = EXCLUDED.label,
    target_page_key = EXCLUDED.target_page_key,
    display_order = EXCLUDED.display_order;
