-- Migration 20261129_001_seed_validation_rules_page.up.sql
-- Seeds the core Page Studio page definition and navigation menu node for Validation Rules & Rule Studio

INSERT INTO public.page_definitions (
    id, tenant_id, name, slug, description, layout, tabs, components, data_sources,
    presentation_events, filter_bar, app_model, version, is_core, status, updated_at
) VALUES (
    '018f9d02-0001-7000-8000-000000000100',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'Validation rules & rule studio',
    'validation-rules',
    'Centralized single-store validation catalog, portable AST bundles, and live evaluation engine.',
    $layout${"root": "catalog_root", "nodes": {"catalog_root": {"id": "catalog_root", "type": "Column", "children": ["porter_widget", "rules_grid"], "style": {"gap": "16px"}}}}$layout$::jsonb,
    $tabs$[{"id": "catalog", "label": "Rule Catalog & GitOps Porter", "layout": {"root": "catalog_root", "nodes": {"catalog_root": {"id": "catalog_root", "type": "Column", "children": ["porter_widget", "rules_grid"], "style": {"gap": "16px"}}}}}, {"id": "evaluator", "label": "Interactive Rule Tester", "layout": {"root": "eval_root", "nodes": {"eval_root": {"id": "eval_root", "type": "Column", "children": ["eval_tester_widget"], "style": {"gap": "16px"}}}}}, {"id": "violations", "label": "Live Violations Monitor", "layout": {"root": "violation_root", "nodes": {"violation_root": {"id": "violation_root", "type": "Column", "children": ["violations_widget"], "style": {"gap": "16px"}}}}}] $tabs$::jsonb,
    $comp${"hdr": {"id": "hdr", "type": "PageHeader", "props": {"icon": "rule", "title": "Validation Rules & Rulefabric Studio", "subtitle": "Centralized single-store validation catalog, portable AST bundles, and live evaluation engine"}, "style": {"flex": "1 1 320px"}}, "bo_select": {"id": "bo_select", "type": "VariableSelect", "props": {"variable": "selected_bo", "label": "Business Object", "minWidth": 180, "options": [{"value": "order", "label": "Order (orm.order)"}, {"value": "execution", "label": "Execution (orm.execution)"}, {"value": "security", "label": "Security (oms.security)"}, {"value": "account", "label": "Account (oms.account)"}, {"value": "party", "label": "Party / Customer (master.customer)"}]}, "style": {"flex": "0 0 auto"}}, "domain_select": {"id": "domain_select", "type": "VariableSelect", "props": {"variable": "domain_filter", "label": "Domain Scope", "emptyLabel": "All Domains", "minWidth": 160, "options": [{"value": "validation", "label": "Business Validation"}, {"value": "mdm", "label": "MDM Data Quality"}, {"value": "compliance", "label": "Regulatory & Compliance"}]}, "style": {"flex": "0 0 auto"}}, "porter_widget": {"id": "porter_widget", "type": "validationRules.RuleBundleManager", "props": {"bo_name": "{{vars.selected_bo}}", "domain": "{{vars.domain_filter}}"}}, "rules_grid": {"id": "rules_grid", "type": "DataGrid", "props": {"query": "rules_list", "rowsPath": "rules", "rowKey": "id", "enableSearch": true, "searchFields": ["name", "rule_key", "description"], "emptyText": "No validation rules found for this Business Object.", "columns": [{"id": "rule_key", "header": "Rule Key", "cell": {"kind": "text", "value": "{{row.rule_key}}"}}, {"id": "name", "header": "Rule Name", "cell": {"kind": "text", "value": "{{row.name}}"}}, {"id": "bo", "header": "Business Object", "cell": {"kind": "chip", "value": "{{row.bo_name}}", "variant": "outlined"}}, {"id": "severity", "header": "Severity", "cell": {"kind": "chip", "value": "{{row.severity}}", "colorMap": {"BLOCK": "error", "WARN": "warning"}}}, {"id": "timing", "header": "Timing", "cell": {"kind": "text", "value": "{{row.timing}}"}}, {"id": "origin", "header": "Origin", "cell": {"kind": "chip", "value": "{{row.origin}}", "colorMap": {"core": "primary", "custom": "default"}}}, {"id": "status", "header": "Active", "cell": {"kind": "chip", "value": "{{row.is_active}}", "colorMap": {"true": "success", "false": "default"}}}]}}, "eval_tester_widget": {"id": "eval_tester_widget", "type": "validationRules.RuleTesterPanel", "props": {"default_bo_name": "{{vars.selected_bo}}"}}, "violations_widget": {"id": "violations_widget", "type": "validationRules.ViolationsLiveViewer", "props": {"limit": 100}}}$comp$::jsonb,
    $ds$[]$ds$::jsonb,
    $pe$[]$pe$::jsonb,
    $fb${"root": "filter_root", "nodes": {"filter_root": {"id": "filter_root", "type": "Column", "children": ["header_row"], "style": {"gap": "12px"}}, "header_row": {"id": "header_row", "type": "Row", "children": ["hdr", "bo_select", "domain_select"], "style": {"alignItems": "center", "gap": "8px", "flexWrap": "wrap"}}}}$fb$::jsonb,
    $app${"chrome": "none", "surface": {"maxWidth": 1400, "padding": 3}, "tabVariable": "tab", "variables": [{"name": "selected_bo", "default": "order", "url": true}, {"name": "domain_filter", "default": "", "url": true}, {"name": "tab", "default": "catalog", "url": true}], "queries": [{"id": "rules_list", "operation": "validationRules.list", "params": {"bo_name": "{{vars.selected_bo}}", "domain": "{{vars.domain_filter}}"}}]}$app$::jsonb,
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
    'e81a3d02-0001-7000-8000-000000000100',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '4ef1676b-e9aa-4012-9e44-bba9091e962d',
    'validation-rules',
    'Validation rules & rule studio',
    'validation-rules',
    14,
    'BASE_USER'
)
ON CONFLICT (tenant_id, node_key) DO UPDATE SET
    label = EXCLUDED.label,
    target_page_key = EXCLUDED.target_page_key,
    display_order = EXCLUDED.display_order;
