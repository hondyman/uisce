-- Migration: 20261225_008_seed_all_remaining_pages.up.sql
-- Seeds page definitions, route aliases, and updates menu nodes for all remaining screens
-- ensuring 100% single-source-of-truth metadata navigation.

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
    -- Intelligence & Observability
    (
        '018f9d02-0001-7000-8000-000000000801',
        v_gold_copy_id,
        'Intelligence Dashboard',
        'intelligence-dashboard',
        'Autonomous optimization insights, workload signals, and recommendations.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "analytics.IntelligenceDashboard", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000802',
        v_gold_copy_id,
        'Index Advisor',
        'intelligence-index-advisor',
        'AI-driven index synthesis, workload pattern analysis, and query tuning.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "analytics.IndexAdvisor", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000803',
        v_gold_copy_id,
        'Storage Tiering',
        'intelligence-storage',
        'Storage efficiency, cold tier archival, and cost optimization telemetry.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "analytics.StorageTiering", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000804',
        v_gold_copy_id,
        'Data Quality',
        'intelligence-data-quality',
        'Anomaly detection, freshness monitoring, and schema drift metrics.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "analytics.DataQuality", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000805',
        v_gold_copy_id,
        'ASO Center',
        'optimization-aso',
        'Autonomous Storage & Query Optimization control center.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "analytics.OptimizationCenter", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000806',
        v_gold_copy_id,
        'Metrics Dashboard',
        'observability-dashboard',
        'Real-time telemetry, latency percentiles, and database metrics.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "analytics.Observability", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000807',
        v_gold_copy_id,
        'SLO Dashboard',
        'observability-slos',
        'Service level objectives, error budget burn rates, and alerts.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "analytics.SLODashboard", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000808',
        v_gold_copy_id,
        'Natural Language Query',
        'nlq-page',
        'Natural language query exploration over semantic models.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "analytics.NLQ", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000809',
        v_gold_copy_id,
        'Global Intelligence',
        'global-intelligence',
        'Unified cross-domain enterprise search and intelligence assistant.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "analytics.GlobalNLQ", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000810',
        v_gold_copy_id,
        'Scenario Analysis',
        'analytics-scenario-analysis',
        'Interactive stress testing, macroeconomic shocks, and portfolio what-if modeling.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "analytics.ScenarioAnalysis", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000811',
        v_gold_copy_id,
        'Portfolio Master',
        'analytics-portfolio-master',
        'Comprehensive multi-asset portfolio positioning, exposure, and attribution.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "analytics.PortfolioMaster", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000812',
        v_gold_copy_id,
        'Advisor Dashboard',
        'analytics-advisor-dashboard',
        'Client wealth dashboard, household views, and personalized action recommendations.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "analytics.AdvisorDashboard", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    -- Fabric & Reporting
    (
        '018f9d02-0001-7000-8000-000000000813',
        v_gold_copy_id,
        'Calculations Library',
        'fabric-calculations',
        'Central registry of metrics, business formulas, and reusable calculations.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "fabric.CalculationsLibrary", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000814',
        v_gold_copy_id,
        'Preaggregations',
        'fabric-preaggregations',
        'Pre-computed rollup cubes, partition refreshes, and acceleration telemetry.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "fabric.Preaggregations", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000815',
        v_gold_copy_id,
        'Fabric Dashboard',
        'fabric-dashboard',
        'Unified semantic fabric overview, node topology, and performance monitors.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "fabric.Dashboard", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000816',
        v_gold_copy_id,
        'Fabric Settings',
        'fabric-settings',
        'System configurations, caching policies, and query engine routing rules.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "fabric.Settings", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000817',
        v_gold_copy_id,
        'Report Library',
        'reports-library',
        'Enterprise catalog of published executive reports, templates, and analytics.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "fabric.ReportLibrary", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000818',
        v_gold_copy_id,
        'Report Builder',
        'reports-builder',
        'Interactive visual designer for building reports and parameterized views.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "fabric.ReportBuilder", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000819',
        v_gold_copy_id,
        'Query Builder',
        'reports-queries',
        'Visual query builder and saved query management repository.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "fabric.QueryLibrary", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000820',
        v_gold_copy_id,
        'Semantic Models',
        'reports-models',
        'Manage analytical semantic models, dimensions, and measures.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "fabric.SemanticModels", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000821',
        v_gold_copy_id,
        'Bundle Explorer',
        'bundle-explorer',
        'Inspect micro-bundles, metadata packages, and cross-tenant exports.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "fabric.BundleExplorer", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000822',
        v_gold_copy_id,
        'Views Catalog',
        'views',
        'Catalog of materialized, logical, and dynamic semantic views.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "fabric.ViewsCatalog", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000823',
        v_gold_copy_id,
        'Business Objects',
        'business-objects',
        'Enterprise business entities, driving tables, and bitemporal mappings.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "fabric.BusinessObjects", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    -- Security & Admin
    (
        '018f9d02-0001-7000-8000-000000000824',
        v_gold_copy_id,
        'LLM Configuration',
        'admin-llm',
        'Manage LLM models, API keys, temperature settings, and tenant tokens.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "security.LLMConfig", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000825',
        v_gold_copy_id,
        'Temporal Operations',
        'admin-temporal-ops',
        'Temporal workflow orchestration, task queue telemetry, and failure recoveries.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "security.TemporalOps", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000826',
        v_gold_copy_id,
        'Entitlement Management',
        'admin-entitlements',
        'Fine-grained attribute-based access control, profile permissions, and matrices.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "security.EntitlementMatrix", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000827',
        v_gold_copy_id,
        'Audit Log',
        'admin-audit',
        'Tamper-evident audit trail for system events, authorization decisions, and changes.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "security.AuditLog", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000828',
        v_gold_copy_id,
        'Manage Resources',
        'tenants-management',
        'Multi-tenant provisioning, database quotas, and isolation policies.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "security.TenantsManagement", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000829',
        v_gold_copy_id,
        'Access Rules',
        'security-access-rules',
        'Security policies, zero-trust constraints, and dynamic authorization rules.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "security.AccessRules", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000830',
        v_gold_copy_id,
        'Access Explanation',
        'access-explanation',
        'Inspect effective user entitlements, RBAC/ABAC rationale, and grant lineage.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "security.AccessExplanation", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000831',
        v_gold_copy_id,
        'JIT Elevation Requests',
        'jit-requests',
        'Just-in-Time privilege escalation requests, approvals, and expiration timers.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "security.JITRequests", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000832',
        v_gold_copy_id,
        'Secrets',
        'secrets-config',
        'Secure credentials, API tokens, encryption key rotation, and KMS config.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "security.Secrets", "inputs": {}}}}'::jsonb,
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
        updated_at = NOW();

    -- 2. Populate Route Aliases for all tenants
    FOR v_tenant IN SELECT tenant_id FROM tenants LOOP
        INSERT INTO route_aliases (tenant_id, path, page_key, created_at)
        VALUES
            -- Intelligence & Observability
            (v_tenant.tenant_id, '/intelligence', 'intelligence-dashboard', NOW()),
            (v_tenant.tenant_id, '/intelligence/index-advisor', 'intelligence-index-advisor', NOW()),
            (v_tenant.tenant_id, '/intelligence/storage', 'intelligence-storage', NOW()),
            (v_tenant.tenant_id, '/intelligence/data-quality', 'intelligence-data-quality', NOW()),
            (v_tenant.tenant_id, '/optimization', 'optimization-aso', NOW()),
            (v_tenant.tenant_id, '/observability', 'observability-dashboard', NOW()),
            (v_tenant.tenant_id, '/observability/slos', 'observability-slos', NOW()),
            (v_tenant.tenant_id, '/nlq', 'nlq-page', NOW()),
            (v_tenant.tenant_id, '/global-intelligence', 'global-intelligence', NOW()),
            (v_tenant.tenant_id, '/analytics/scenario-analysis', 'analytics-scenario-analysis', NOW()),
            (v_tenant.tenant_id, '/analytics/portfolio-master', 'analytics-portfolio-master', NOW()),
            (v_tenant.tenant_id, '/analytics/advisor-dashboard', 'analytics-advisor-dashboard', NOW()),
            -- Fabric & Reporting
            (v_tenant.tenant_id, '/fabric/calculations', 'fabric-calculations', NOW()),
            (v_tenant.tenant_id, '/fabric/preaggregations', 'fabric-preaggregations', NOW()),
            (v_tenant.tenant_id, '/fabric/dashboard', 'fabric-dashboard', NOW()),
            (v_tenant.tenant_id, '/fabric/settings', 'fabric-settings', NOW()),
            (v_tenant.tenant_id, '/reports/library', 'reports-library', NOW()),
            (v_tenant.tenant_id, '/reports/builder', 'reports-builder', NOW()),
            (v_tenant.tenant_id, '/reports/queries', 'reports-queries', NOW()),
            (v_tenant.tenant_id, '/reports/models', 'reports-models', NOW()),
            (v_tenant.tenant_id, '/bundle-explorer', 'bundle-explorer', NOW()),
            (v_tenant.tenant_id, '/fabric/bundles', 'bundle-explorer', NOW()),
            (v_tenant.tenant_id, '/views', 'views', NOW()),
            (v_tenant.tenant_id, '/business-objects', 'business-objects', NOW()),
            -- Security & Admin
            (v_tenant.tenant_id, '/admin/llm', 'admin-llm', NOW()),
            (v_tenant.tenant_id, '/admin/temporal-ops', 'admin-temporal-ops', NOW()),
            (v_tenant.tenant_id, '/admin/entitlements', 'admin-entitlements', NOW()),
            (v_tenant.tenant_id, '/audit', 'admin-audit', NOW()),
            (v_tenant.tenant_id, '/tenants', 'tenants-management', NOW()),
            (v_tenant.tenant_id, '/security/access-rules', 'security-access-rules', NOW()),
            (v_tenant.tenant_id, '/access-explanation', 'access-explanation', NOW()),
            (v_tenant.tenant_id, '/jit-request', 'jit-requests', NOW()),
            (v_tenant.tenant_id, '/secrets/config', 'secrets-config', NOW())
        ON CONFLICT (tenant_id, path) DO UPDATE
        SET page_key = EXCLUDED.page_key;
    END LOOP;

    -- 3. Repoint all remaining Navigation Menu Nodes
    -- Intelligence & Observability
    UPDATE navigation_menu_nodes SET target_page_key = 'intelligence-dashboard', target_route = NULL WHERE target_route = '/intelligence';
    UPDATE navigation_menu_nodes SET target_page_key = 'intelligence-index-advisor', target_route = NULL WHERE target_route = '/intelligence/index-advisor';
    UPDATE navigation_menu_nodes SET target_page_key = 'intelligence-storage', target_route = NULL WHERE target_route = '/intelligence/storage';
    UPDATE navigation_menu_nodes SET target_page_key = 'intelligence-data-quality', target_route = NULL WHERE target_route = '/intelligence/data-quality';
    UPDATE navigation_menu_nodes SET target_page_key = 'optimization-aso', target_route = NULL WHERE target_route = '/optimization';
    UPDATE navigation_menu_nodes SET target_page_key = 'observability-dashboard', target_route = NULL WHERE target_route = '/observability';
    UPDATE navigation_menu_nodes SET target_page_key = 'observability-slos', target_route = NULL WHERE target_route = '/observability/slos';
    UPDATE navigation_menu_nodes SET target_page_key = 'nlq-page', target_route = NULL WHERE target_route = '/nlq';
    UPDATE navigation_menu_nodes SET target_page_key = 'global-intelligence', target_route = NULL WHERE target_route = '/global-intelligence';
    UPDATE navigation_menu_nodes SET target_page_key = 'analytics-scenario-analysis', target_route = NULL WHERE target_route = '/analytics/scenario-analysis';
    UPDATE navigation_menu_nodes SET target_page_key = 'analytics-portfolio-master', target_route = NULL WHERE target_route = '/analytics/portfolio-master';
    UPDATE navigation_menu_nodes SET target_page_key = 'analytics-advisor-dashboard', target_route = NULL WHERE target_route = '/analytics/advisor-dashboard';

    -- Fabric & Reporting
    UPDATE navigation_menu_nodes SET target_page_key = 'fabric-calculations', target_route = NULL WHERE target_route = '/fabric/calculations';
    UPDATE navigation_menu_nodes SET target_page_key = 'fabric-preaggregations', target_route = NULL WHERE target_route = '/fabric/preaggregations';
    UPDATE navigation_menu_nodes SET target_page_key = 'fabric-dashboard', target_route = NULL WHERE target_route = '/fabric/dashboard';
    UPDATE navigation_menu_nodes SET target_page_key = 'fabric-settings', target_route = NULL WHERE target_route = '/fabric/settings';
    UPDATE navigation_menu_nodes SET target_page_key = 'reports-library', target_route = NULL WHERE target_route = '/reports/library';
    UPDATE navigation_menu_nodes SET target_page_key = 'reports-builder', target_route = NULL WHERE target_route = '/reports/builder';
    UPDATE navigation_menu_nodes SET target_page_key = 'reports-queries', target_route = NULL WHERE target_route = '/reports/queries';
    UPDATE navigation_menu_nodes SET target_page_key = 'reports-models', target_route = NULL WHERE target_route = '/reports/models';
    UPDATE navigation_menu_nodes SET target_page_key = 'bundle-explorer', target_route = NULL WHERE target_route IN ('/bundle-explorer', '/fabric/bundles');
    UPDATE navigation_menu_nodes SET target_page_key = 'views', target_route = NULL WHERE target_route = '/views';
    UPDATE navigation_menu_nodes SET target_page_key = 'business-objects', target_route = NULL WHERE target_route = '/business-objects';

    -- Security & Admin
    UPDATE navigation_menu_nodes SET target_page_key = 'admin-llm', target_route = NULL WHERE target_route = '/admin/llm';
    UPDATE navigation_menu_nodes SET target_page_key = 'admin-temporal-ops', target_route = NULL WHERE target_route = '/admin/temporal-ops';
    UPDATE navigation_menu_nodes SET target_page_key = 'admin-entitlements', target_route = NULL WHERE target_route = '/admin/entitlements';
    UPDATE navigation_menu_nodes SET target_page_key = 'admin-audit', target_route = NULL WHERE target_route = '/audit';
    UPDATE navigation_menu_nodes SET target_page_key = 'tenants-management', target_route = NULL WHERE target_route = '/tenants';
    UPDATE navigation_menu_nodes SET target_page_key = 'security-access-rules', target_route = NULL WHERE target_route = '/security/access-rules';
    UPDATE navigation_menu_nodes SET target_page_key = 'access-explanation', target_route = NULL WHERE target_route = '/access-explanation';
    UPDATE navigation_menu_nodes SET target_page_key = 'jit-requests', target_route = NULL WHERE target_route = '/jit-request';
    UPDATE navigation_menu_nodes SET target_page_key = 'secrets-config', target_route = NULL WHERE target_route = '/secrets/config';

    -- Low-Code Designers & Tools
    UPDATE navigation_menu_nodes SET target_page_key = 'menu-designer', target_route = NULL WHERE target_route = '/menu-designer';
    UPDATE navigation_menu_nodes SET target_page_key = 'page-studio', target_route = NULL WHERE target_route = '/page-studio';
    UPDATE navigation_menu_nodes SET target_page_key = 'api-studio', target_route = NULL WHERE target_route = '/api-studio';
    UPDATE navigation_menu_nodes SET target_page_key = 'catalog-api-inventory', target_route = NULL WHERE target_route = '/api-catalog';

END $$;
