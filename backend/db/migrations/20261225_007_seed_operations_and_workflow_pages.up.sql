-- Migration: 20261225_007_seed_operations_and_workflow_pages.up.sql
-- Seeds page definitions, route aliases, and updates menu nodes for Workflow & Operations screens

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
        '018f9d02-0001-7000-8000-000000000601',
        v_gold_copy_id,
        'Process Catalog',
        'core-process-catalog',
        'Catalog of business processes, choreography, and execution history.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "workflow.ProcessCatalog", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000602',
        v_gold_copy_id,
        'Approval Inbox',
        'core-approval-inbox',
        'Unified queue for pending business and operational approvals.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "workflow.ApprovalInbox", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000603',
        v_gold_copy_id,
        'Approval Workflows',
        'core-approval-workflows',
        'Monitor multi-stage approval chains, escalation paths, and turnaround metrics.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "workflow.ApprovalWorkflows", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000604',
        v_gold_copy_id,
        'Process Designer',
        'client-workflow-studio',
        'Visual process modeling, workflow state machine authoring, and orchestration.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "workflow.WorkflowStudio", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000605',
        v_gold_copy_id,
        'Workflow Designer',
        'core-workflow-designer',
        'Step-by-step workflow canvas and node choreography.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "workflow.WorkflowDesigner", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000606',
        v_gold_copy_id,
        'Business Rules',
        'client-rules-editor',
        'Rule authoring, trigger matrices, and condition evaluation.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "workflow.BusinessRules", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000607',
        v_gold_copy_id,
        'Notification Center',
        'core-notifications',
        'System alerts, real-time event feeds, and delivery logs.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "workflow.NotificationCenter", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000608',
        v_gold_copy_id,
        'Notification Templates',
        'core-notification-templates',
        'Configurable email, SMS, and in-app message templates.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "workflow.NotificationTemplates", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000609',
        v_gold_copy_id,
        'Notification Preferences',
        'core-notification-preferences',
        'User and tenant alert subscriptions, channels, and thresholds.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "workflow.NotificationPreferences", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000610',
        v_gold_copy_id,
        'SLA Dashboard',
        'core-sla-dashboard',
        'Service level agreements, breach forecasting, and execution velocity.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "workflow.SLADashboard", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000611',
        v_gold_copy_id,
        'Flow Builder',
        'core-flow-builder',
        'Dataflow graph visual designer and execution engine.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "workflow.FlowBuilder", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000612',
        v_gold_copy_id,
        'Validation Rules',
        'core-validation-rules',
        'Data validation rules, constraint expressions, and error conditions.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "workflow.ValidationRules", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000613',
        v_gold_copy_id,
        'Calculated Fields',
        'core-calculated-fields',
        'Expression engine for computed attributes and formula fields.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "workflow.CalculatedFields", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000614',
        v_gold_copy_id,
        'Run Validations',
        'core-validation',
        'On-demand portfolio and investment validation engine run.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "workflow.InvestmentValidation", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000615',
        v_gold_copy_id,
        'BP Console',
        'bp-console',
        'Execution monitoring, live process instances, and work queues.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "workflow.BPConsole", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000616',
        v_gold_copy_id,
        'Instance Explorer',
        'bp-console-instances',
        'Explore live and archived BP execution instances.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "workflow.BPConsole", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000617',
        v_gold_copy_id,
        'Work Queues',
        'bp-console-queues',
        'Monitor task queues and workload distribution across queues.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "workflow.BPConsole", "inputs": {}}}}'::jsonb,
        '[]'::jsonb, 1, true, 'published', '[]'::jsonb, '[]'::jsonb,
        '{"root": "fb_root", "nodes": {"fb_root": {"id": "fb_root", "type": "Row", "children": ["header"], "style": {"width": "100%", "alignItems": "center"}}}}'::jsonb,
        '{"chrome": "none", "surface": {"padding": 3, "maxWidth": 1600}, "queries": [], "variables": []}'::jsonb,
        false, NOW(), NOW()
    ),
    (
        '018f9d02-0001-7000-8000-000000000618',
        v_gold_copy_id,
        'ChangeSets',
        'governance-changesets',
        'Audit and approve schema, metadata, and configuration changesets.',
        '{"root": "root", "nodes": {"root": {"id": "root", "type": "Column", "children": ["dc"], "style": {"gap": "16px"}}}}'::jsonb,
        '{"dc": {"id": "dc", "type": "DomainComponent", "props": {"component": "workflow.GovernanceChangeSets", "inputs": {}}}}'::jsonb,
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
            (v_tenant.tenant_id, '/core/process-catalog', 'core-process-catalog', NOW()),
            (v_tenant.tenant_id, '/core/approval-inbox', 'core-approval-inbox', NOW()),
            (v_tenant.tenant_id, '/core/approval-workflows', 'core-approval-workflows', NOW()),
            (v_tenant.tenant_id, '/client-portal/workflow-studio', 'client-workflow-studio', NOW()),
            (v_tenant.tenant_id, '/core/workflow-designer', 'core-workflow-designer', NOW()),
            (v_tenant.tenant_id, '/client-portal/rules-editor', 'client-rules-editor', NOW()),
            (v_tenant.tenant_id, '/core/notifications', 'core-notifications', NOW()),
            (v_tenant.tenant_id, '/core/notifications/templates', 'core-notification-templates', NOW()),
            (v_tenant.tenant_id, '/core/notifications/preferences', 'core-notification-preferences', NOW()),
            (v_tenant.tenant_id, '/core/sla-dashboard', 'core-sla-dashboard', NOW()),
            (v_tenant.tenant_id, '/core/flow-builder', 'core-flow-builder', NOW()),
            (v_tenant.tenant_id, '/core/validation-rules', 'core-validation-rules', NOW()),
            (v_tenant.tenant_id, '/core/validation-rules/editor', 'core-validation-rules', NOW()),
            (v_tenant.tenant_id, '/core/calculated-fields', 'core-calculated-fields', NOW()),
            (v_tenant.tenant_id, '/core/validation', 'core-validation', NOW()),
            (v_tenant.tenant_id, '/bp-console', 'bp-console', NOW()),
            (v_tenant.tenant_id, '/bp-console/instances', 'bp-console-instances', NOW()),
            (v_tenant.tenant_id, '/bp-console/queues', 'bp-console-queues', NOW()),
            (v_tenant.tenant_id, '/governance/changesets', 'governance-changesets', NOW())
        ON CONFLICT (tenant_id, path) DO UPDATE
        SET page_key = EXCLUDED.page_key;
    END LOOP;

    -- 3. Repoint Navigation Menu Nodes
    UPDATE navigation_menu_nodes SET target_page_key = 'core-process-catalog', target_route = NULL WHERE target_route = '/core/process-catalog';
    UPDATE navigation_menu_nodes SET target_page_key = 'core-approval-inbox', target_route = NULL WHERE target_route = '/core/approval-inbox';
    UPDATE navigation_menu_nodes SET target_page_key = 'core-approval-workflows', target_route = NULL WHERE target_route = '/core/approval-workflows';
    UPDATE navigation_menu_nodes SET target_page_key = 'client-workflow-studio', target_route = NULL WHERE target_route = '/client-portal/workflow-studio';
    UPDATE navigation_menu_nodes SET target_page_key = 'core-workflow-designer', target_route = NULL WHERE target_route = '/core/workflow-designer';
    UPDATE navigation_menu_nodes SET target_page_key = 'client-rules-editor', target_route = NULL WHERE target_route = '/client-portal/rules-editor';
    UPDATE navigation_menu_nodes SET target_page_key = 'core-notifications', target_route = NULL WHERE target_route = '/core/notifications';
    UPDATE navigation_menu_nodes SET target_page_key = 'core-notification-templates', target_route = NULL WHERE target_route = '/core/notifications/templates';
    UPDATE navigation_menu_nodes SET target_page_key = 'core-notification-preferences', target_route = NULL WHERE target_route = '/core/notifications/preferences';
    UPDATE navigation_menu_nodes SET target_page_key = 'core-sla-dashboard', target_route = NULL WHERE target_route = '/core/sla-dashboard';
    UPDATE navigation_menu_nodes SET target_page_key = 'core-flow-builder', target_route = NULL WHERE target_route = '/core/flow-builder';
    UPDATE navigation_menu_nodes SET target_page_key = 'core-validation-rules', target_route = NULL WHERE target_route = '/core/validation-rules';
    UPDATE navigation_menu_nodes SET target_page_key = 'core-calculated-fields', target_route = NULL WHERE target_route = '/core/calculated-fields';
    UPDATE navigation_menu_nodes SET target_page_key = 'core-validation', target_route = NULL WHERE target_route = '/core/validation';
    UPDATE navigation_menu_nodes SET target_page_key = 'bp-console', target_route = NULL WHERE target_route = '/bp-console';
    UPDATE navigation_menu_nodes SET target_page_key = 'bp-console-instances', target_route = NULL WHERE target_route = '/bp-console/instances';
    UPDATE navigation_menu_nodes SET target_page_key = 'bp-console-queues', target_route = NULL WHERE target_route = '/bp-console/queues';
    UPDATE navigation_menu_nodes SET target_page_key = 'governance-changesets', target_route = NULL WHERE target_route = '/governance/changesets';

END $$;
