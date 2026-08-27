-- 20260930_002_seed_core_report_definitions.up.sql
-- Seeds the 7 core reports into report_templates (legacy) and report_definitions (new).
-- Uses deterministic UUIDs derived from report key for portability.

BEGIN;

DO $$
DECLARE
    gct UUID;
    gctd UUID;
BEGIN
    -- Resolve gold-copy tenant
    SELECT id INTO gct FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF gct IS NULL THEN
        gct := '00000000-0000-0000-0000-000000000001'::UUID;
    END IF;

    -- Resolve gold-copy tenant datasource
    SELECT datasource_id INTO gctd FROM public.tenant_datasources WHERE tenant_id = gct LIMIT 1;
    IF gctd IS NULL THEN
        RAISE WARNING 'No tenant_datasource found for gold-copy tenant % — report_definitions will use NULL placeholder', gct;
        gctd := NULL;
    END IF;

    -- ========================================================================
    -- 1. rep-core-001 — AUM & Fee Revenue Summary
    -- ========================================================================
    INSERT INTO report_templates (id, tenant_id, template_name, description, category, layout_config, is_active, is_public, version)
    VALUES (
        'f48a510c-1fa3-5054-85ac-10e067480625'::UUID, gct,
        'AUM & Fee Revenue Summary',
        'Institutional and wealth management advisory fee schedule breakdown by asset tier.',
        'Core Financial Reports',
        '{"_schemaVersion":2,"reportTitle":"AUM & Fee Revenue Summary","groupDefinitions":[],"sections":[{"id":"s1","title":"AUM & Fee Revenue","elements":[{"id":"el1","type":"table","columns":[{"label":"Account Type","dimension":"account_type","width":180},{"label":"AUM (USD)","measure":"aum_basis_amount","format":"currency","width":140,"alignment":"right"},{"label":"Fee Schedule","dimension":"fee_schedule_code","width":140},{"label":"Eff. Fee (bps)","measure":"effective_fee_bps","format":"number","width":120,"alignment":"right"},{"label":"YTD Revenue","measure":"ytd_revenue","format":"currency","width":140,"alignment":"right"}],"banding":"row","freezePane":true,"pagination":true}],"config":{"visible":true,"backgroundColor":"#ffffff"}}],"sectionConfig":{"s1":{"visible":true}}}',
        true, false, 1
    )
    ON CONFLICT (id) DO UPDATE SET layout_config = EXCLUDED.layout_config, description = EXCLUDED.description;

    INSERT INTO report_definitions
        (id, tenant_id, tenant_datasource_id, report_key, display_name, description, category,
         report_type, output_formats, definition, semantic_query, version, is_current, is_core, status, created_by)
    VALUES (
        'f48a510c-1fa3-5054-85ac-10e067480625'::UUID, gct, gctd,
        'rep-core-001',
        'AUM & Fee Revenue Summary',
        'Institutional and wealth management advisory fee schedule breakdown by asset tier.',
        'Core Financial Reports',
        'paginated', '["pdf","html","excel"]',
        '{"metadata":{"key":"rep-core-001","displayName":"AUM & Fee Revenue Summary","category":"Core Financial Reports","tags":["AUM","Revenue","Monthly"]},"parameters":[{"name":"date_range","type":"dateRange","label":"Period","required":true,"default":"last_90_days"},{"name":"account_type","type":"multiSelect","label":"Account Type","required":false,"options":[{"value":"institutional","label":"Institutional"},{"value":"retail_wealth","label":"Retail Wealth"},{"value":"sma","label":"SMA"}]},{"name":"currency","type":"select","label":"Currency","required":true,"default":"USD"}],"dataBindings":{"primary":{"cube":"oms.account","measures":["SUM(aum_basis_amount)","SUM(ytd_revenue)"],"dimensions":["fee_schedule_code","account_type"],"filters":[{"dimension":"account_type","operator":"IN","parameter":"account_type"}]}},"layout":{"pageSettings":{"size":"letter","orientation":"landscape","margins":{"top":72,"right":72,"bottom":72,"left":72}},"body":{"sections":[{"id":"s1","type":"table","title":"AUM & Fee Revenue","dataBinding":"primary","columns":[{"label":"Account Type","dimension":"account_type","width":180},{"label":"AUM (USD)","measure":"aum_basis_amount","format":"currency","width":140,"alignment":"right"},{"label":"Fee Schedule","dimension":"fee_schedule_code","width":140},{"label":"Eff. Fee (bps)","measure":"effective_fee_bps","format":"number","width":120,"alignment":"right"},{"label":"YTD Revenue","measure":"ytd_revenue","format":"currency","width":140,"alignment":"right"}],"banding":"row","freezePane":true,"pagination":true}]}}}',
        '{"data_bindings":[{"bo_path":"oms.account/institutional","field_allowlist":["sponsor_id","mandate_type","fee_schedule_code"],"measures":[{"field":"aum_basis_amount","aggregation":"SUM"},{"field":"effective_fee_bps","aggregation":"AVG"}],"dimensions":["fee_schedule_code"],"filters":[]},{"bo_path":"oms.account/retail_wealth","field_allowlist":["tax_id_type","accredited_investor_status","fee_schedule_code"],"measures":[{"field":"aum_basis_amount","aggregation":"SUM"}],"dimensions":["fee_schedule_code"],"filters":[]},{"bo_path":"master.sales_ledger/aum_management_fee","field_allowlist":["aum_basis_amount","effective_fee_bps","billing_period_end"],"measures":[{"field":"aum_basis_amount","aggregation":"SUM"}],"dimensions":["fee_schedule_code"],"filters":[]}]}',
        1, true, true, 'published', NULL
    )
    ON CONFLICT (tenant_id, tenant_datasource_id, report_key, version)
        WHERE is_current = true DO UPDATE SET definition = EXCLUDED.definition, display_name = EXCLUDED.display_name;

    -- ========================================================================
    -- 2. rep-core-002 — Quarterly LP Distribution Matrix
    -- ========================================================================
    INSERT INTO report_templates (id, tenant_id, template_name, description, category, layout_config, is_active, is_public, version)
    VALUES (
        'f7e11ab5-63b8-5a18-807c-df028ed90217'::UUID, gct,
        'Quarterly LP Distribution Matrix',
        'Alternative investment capital call and cash flow waterfall summary across fund tranches.',
        'Core Financial Reports',
        '{"_schemaVersion":2,"reportTitle":"Quarterly LP Distribution Matrix","groupDefinitions":[],"sections":[{"id":"s1","title":"LP Distribution","elements":[{"id":"el1","type":"table","columns":[{"label":"Fund","dimension":"investment_name","width":200},{"label":"Vintage","dimension":"vintage_year","width":80},{"label":"Quart er","dimension":"quarter","width":80},{"label":"Capital Called","measure":"called_capital","format":"currency","width":140,"alignment":"right"},{"label":"LP Distribution","measure":"lp_distribution_amount","format":"currency","width":140,"alignment":"right"},{"label":"DPI","measure":"dpi","format":"percent","width":80,"alignment":"right"},{"label":"RVPI","measure":"rvpi","format":"percent","width":80,"alignment":"right"}],"banding":"row","freezePane":true,"pagination":true}],"config":{"visible":true,"backgroundColor":"#ffffff"}}],"sectionConfig":{"s1":{"visible":true}}}',
        true, false, 1
    )
    ON CONFLICT (id) DO UPDATE SET layout_config = EXCLUDED.layout_config, description = EXCLUDED.description;

    INSERT INTO report_definitions
        (id, tenant_id, tenant_datasource_id, report_key, display_name, description, category,
         report_type, output_formats, definition, semantic_query, version, is_current, is_core, status, created_by)
    VALUES (
        'f7e11ab5-63b8-5a18-807c-df028ed90217'::UUID, gct, gctd,
        'rep-core-002',
        'Quarterly LP Distribution Matrix',
        'Alternative investment capital call and cash flow waterfall summary across fund tranches.',
        'Core Financial Reports',
        'paginated', '["pdf","html","excel"]',
        '{"metadata":{"key":"rep-core-002","displayName":"Quarterly LP Distribution Matrix","category":"Core Financial Reports","tags":["LP","Distribution","Quarterly"]},"parameters":[{"name":"period","type":"dateRange","label":"Period","required":true,"default":"this_quarter"},{"name":"fund_type","type":"multiSelect","label":"Fund Type","required":false,"options":[{"value":"private_equity","label":"Private Equity"},{"value":"venture_capital","label":"Venture Capital"},{"value":"hedge_fund","label":"Hedge Fund"},{"value":"real_estate","label":"Real Estate"}]}],"dataBindings":{"primary":{"cube":"altinv.alternative_investment","measures":["SUM(called_capital)","SUM(lp_distribution_amount)","AVG(dpi)","AVG(rvpi)"],"dimensions":["investment_name","vintage_year","quarter"],"filters":[{"dimension":"fund_type","operator":"IN","parameter":"fund_type"}]}},"layout":{"pageSettings":{"size":"letter","orientation":"landscape","margins":{"top":72,"right":72,"bottom":72,"left":72}},"body":{"sections":[{"id":"s1","type":"table","title":"LP Distribution","dataBinding":"primary","columns":[{"label":"Fund","dimension":"investment_name","width":200},{"label":"Vintage","dimension":"vintage_year","width":80},{"label":"Quarter","dimension":"quarter","width":80},{"label":"Capital Called","measure":"called_capital","format":"currency","width":140,"alignment":"right"},{"label":"LP Distribution","measure":"lp_distribution_amount","format":"currency","width":140,"alignment":"right"},{"label":"DPI","measure":"dpi","format":"percent","width":80,"alignment":"right"},{"label":"RVPI","measure":"rvpi","format":"percent","width":80,"alignment":"right"}],"banding":"row","freezePane":true,"pagination":true}]}}}',
        '{"data_bindings":[{"bo_path":"altinv.alternative_investment/private_equity","field_allowlist":["investment_name","vintage_year","committed_capital","called_capital","unfunded_commitment","dpi","rvpi"],"measures":[{"field":"called_capital","aggregation":"SUM"},{"field":"dpi","aggregation":"AVG"},{"field":"rvpi","aggregation":"AVG"}],"dimensions":["vintage_year"],"filters":[]},{"bo_path":"cash_flow.settlement/lp_distribution","field_allowlist":["amount","settlement_date","due_date","management_fee_portion","investment_portion","return_of_capital"],"measures":[{"field":"amount","aggregation":"SUM"}],"dimensions":["due_date"],"filters":[]}]}',
        1, true, true, 'published', NULL
    )
    ON CONFLICT (tenant_id, tenant_datasource_id, report_key, version)
        WHERE is_current = true DO UPDATE SET definition = EXCLUDED.definition, display_name = EXCLUDED.display_name;

    -- ========================================================================
    -- 3. rep-core-003 — Portfolio Rebalance & TLH Impact
    -- ========================================================================
    INSERT INTO report_templates (id, tenant_id, template_name, description, category, layout_config, is_active, is_public, version)
    VALUES (
        'd832fc37-b0eb-57ee-aa00-dab387cc23e2'::UUID, gct,
        'Portfolio Rebalance & TLH Impact',
        'Automated tax-loss harvesting tracking with pre/post wash sale tracking.',
        'Portfolio Operations',
        '{"_schemaVersion":2,"reportTitle":"Portfolio Rebalance & TLH Impact","groupDefinitions":[],"sections":[{"id":"s1","title":"Rebalance Impact","elements":[{"id":"el1","type":"table","columns":[{"label":"Account","dimension":"account_number","width":150},{"label":"Security","dimension":"ticker","width":100},{"label":"Pre-Qty","measure":"pre_quantity","format":"number","width":100,"alignment":"right"},{"label":"Post-Qty","measure":"post_quantity","format":"number","width":100,"alignment":"right"},{"label":"Qty Change","measure":"quantity_change","format":"number","width":100,"alignment":"right"},{"label":"Realized P&L","measure":"realized_pnl","format":"currency","width":120,"alignment":"right"},{"label":"Tax Alpha","measure":"tax_alpha","format":"currency","width":120,"alignment":"right"}],"banding":"row","freezePane":true,"pagination":true}],"config":{"visible":true,"backgroundColor":"#ffffff"}}],"sectionConfig":{"s1":{"visible":true}}}',
        true, false, 1
    )
    ON CONFLICT (id) DO UPDATE SET layout_config = EXCLUDED.layout_config, description = EXCLUDED.description;

    INSERT INTO report_definitions
        (id, tenant_id, tenant_datasource_id, report_key, display_name, description, category,
         report_type, output_formats, definition, semantic_query, version, is_current, is_core, status, created_by)
    VALUES (
        'd832fc37-b0eb-57ee-aa00-dab387cc23e2'::UUID, gct, gctd,
        'rep-core-003',
        'Portfolio Rebalance & TLH Impact',
        'Automated tax-loss harvesting tracking with pre/post wash sale tracking.',
        'Portfolio Operations',
        'paginated', '["pdf","html","excel"]',
        '{"metadata":{"key":"rep-core-003","displayName":"Portfolio Rebalance & TLH Impact","category":"Portfolio Operations","tags":["Rebalance","Tax-Loss","TLH"]},"parameters":[{"name":"date_range","type":"dateRange","label":"Period","required":true,"default":"last_30_days"},{"name":"account","type":"select","label":"Account","required":false}],"dataBindings":{"primary":{"cube":"oms.position","measures":["SUM(quantity)","SUM(market_value)"],"dimensions":["account_number","ticker","subtype_code"],"filters":[]}},"layout":{"pageSettings":{"size":"letter","orientation":"landscape","margins":{"top":72,"right":72,"bottom":72,"left":72}},"body":{"sections":[{"id":"s1","type":"table","title":"Rebalance Impact","dataBinding":"primary","columns":[{"label":"Account","dimension":"account_number","width":150},{"label":"Security","dimension":"ticker","width":100},{"label":"Pre-Qty","measure":"pre_quantity","format":"number","width":100,"alignment":"right"},{"label":"Post-Qty","measure":"post_quantity","format":"number","width":100,"alignment":"right"},{"label":"Qty Change","measure":"quantity_change","format":"number","width":100,"alignment":"right"},{"label":"Realized P&L","measure":"realized_pnl","format":"currency","width":120,"alignment":"right"},{"label":"Tax Alpha","measure":"tax_alpha","format":"currency","width":120,"alignment":"right"}],"banding":"row","freezePane":true,"pagination":true}]}}}',
        '{"data_bindings":[{"bo_path":"oms.position/settled_long","field_allowlist":["custody_account_id","settled_shares","cost_basis_method"],"measures":[{"field":"settled_shares","aggregation":"SUM"}],"dimensions":["custody_account_id"],"filters":[]},{"bo_path":"oms.trade_order/block_parent","field_allowlist":["allocation_profile_id","total_requested_quantity","average_price"],"measures":[{"field":"total_requested_quantity","aggregation":"SUM"}],"dimensions":["allocation_profile_id"],"filters":[]}]}',
        1, true, true, 'published', NULL
    )
    ON CONFLICT (tenant_id, tenant_datasource_id, report_key, version)
        WHERE is_current = true DO UPDATE SET definition = EXCLUDED.definition, display_name = EXCLUDED.display_name;

    -- ========================================================================
    -- 4. rep-custom-001 — High-Net-Worth Household Allocation
    -- ========================================================================
    INSERT INTO report_templates (id, tenant_id, template_name, description, category, layout_config, is_active, is_public, version)
    VALUES (
        '715b3c96-f441-5c81-98ef-0b04cfe78ad1'::UUID, gct,
        'High-Net-Worth Household Allocation',
        'Custom client wealth report highlighting municipal debt yield vs equity momentum.',
        'Private Wealth Advisory',
        '{"_schemaVersion":2,"reportTitle":"High-Net-Worth Household Allocation","groupDefinitions":[],"sections":[{"id":"s1","title":"Asset Allocation","elements":[{"id":"el1","type":"table","columns":[{"label":"Client","dimension":"customer_name","width":180},{"label":"Asset Class","dimension":"asset_class","width":140},{"label":"Market Value","measure":"market_value","format":"currency","width":140,"alignment":"right"},{"label":"Yield","measure":"coupon_rate","format":"percent","width":100,"alignment":"right"},{"label":"Credit Rating","dimension":"credit_rating_sp","width":100}],"banding":"row","freezePane":true,"pagination":true}],"config":{"visible":true,"backgroundColor":"#ffffff"}}],"sectionConfig":{"s1":{"visible":true}}}',
        true, false, 1
    )
    ON CONFLICT (id) DO UPDATE SET layout_config = EXCLUDED.layout_config, description = EXCLUDED.description;

    INSERT INTO report_definitions
        (id, tenant_id, tenant_datasource_id, report_key, display_name, description, category,
         report_type, output_formats, definition, semantic_query, version, is_current, is_core, status, created_by)
    VALUES (
        '715b3c96-f441-5c81-98ef-0b04cfe78ad1'::UUID, gct, gctd,
        'rep-custom-001',
        'High-Net-Worth Household Allocation',
        'Custom client wealth report highlighting municipal debt yield vs equity momentum.',
        'Private Wealth Advisory',
        'paginated', '["pdf","html","excel"]',
        '{"metadata":{"key":"rep-custom-001","displayName":"High-Net-Worth Household Allocation","category":"Private Wealth Advisory","tags":["HNW","Allocation","Wealth"]},"parameters":[{"name":"household_id","type":"select","label":"Household","required":true},{"name":"date","type":"date","label":"As of Date","required":true,"default":"today"}],"dataBindings":{"primary":{"cube":"oms.account","measures":["SUM(market_value)"],"dimensions":["customer_name","asset_class","coupon_rate","credit_rating_sp"],"filters":[]}},"layout":{"pageSettings":{"size":"letter","orientation":"landscape","margins":{"top":72,"right":72,"bottom":72,"left":72}},"body":{"sections":[{"id":"s1","type":"table","title":"Asset Allocation","dataBinding":"primary","columns":[{"label":"Client","dimension":"customer_name","width":180},{"label":"Asset Class","dimension":"asset_class","width":140},{"label":"Market Value","measure":"market_value","format":"currency","width":140,"alignment":"right"},{"label":"Yield","measure":"coupon_rate","format":"percent","width":100,"alignment":"right"},{"label":"Credit Rating","dimension":"credit_rating_sp","width":100}],"banding":"row","freezePane":true,"pagination":true}]}}}',
        '{"data_bindings":[{"bo_path":"oms.account/retail_wealth","field_allowlist":["tax_id_type","accredited_investor_status"],"measures":[],"dimensions":["account_number"],"filters":[]},{"bo_path":"oms.position/settled_long","field_allowlist":["quantity","market_value"],"measures":[{"field":"market_value","aggregation":"SUM"}],"dimensions":["account_id"],"filters":[]},{"bo_path":"oms.security/sovereign_debt","field_allowlist":["ticker","coupon_rate","credit_rating_sp"],"measures":[],"dimensions":["ticker"],"filters":[]}]}',
        1, true, false, 'published', NULL
    )
    ON CONFLICT (tenant_id, tenant_datasource_id, report_key, version)
        WHERE is_current = true DO UPDATE SET definition = EXCLUDED.definition, display_name = EXCLUDED.display_name;

    -- ========================================================================
    -- 5. rep-core-004 — Multi-Asset Factor Risk Decomposition
    -- ========================================================================
    INSERT INTO report_templates (id, tenant_id, template_name, description, category, layout_config, is_active, is_public, version)
    VALUES (
        'b84e5bfc-d50f-576a-8750-ca26e509e483'::UUID, gct,
        'Multi-Asset Factor Risk Decomposition',
        'Barra-style risk factor exposures across equities, fixed income, and commodities.',
        'Portfolio Operations',
        '{"_schemaVersion":2,"reportTitle":"Multi-Asset Factor Risk Decomposition","groupDefinitions":[],"sections":[{"id":"s1","title":"Factor Exposures","elements":[{"id":"el1","type":"table","columns":[{"label":"Factor","dimension":"factor_name","width":160},{"label":"Instrument Class","dimension":"instrument_class","width":140},{"label":"Exposure","measure":"factor_exposure","format":"number","width":120,"alignment":"right"},{"label":"Contribution","measure":"risk_contribution","format":"percent","width":120,"alignment":"right"}],"banding":"row","freezePane":true,"pagination":true}],"config":{"visible":true,"backgroundColor":"#ffffff"}}],"sectionConfig":{"s1":{"visible":true}}}',
        true, false, 1
    )
    ON CONFLICT (id) DO UPDATE SET layout_config = EXCLUDED.layout_config, description = EXCLUDED.description;

    INSERT INTO report_definitions
        (id, tenant_id, tenant_datasource_id, report_key, display_name, description, category,
         report_type, output_formats, definition, semantic_query, version, is_current, is_core, status, created_by)
    VALUES (
        'b84e5bfc-d50f-576a-8750-ca26e509e483'::UUID, gct, gctd,
        'rep-core-004',
        'Multi-Asset Factor Risk Decomposition',
        'Barra-style risk factor exposures across equities, fixed income, and commodities.',
        'Portfolio Operations',
        'paginated', '["pdf","html","excel"]',
        '{"metadata":{"key":"rep-core-004","displayName":"Multi-Asset Factor Risk Decomposition","category":"Portfolio Operations","tags":["Risk","Factor","Multi-Asset"]},"parameters":[{"name":"date","type":"date","label":"As of Date","required":true,"default":"today"},{"name":"portfolio","type":"select","label":"Portfolio","required":true}],"dataBindings":{"primary":{"cube":"oms.position","measures":["SUM(notional_amount)","SUM(unrealized_pnl)"],"dimensions":["subtype_code","instrument_class","factor_name"],"filters":[]}},"layout":{"pageSettings":{"size":"letter","orientation":"landscape","margins":{"top":72,"right":72,"bottom":72,"left":72}},"body":{"sections":[{"id":"s1","type":"table","title":"Factor Exposures","dataBinding":"primary","columns":[{"label":"Factor","dimension":"factor_name","width":160},{"label":"Instrument Class","dimension":"instrument_class","width":140},{"label":"Exposure","measure":"factor_exposure","format":"number","width":120,"alignment":"right"},{"label":"Contribution","measure":"risk_contribution","format":"percent","width":120,"alignment":"right"}],"banding":"row","freezePane":true,"pagination":true}]}}}',
        '{"data_bindings":[{"bo_path":"oms.position/settled_long","field_allowlist":["quantity","market_value","notional_amount"],"measures":[{"field":"notional_amount","aggregation":"SUM"},{"field":"unrealized_pnl","aggregation":"SUM"}],"dimensions":["subtype_code"],"filters":[]},{"bo_path":"oms.security/equity","field_allowlist":["ticker","isin"],"measures":[],"dimensions":["ticker"],"filters":[]}]}',
        1, true, true, 'published', NULL
    )
    ON CONFLICT (tenant_id, tenant_datasource_id, report_key, version)
        WHERE is_current = true DO UPDATE SET definition = EXCLUDED.definition, display_name = EXCLUDED.display_name;

    -- ========================================================================
    -- 6. rep-core-005 — Bitemporal Change Audit Log
    -- ========================================================================
    INSERT INTO report_templates (id, tenant_id, template_name, description, category, layout_config, is_active, is_public, version)
    VALUES (
        'e72a9cf1-e763-5f14-bb80-ca826c1ef26f'::UUID, gct,
        'Bitemporal Change Audit Log',
        'SEC rule compliance audit log tracking system-time vs valid-time entity changes.',
        'Compliance & Audit',
        '{"_schemaVersion":2,"reportTitle":"Bitemporal Change Audit Log","groupDefinitions":[],"sections":[{"id":"s1","title":"Audit Entries","elements":[{"id":"el1","type":"table","columns":[{"label":"Entity","dimension":"entity_type","width":140},{"label":"Entity ID","dimension":"entity_id","width":200},{"label":"System Time","dimension":"system_time","format":"datetime","width":150},{"label":"Valid From","dimension":"valid_from","format":"datetime","width":150},{"label":"Valid To","dimension":"valid_to","format":"datetime","width":150},{"label":"Changed By","dimension":"changed_by","width":140},{"label":"Action","dimension":"action","width":80}],"banding":"row","freezePane":true,"pagination":true}],"config":{"visible":true,"backgroundColor":"#ffffff"}}],"sectionConfig":{"s1":{"visible":true}}}',
        true, false, 1
    )
    ON CONFLICT (id) DO UPDATE SET layout_config = EXCLUDED.layout_config, description = EXCLUDED.description;

    INSERT INTO report_definitions
        (id, tenant_id, tenant_datasource_id, report_key, display_name, description, category,
         report_type, output_formats, definition, semantic_query, version, is_current, is_core, status, created_by)
    VALUES (
        'e72a9cf1-e763-5f14-bb80-ca826c1ef26f'::UUID, gct, gctd,
        'rep-core-005',
        'Bitemporal Change Audit Log',
        'SEC rule compliance audit log tracking system-time vs valid-time entity changes.',
        'Compliance & Audit',
        'paginated', '["pdf","html","excel"]',
        '{"metadata":{"key":"rep-core-005","displayName":"Bitemporal Change Audit Log","category":"Compliance & Audit","tags":["Audit","Bitemporal","Compliance"]},"parameters":[{"name":"date_range","type":"dateRange","label":"Period","required":true,"default":"last_30_days"},{"name":"entity_type","type":"multiSelect","label":"Entity Type","required":false,"options":[{"value":"oms.account","label":"Account"},{"value":"oms.position","label":"Position"},{"value":"oms.security","label":"Security"},{"value":"oms.trade_order","label":"Trade Order"},{"value":"altinv.alternative_investment","label":"Alt Investment"},{"value":"cash_flow.settlement","label":"Settlement"}]}],"dataBindings":{"primary":{"cube":"oms.account","measures":[],"dimensions":["entity_id","system_time","valid_from","valid_to","changed_by","action"],"filters":[]}},"layout":{"pageSettings":{"size":"letter","orientation":"landscape","margins":{"top":72,"right":72,"bottom":72,"left":72}},"body":{"sections":[{"id":"s1","type":"table","title":"Audit Entries","dataBinding":"primary","columns":[{"label":"Entity","dimension":"entity_type","width":140},{"label":"Entity ID","dimension":"entity_id","width":200},{"label":"System Time","dimension":"system_time","format":"datetime","width":150},{"label":"Valid From","dimension":"valid_from","format":"datetime","width":150},{"label":"Valid To","dimension":"valid_to","format":"datetime","width":150},{"label":"Changed By","dimension":"changed_by","width":140},{"label":"Action","dimension":"action","width":80}],"banding":"row","freezePane":true,"pagination":true}]}}}',
        '{"data_bindings":[{"bo_path":"oms.account","field_allowlist":["created_at","updated_at","valid_from","valid_to"],"measures":[],"dimensions":["account_number","created_at","updated_at","valid_from","valid_to"],"filters":[]},{"bo_path":"oms.position","field_allowlist":["created_at","updated_at","valid_from","valid_to"],"measures":[],"dimensions":["created_at","updated_at","valid_from","valid_to"],"filters":[]}]}',
        1, true, true, 'published', NULL
    )
    ON CONFLICT (tenant_id, tenant_datasource_id, report_key, version)
        WHERE is_current = true DO UPDATE SET definition = EXCLUDED.definition, display_name = EXCLUDED.display_name;

    -- ========================================================================
    -- 7. rep-shared-002 — Executive Liquidity & Cash Flow Forecast
    -- ========================================================================
    INSERT INTO report_templates (id, tenant_id, template_name, description, category, layout_config, is_active, is_public, version)
    VALUES (
        'adda052a-9a49-5cc2-b015-c77ce2ee6f2c'::UUID, gct,
        'Executive Liquidity & Cash Flow Forecast',
        'Rolling 30/60/90 day treasury cash obligations and settlement recon.',
        'Executive & Board',
        '{"_schemaVersion":2,"reportTitle":"Executive Liquidity & Cash Flow Forecast","groupDefinitions":[],"sections":[{"id":"s1","title":"30/60/90 Day Forecast","elements":[{"id":"el1","type":"table","columns":[{"label":"Currency","dimension":"currency","width":100},{"label":"Bucket","dimension":"bucket","width":100},{"label":"Settlement Type","dimension":"subtype_code","width":160},{"label":"Amount","measure":"amount","format":"currency","width":140,"alignment":"right"},{"label":"Due Date","dimension":"due_date","format":"date","width":120}],"banding":"row","freezePane":true,"pagination":true}],"config":{"visible":true,"backgroundColor":"#ffffff"}}],"sectionConfig":{"s1":{"visible":true}}}',
        true, false, 1
    )
    ON CONFLICT (id) DO UPDATE SET layout_config = EXCLUDED.layout_config, description = EXCLUDED.description;

    INSERT INTO report_definitions
        (id, tenant_id, tenant_datasource_id, report_key, display_name, description, category,
         report_type, output_formats, definition, semantic_query, version, is_current, is_core, status, created_by)
    VALUES (
        'adda052a-9a49-5cc2-b015-c77ce2ee6f2c'::UUID, gct, gctd,
        'rep-shared-002',
        'Executive Liquidity & Cash Flow Forecast',
        'Rolling 30/60/90 day treasury cash obligations and settlement recon.',
        'Executive & Board',
        'paginated', '["pdf","html","excel"]',
        '{"metadata":{"key":"rep-shared-002","displayName":"Executive Liquidity & Cash Flow Forecast","category":"Executive & Board","tags":["Liquidity","Treasury","Forecast"]},"parameters":[{"name":"date","type":"date","label":"As of Date","required":true,"default":"today"},{"name":"currency","type":"select","label":"Currency","required":false,"default":"USD"}],"dataBindings":{"primary":{"cube":"cash_flow.settlement","measures":["SUM(amount)"],"dimensions":["currency","bucket","subtype_code","due_date"],"filters":[]}},"layout":{"pageSettings":{"size":"letter","orientation":"landscape","margins":{"top":72,"right":72,"bottom":72,"left":72}},"body":{"sections":[{"id":"s1","type":"table","title":"30/60/90 Day Forecast","dataBinding":"primary","columns":[{"label":"Currency","dimension":"currency","width":100},{"label":"Bucket","dimension":"bucket","width":100},{"label":"Settlement Type","dimension":"subtype_code","width":160},{"label":"Amount","measure":"amount","format":"currency","width":140,"alignment":"right"},{"label":"Due Date","dimension":"due_date","format":"date","width":120}],"banding":"row","freezePane":true,"pagination":true}]}}}',
        '{"data_bindings":[{"bo_path":"cash_flow.settlement/dividend","field_allowlist":["amount","currency","settlement_date","due_date","ex_date","record_date"],"measures":[{"field":"amount","aggregation":"SUM"}],"dimensions":["due_date","currency"],"filters":[]},{"bo_path":"cash_flow.settlement/lp_distribution","field_allowlist":["amount","due_date","return_of_capital","carried_interest_retained"],"measures":[{"field":"amount","aggregation":"SUM"}],"dimensions":["due_date","currency"],"filters":[]},{"bo_path":"cash_flow.settlement/capital_call","field_allowlist":["amount","due_date","management_fee_portion","mandatory_flag"],"measures":[{"field":"amount","aggregation":"SUM"}],"dimensions":["due_date"],"filters":[]},{"bo_path":"oms.account/corporate_treasury","field_allowlist":["wire_limit_daily","base_currency"],"measures":[],"dimensions":["base_currency"],"filters":[]}]}',
        1, true, true, 'published', NULL
    )
    ON CONFLICT (tenant_id, tenant_datasource_id, report_key, version)
        WHERE is_current = true DO UPDATE SET definition = EXCLUDED.definition, display_name = EXCLUDED.display_name;

END $$;

COMMIT;
