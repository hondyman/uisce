-- Migration 20261224_021_seed_compliance_hub_page.up.sql
-- Seeds the core Page Studio page definition and navigation menu node for Compliance & Exception Governance

INSERT INTO public.page_definitions (
    id, tenant_id, name, slug, description, layout, tabs, components, data_sources,
    presentation_events, filter_bar, app_model, version, is_core, status, updated_at
) VALUES (
    '018f9d02-0001-7000-8000-000000000500',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'Compliance & Exception Governance',
    'compliance-hub',
    'Unified compliance console: 94 core rules, tenant overrides, pre-trade blotter, post-trade exception surveillance, regulatory change queue, compliance calendar, and limit utilization.',
    $layout${"root":"root","nodes":{"root":{"id":"root","type":"Column","children":["hdr"],"style":{"gap":"16px"}}}}$layout$::jsonb,
    $tabs$[{"id":"library","label":"Core Rule Library (94 Rules)","layout":{"root":"lib_root","nodes":{"lib_root":{"id":"lib_root","type":"Column","children":["lib_widget"],"style":{"gap":"16px"}}}}},{"id":"matrix","label":"Tenant Rule Matrix","layout":{"root":"matrix_root","nodes":{"matrix_root":{"id":"matrix_root","type":"Column","children":["matrix_widget"],"style":{"gap":"16px"}}}}},{"id":"blotter","label":"Pre-Trade Blotter","layout":{"root":"blotter_root","nodes":{"blotter_root":{"id":"blotter_root","type":"Column","children":["blotter_widget"],"style":{"gap":"16px"}}}}},{"id":"surveillance","label":"Surveillance & Exceptions","layout":{"root":"surv_root","nodes":{"surv_root":{"id":"surv_root","type":"Column","children":["surv_widget"],"style":{"gap":"16px"}}}}},{"id":"regulatory","label":"Regulatory Triage","layout":{"root":"reg_root","nodes":{"reg_root":{"id":"reg_root","type":"Column","children":["reg_widget"],"style":{"gap":"16px"}}}}},{"id":"calendar","label":"Compliance Calendar","layout":{"root":"cal_root","nodes":{"cal_root":{"id":"cal_root","type":"Column","children":["cal_widget"],"style":{"gap":"16px"}}}}},{"id":"limits","label":"Limit Utilization","layout":{"root":"limits_root","nodes":{"limits_root":{"id":"limits_root","type":"Column","children":["limits_widget"],"style":{"gap":"16px"}}}}}]]$tabs$::jsonb,
    $comp${"hdr":{"id":"hdr","type":"PageHeader","props":{"icon":"security","title":"Compliance & Exception Governance","subtitle":"Unified console: 94-rule core library, tenant activation matrix, pre-trade blotter, and surveillance finding queues"},"style":{"flex":"1 1 320px"}},"lib_widget":{"id":"lib_widget","type":"DomainComponent","props":{"component":"compliance.RuleLibraryExplorer","inputs":{}}},"matrix_widget":{"id":"matrix_widget","type":"DomainComponent","props":{"component":"compliance.RuleActivationMatrix","inputs":{}}},"blotter_widget":{"id":"blotter_widget","type":"DomainComponent","props":{"component":"compliance.DecisionBlotter","inputs":{}}},"surv_widget":{"id":"surv_widget","type":"DomainComponent","props":{"component":"compliance.SurveillanceQueue","inputs":{}}},"reg_widget":{"id":"reg_widget","type":"DomainComponent","props":{"component":"compliance.RegulatoryQueue","inputs":{}}},"cal_widget":{"id":"cal_widget","type":"DomainComponent","props":{"component":"compliance.Calendar","inputs":{}}},"limits_widget":{"id":"limits_widget","type":"DomainComponent","props":{"component":"compliance.LimitDashboard","inputs":{}}}}$comp$::jsonb,
    $ds$[]$ds$::jsonb,
    $pe$[]$pe$::jsonb,
    $fb${}$fb$::jsonb,
    $app${"chrome":"none","surface":{"maxWidth":1600,"padding":3},"tabVariable":"tab","variables":[{"name":"tab","default":"library","url":true}],"queries":[]}$app$::jsonb,
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

-- Add to Governance navigation menu
INSERT INTO public.navigation_menu_nodes (
    id, tenant_id, parent_id, node_key, label, target_page_key, display_order, required_entitlement
) VALUES (
    'e81a3d02-0001-7000-8000-000000000500',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '4ef1676b-e9aa-4012-9e44-bba9091e962d',
    'compliance-hub',
    'Compliance & Exception Governance',
    'compliance-hub',
    15,
    'BASE_USER'
)
ON CONFLICT (tenant_id, node_key) DO UPDATE SET
    label = EXCLUDED.label,
    target_page_key = EXCLUDED.target_page_key,
    display_order = EXCLUDED.display_order;
