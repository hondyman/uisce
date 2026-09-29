-- Migration 20261118_003_seed_mdm_source_scoring_page.up.sql
-- Seeds the core Page Studio page definition and navigation menu node for MDM Source Scoring & Displacement

INSERT INTO public.page_definitions (
    id, tenant_id, name, slug, description, layout, tabs, components, data_sources,
    presentation_events, filter_bar, app_model, version, is_core, status, updated_at
) VALUES (
    '018f9d01-0001-7000-8000-000000000099',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'Source scoring & displacement',
    'mdm-source-scoring',
    'Vendor quality scoring, substitution rates, override endorsements, value-for-money efficient frontier, and displacement readiness simulation.',
    $layout${"root":"matrix_root","nodes":{"matrix_root":{"id":"matrix_root","type":"Column","children":["matrix_grid"],"style":{"gap":"16px"}}}}$layout$::jsonb,
    $tabs$[{"id":"matrix","label":"Sufficiency Matrix","layout":{"root":"matrix_root","nodes":{"matrix_root":{"id":"matrix_root","type":"Column","children":["matrix_grid"],"style":{"gap":"16px"}}}}},{"id":"frontier","label":"Value for Money","layout":{"root":"frontier_root","nodes":{"frontier_root":{"id":"frontier_root","type":"Column","children":["frontier_grid"],"style":{"gap":"16px"}}}}},{"id":"displacement","label":"Displacement Simulator","layout":{"root":"disp_root","nodes":{"disp_root":{"id":"disp_root","type":"Column","children":["candidate_select","displacement_kpis","residual_gaps_grid"],"style":{"gap":"16px"}}}}},{"id":"tolerances","label":"Tolerance Registry","layout":{"root":"tol_root","nodes":{"tol_root":{"id":"tol_root","type":"Column","children":["tolerances_grid"],"style":{"gap":"16px"}}}}}]$tabs$::jsonb,
    $comp${"hdr":{"id":"hdr","type":"PageHeader","props":{"icon":"analytics","title":"Source Scoring & Vendor Displacement","subtitle":"Evaluate vendor sufficiency, displacement readiness, and cost-efficiency frontier"},"style":{"flex":"1 1 320px"}},"universe_select":{"id":"universe_select","type":"VariableSelect","props":{"variable":"universe_size","label":"Universe Scope","minWidth":160,"options":[{"value":"42000","label":"42,000 Entities (Full)"},{"value":"10000","label":"10,000 Entities (Bake-off Sample)"},{"value":"5000","label":"5,000 Entities (Core G10)"}]},"style":{"flex":"0 0 auto"}},"vendor_filter":{"id":"vendor_filter","type":"VariableSelect","props":{"variable":"vendor_filter","label":"Vendor Focus","emptyLabel":"All Vendors","minWidth":180,"options":[{"value":"BBG","label":"Bloomberg (Preference 1)"},{"value":"RFT","label":"Refinitiv (LSEG)"},{"value":"FDS","label":"FactSet"},{"value":"ICE","label":"ICE Data Services"},{"value":"SPG","label":"S&P Global MI"}]},"style":{"flex":"0 0 auto"}},"sync_mart_btn":{"id":"sync_mart_btn","type":"ActionButton","props":{"label":"Sync StarRocks Hot Mart","icon":"refresh","variant":"outlined","onClick":[{"kind":"runOperation","operation":"mdmScoring.syncMart","successMessage":"StarRocks hot analytical mart synchronized."}]},"style":{"flex":"0 0 auto"}},"run_pipeline_btn":{"id":"run_pipeline_btn","type":"ActionButton","props":{"label":"Run Ingest & Scoring Pipeline","icon":"play","variant":"contained","onClick":[{"kind":"runOperation","operation":"mdmScoring.runPipeline","successMessage":"MDM Multi-Vendor Ingestion & Scoring pipeline launched via Temporal."}]},"style":{"flex":"0 0 auto"}},"pipeline_hud":{"id":"pipeline_hud","type":"AlertBanner","props":{"severity":"info","text":"Tripartite Pipeline: Raw files → Apache Iceberg Parquet Lakehouse → Centralized Validation Engine → Staging DB → MDM Survivorship Mastering → Vendor Quality & Displacement Scoring Mart (StarRocks: mdm_analytics.vendor_substitution_daily)."}},"matrix_grid":{"id":"matrix_grid","type":"DataGrid","props":{"query":"scorecard","rowsPath":"substitution_matrix","rowKey":"attribute_code","emptyText":"No scoring data available for this universe.","columns":[{"id":"attr","header":"Attribute","cell":{"kind":"twoLine","primary":"{{row.attribute_code}}","secondary":"Tier {{row.tier}}"}},{"id":"vendor","header":"Vendor","cell":{"kind":"chip","value":"{{row.vendor_id}}","variant":"outlined"}},{"id":"tier","header":"Tier","cell":{"kind":"chip","value":"{{row.tier}}","colorMap":{"1":"warning","2":"primary","3":"default"}}},{"id":"sufficiency","header":"Substitution Rate","cell":{"kind":"percent","value":"{{row.sufficiency_rate_pct}}"},"align":"right"},{"id":"coverage","header":"Coverage","cell":{"kind":"percent","value":"{{row.coverage_pct}}"},"align":"right"},{"id":"cond_suff","header":"Conditional Sufficiency","cell":{"kind":"percent","value":"{{row.conditional_sufficiency_pct}}"},"align":"right"},{"id":"solo_rate","header":"Solo Rate","cell":{"kind":"percent","value":"{{row.solo_rate_pct}}"},"align":"right"},{"id":"oer","header":"Override Endorsement (OER)","cell":{"kind":"percent","value":"{{row.override_endorsement_rate_pct}}"},"align":"right"}]}},"frontier_grid":{"id":"frontier_grid","type":"DataGrid","props":{"query":"scorecard","rowsPath":"frontier_points","rowKey":"vendor_id","emptyText":"No frontier calculations available.","columns":[{"id":"vendor","header":"Vendor","cell":{"kind":"twoLine","primary":"{{row.vendor_name}}","secondary":"{{row.vendor_id}}"}},{"id":"cost","header":"Annual Spend ($)","cell":{"kind":"number","value":"{{row.annual_cost}}"},"align":"right"},{"id":"quality","header":"Composite Quality Index","cell":{"kind":"number","value":"{{row.quality_index}}"},"align":"right"},{"id":"cost_per_pt","header":"Cost per Quality Point ($)","cell":{"kind":"number","value":"{{row.cost_per_quality_point}}"},"align":"right"},{"id":"on_frontier","header":"Frontier Status","cell":{"kind":"chip","value":"{{row.is_on_frontier}}","colorMap":{"true":"success","false":"error","Optimal":"success","Suboptimal":"error"}}}]}},"candidate_select":{"id":"candidate_select","type":"VariableSelect","props":{"variable":"candidate_vendor","label":"Candidate for Removal","minWidth":220,"options":[{"value":"BBG","label":"Bloomberg ($2,140,000/yr)"},{"value":"RFT","label":"Refinitiv ($1,180,000/yr)"},{"value":"FDS","label":"FactSet ($720,000/yr)"},{"value":"ICE","label":"ICE Data ($540,000/yr)"},{"value":"SPG","label":"S&P Global ($610,000/yr)"}]},"style":{"flex":"0 0 auto"}},"displacement_kpis":{"id":"displacement_kpis","type":"DataGrid","props":{"query":"displacement","rowsPath":"tier_summaries","rowKey":"tier","emptyText":"Select a candidate vendor to simulate removal.","columns":[{"id":"tier","header":"Attribute Tier","cell":{"kind":"chip","value":"{{row.tier}}","colorMap":{"1":"warning","2":"primary","3":"default"}}},{"id":"attrs","header":"Attributes Count","cell":{"kind":"number","value":"{{row.attributes_count}}"},"align":"right"},{"id":"unchanged","header":"Unchanged (Takeover Match)","cell":{"kind":"percent","value":"{{row.unchanged_pct}}"},"align":"right"},{"id":"changed","header":"Changed (Within Tolerance)","cell":{"kind":"percent","value":"{{row.changed_pct}}"},"align":"right"},{"id":"now_null","header":"Now NULL (Sole Source Lost)","cell":{"kind":"percent","value":"{{row.now_null_pct}}"},"align":"right"}]}},"residual_gaps_grid":{"id":"residual_gaps_grid","type":"DataGrid","props":{"query":"displacement","rowsPath":"residual_gaps","rowKey":"attribute_code","emptyText":"No residual gaps detected for this removal scenario.","columns":[{"id":"attr","header":"Attribute Code","cell":{"kind":"text","value":"{{row.attribute_code}}"}},{"id":"tier","header":"Tier","cell":{"kind":"chip","value":"{{row.tier}}","colorMap":{"1":"warning","2":"primary","3":"default"}}},{"id":"solo_rate","header":"Solo Rate","cell":{"kind":"percent","value":"{{row.solo_rate_pct}}"},"align":"right"},{"id":"records_lost","header":"Records Lost Without Replacement","cell":{"kind":"number","value":"{{row.records_lost}}"},"align":"right"}]}},"tolerances_grid":{"id":"tolerances_grid","type":"DataGrid","props":{"query":"tolerances","emptyText":"No tolerances registered.","columns":[{"id":"code","header":"Attribute Code","cell":{"kind":"text","value":"{{row.attribute_code}}"}},{"id":"tier","header":"Tier","cell":{"kind":"chip","value":"{{row.tier}}","colorMap":{"1":"warning","2":"primary","3":"default"}}},{"id":"match_type","header":"Match Rule","cell":{"kind":"text","value":"{{row.match_type}}"}},{"id":"tolerance","header":"Tolerance Value","cell":{"kind":"number","value":"{{row.tolerance_val}}"},"align":"right"},{"id":"weight","header":"Tier Weight","cell":{"kind":"number","value":"{{row.tier_weight}}"},"align":"right"},{"id":"desc","header":"Description","cell":{"kind":"text","value":"{{row.description}}"}}]}}}$comp$::jsonb,
    $ds$[]$ds$::jsonb,
    $pe$[]$pe$::jsonb,
    $fb${"root":"filter_root","nodes":{"filter_root":{"id":"filter_root","type":"Column","children":["header_row","pipeline_hud"],"style":{"gap":"12px"}},"header_row":{"id":"header_row","type":"Row","children":["hdr","universe_select","vendor_filter","sync_mart_btn","run_pipeline_btn"],"style":{"alignItems":"center","gap":"8px"}}}}$fb$::jsonb,
    $app${"chrome":"none","surface":{"maxWidth":1400,"padding":3},"tabVariable":"tab","variables":[{"name":"universe_size","default":"42000","url":true},{"name":"candidate_vendor","default":"BBG","url":true},{"name":"vendor_filter","default":"","url":true},{"name":"tab","default":"matrix","url":true}],"queries":[{"id":"scorecard","operation":"mdmScoring.scorecard","params":{"universe_size":"{{vars.universe_size}}"}},{"id":"tolerances","operation":"mdmScoring.tolerances"},{"id":"displacement","operation":"mdmScoring.displacement","params":{"dropped_vendor_id":"{{vars.candidate_vendor}}"}}]}$app$::jsonb,
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

-- Add to Master Data navigation menu
INSERT INTO public.navigation_menu_nodes (
    id, tenant_id, parent_id, node_key, label, target_page_key, display_order, required_entitlement
) VALUES (
    'e81a3d01-0001-7000-8000-000000000099',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '4ef1676b-e9aa-4012-9e44-bba9091e962d',
    'mdm-source-scoring',
    'Source scoring & displacement',
    'mdm-source-scoring',
    13,
    'BASE_USER'
)
ON CONFLICT (tenant_id, node_key) DO UPDATE SET
    label = EXCLUDED.label,
    target_page_key = EXCLUDED.target_page_key,
    display_order = EXCLUDED.display_order;
