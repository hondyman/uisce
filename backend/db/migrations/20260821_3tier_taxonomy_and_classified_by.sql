-- Migration: 20260821_3tier_taxonomy_and_classified_by.sql
-- Purpose: Seed strict 3-Tier Taxonomy (L1 Domain -> L2 Category -> L3 Classification) and wire Business Terms with CLASSIFIED_BY edges
-- Date: 2026-08-21

BEGIN;

DO $$
DECLARE
    v_gold_tenant_id UUID;
    v_system_tenant_id UUID := '00000000-0000-0000-0000-000000000000';
    v_tenant_id TEXT;
    
    -- Node Types
    v_type_l1 UUID := '55555555-5555-5555-5555-555555550001';
    v_type_l2 UUID := '55555555-5555-5555-5555-555555550002';
    v_type_l3 UUID := '55555555-5555-5555-5555-555555550003';
    v_type_bterm UUID := '21645d21-de5f-4feb-af99-99273ea75626';

    -- Edge Types
    v_edge_type_classified UUID := '66666666-6666-6666-6666-666666660001';

    -- L1 Domain Node IDs
    v_l1_oms UUID := 'a1000000-0000-0000-0000-000000000001';
    v_l1_portfolio UUID := 'a1000000-0000-0000-0000-000000000002';
    v_l1_securities UUID := 'a1000000-0000-0000-0000-000000000003';
    v_l1_platform UUID := 'a1000000-0000-0000-0000-000000000004';

    -- L2 Category Node IDs
    v_l2_orders_alloc UUID := 'a2000000-0000-0000-0000-000000000001';
    v_l2_fin_protocols UUID := 'a2000000-0000-0000-0000-000000000002';
    v_l2_intermediaries UUID := 'a2000000-0000-0000-0000-000000000003';
    v_l2_account_master UUID := 'a2000000-0000-0000-0000-000000000004';
    v_l2_pos_valuation UUID := 'a2000000-0000-0000-0000-000000000005';
    v_l2_asset_servicing UUID := 'a2000000-0000-0000-0000-000000000006';
    v_l2_security_master UUID := 'a2000000-0000-0000-0000-000000000007';
    v_l2_pricing_analytics UUID := 'a2000000-0000-0000-0000-000000000008';
    v_l2_gov_ops UUID := 'a2000000-0000-0000-0000-000000000009';

    -- L3 Classification Node IDs
    v_l3_order_lifecycle UUID := 'a3000000-0000-0000-0000-000000000001';
    v_l3_trade_allocation UUID := 'a3000000-0000-0000-0000-000000000002';
    v_l3_trade_execution UUID := 'a3000000-0000-0000-0000-000000000003';
    v_l3_fix_session UUID := 'a3000000-0000-0000-0000-000000000004';
    v_l3_fix_payload UUID := 'a3000000-0000-0000-0000-000000000005';
    v_l3_broker_counterparty UUID := 'a3000000-0000-0000-0000-000000000006';
    v_l3_acct_ident UUID := 'a3000000-0000-0000-0000-000000000007';
    v_l3_custodial_safe UUID := 'a3000000-0000-0000-0000-000000000008';
    v_l3_acct_gov UUID := 'a3000000-0000-0000-0000-000000000009';
    v_l3_pos_lot_balances UUID := 'a3000000-0000-0000-0000-000000000010';
    v_l3_monetary_amounts UUID := 'a3000000-0000-0000-0000-000000000011';
    v_l3_corp_action UUID := 'a3000000-0000-0000-0000-000000000012';
    v_l3_symbology UUID := 'a3000000-0000-0000-0000-000000000013';
    v_l3_mkt_currency_ref UUID := 'a3000000-0000-0000-0000-000000000014';
    v_l3_quotes_pricing UUID := 'a3000000-0000-0000-0000-000000000015';
    v_l3_audit_temporal UUID := 'a3000000-0000-0000-0000-000000000016';
    v_l3_exchange_cal UUID := 'a3000000-0000-0000-0000-000000000017';
    v_l3_monitoring_rules UUID := 'a3000000-0000-0000-0000-000000000018';

BEGIN
    SELECT id INTO v_gold_tenant_id FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_gold_tenant_id IS NULL THEN
        SELECT id INTO v_gold_tenant_id FROM public.tenants ORDER BY created_at LIMIT 1;
    END IF;
    v_tenant_id := COALESCE(v_gold_tenant_id, v_system_tenant_id)::text;

    -- 0. Clean up noise: Remove redundant/orphaned 1:1 mirror Classification_L4 nodes and circular edges
    DELETE FROM catalog_edge 
    WHERE edge_type_name IN ('CLASSIFIED_AS_L4', 'IS_L4_CLASSIFICATION_OF')
       OR target_id IN (SELECT id FROM catalog_node WHERE node_type_id = '55555555-5555-5555-5555-555555550004');

    DELETE FROM catalog_node 
    WHERE node_type_id = '55555555-5555-5555-5555-555555550004';

    DELETE FROM catalog_node_type 
    WHERE type_name = 'Classification_L4';

    -- 1. Ensure Classification Node Types exist
    INSERT INTO catalog_node_type (id, tenant_id, type_name, display_name, description, is_active, created_at, updated_at)
    VALUES
        (v_type_l1, v_tenant_id, 'Classification_L1', 'Tier 1 Domain', 'Top-level enterprise business domain', true, NOW(), NOW()),
        (v_type_l2, v_tenant_id, 'Classification_L2', 'Tier 2 Category', 'Sub-domain functional category', true, NOW(), NOW()),
        (v_type_l3, v_tenant_id, 'Classification_L3', 'Tier 3 Classification', 'Target taxonomy classification for business terms', true, NOW(), NOW())
    ON CONFLICT (tenant_id, type_name) DO UPDATE SET is_active = true, display_name = EXCLUDED.display_name;

    -- 2. Ensure CLASSIFIED_BY edge type exists
    INSERT INTO catalog_edge_type (id, tenant_id, edge_type_name, description, is_active, is_directed, config, created_at, updated_at)
    VALUES
        (v_edge_type_classified, v_tenant_id, 'CLASSIFIED_BY', 'Connects a business term leaf directly to its Tier-3 Classification node', true, true, '{"hierarchy_tier": "L3"}'::jsonb, NOW(), NOW())
    ON CONFLICT (tenant_id, edge_type_name) DO UPDATE SET is_active = true;

    -- 3. Seed Tier 1 Domains (parent_id = NULL)
    INSERT INTO catalog_node (id, tenant_id, node_type_id, parent_id, node_name, qualified_path, description, is_active, properties, created_at, updated_at)
    VALUES
        (v_l1_oms, v_tenant_id, v_type_l1, NULL, 'Trading & Execution (OMS/EMS)', 'domain/oms_ems', 'Trading lifecycle, execution, allocations, and protocols', true, '{"tier": 1, "tier_name": "Domain"}'::jsonb, NOW(), NOW()),
        (v_l1_portfolio, v_tenant_id, v_type_l1, NULL, 'Portfolio, Accounting & Custody', 'domain/portfolio_custody', 'Accounts, safekeeping, positions, balances, and corporate actions', true, '{"tier": 1, "tier_name": "Domain"}'::jsonb, NOW(), NOW()),
        (v_l1_securities, v_tenant_id, v_type_l1, NULL, 'Securities & Market Data', 'domain/securities_market_data', 'Financial instruments, symbology, quotes, and market analytics', true, '{"tier": 1, "tier_name": "Domain"}'::jsonb, NOW(), NOW()),
        (v_l1_platform, v_tenant_id, v_type_l1, NULL, 'Platform Ops & Data Lake', 'domain/platform_ops', 'Governance, temporal metadata, calendars, and compliance monitoring', true, '{"tier": 1, "tier_name": "Domain"}'::jsonb, NOW(), NOW())
    ON CONFLICT (tenant_id, node_name) DO UPDATE SET 
        node_type_id = EXCLUDED.node_type_id, 
        parent_id = EXCLUDED.parent_id,
        properties = EXCLUDED.properties;

    -- 4. Seed Tier 2 Categories (parent_id = L1 Domain)
    INSERT INTO catalog_node (id, tenant_id, node_type_id, parent_id, node_name, qualified_path, description, is_active, properties, created_at, updated_at)
    VALUES
        -- Under OMS/EMS
        (v_l2_orders_alloc, v_tenant_id, v_type_l2, v_l1_oms, 'Orders & Allocations', 'category/orders_allocations', 'Order lifecycle, blocks, and post-trade allocations', true, '{"tier": 2, "tier_name": "Category"}'::jsonb, NOW(), NOW()),
        (v_l2_fin_protocols, v_tenant_id, v_type_l2, v_l1_oms, 'Financial Protocols', 'category/financial_protocols', 'FIX sessions, network connectivity, and message specifications', true, '{"tier": 2, "tier_name": "Category"}'::jsonb, NOW(), NOW()),
        (v_l2_intermediaries, v_tenant_id, v_type_l2, v_l1_oms, 'Intermediaries', 'category/intermediaries', 'Executing brokers, clearing counterparties, and venues', true, '{"tier": 2, "tier_name": "Category"}'::jsonb, NOW(), NOW()),
        -- Under Portfolio & Custody
        (v_l2_account_master, v_tenant_id, v_type_l2, v_l1_portfolio, 'Account Master', 'category/account_master', 'Account entities, custodian depositories, and account governance', true, '{"tier": 2, "tier_name": "Category"}'::jsonb, NOW(), NOW()),
        (v_l2_pos_valuation, v_tenant_id, v_type_l2, v_l1_portfolio, 'Positions & Valuation', 'category/positions_valuation', 'Holdings, lots, base cost, and monetary values', true, '{"tier": 2, "tier_name": "Category"}'::jsonb, NOW(), NOW()),
        (v_l2_asset_servicing, v_tenant_id, v_type_l2, v_l1_portfolio, 'Asset Servicing', 'category/asset_servicing', 'Corporate action notifications, entitlements, and lifecycle dates', true, '{"tier": 2, "tier_name": "Category"}'::jsonb, NOW(), NOW()),
        -- Under Securities & Market Data
        (v_l2_security_master, v_tenant_id, v_type_l2, v_l1_securities, 'Security Master', 'category/security_master', 'Instrument symbology and reference currencies', true, '{"tier": 2, "tier_name": "Category"}'::jsonb, NOW(), NOW()),
        (v_l2_pricing_analytics, v_tenant_id, v_type_l2, v_l1_securities, 'Pricing & Analytics', 'category/pricing_analytics', 'Market quotes, valuation curves, and exchange listings', true, '{"tier": 2, "tier_name": "Category"}'::jsonb, NOW(), NOW()),
        -- Under Platform Ops
        (v_l2_gov_ops, v_tenant_id, v_type_l2, v_l1_platform, 'Governance & Ops', 'category/governance_ops', 'Temporal audit logs, holiday calendars, and operational alerts', true, '{"tier": 2, "tier_name": "Category"}'::jsonb, NOW(), NOW())
    ON CONFLICT (tenant_id, node_name) DO UPDATE SET 
        node_type_id = EXCLUDED.node_type_id, 
        parent_id = EXCLUDED.parent_id,
        properties = EXCLUDED.properties;

    -- 5. Seed Tier 3 Classifications (parent_id = L2 Category)
    INSERT INTO catalog_node (id, tenant_id, node_type_id, parent_id, node_name, qualified_path, description, is_active, properties, created_at, updated_at)
    VALUES
        -- OMS
        (v_l3_order_lifecycle, v_tenant_id, v_type_l3, v_l2_orders_alloc, 'Order Lifecycle', 'classification/order_lifecycle', 'Top-level order attributes, side, state, and tracking', true, '{"tier": 3, "tier_name": "Classification", "breadcrumb": "Trading & Execution (OMS/EMS) > Orders & Allocations > Order Lifecycle"}'::jsonb, NOW(), NOW()),
        (v_l3_trade_allocation, v_tenant_id, v_type_l3, v_l2_orders_alloc, 'Trade Allocation', 'classification/trade_allocation', 'Block splitting, child shares, allocation accounts', true, '{"tier": 3, "tier_name": "Classification", "breadcrumb": "Trading & Execution (OMS/EMS) > Orders & Allocations > Trade Allocation"}'::jsonb, NOW(), NOW()),
        (v_l3_trade_execution, v_tenant_id, v_type_l3, v_l2_orders_alloc, 'Trade Execution', 'classification/trade_execution', 'Fills, execution timestamps, venues, and prices', true, '{"tier": 3, "tier_name": "Classification", "breadcrumb": "Trading & Execution (OMS/EMS) > Orders & Allocations > Trade Execution"}'::jsonb, NOW(), NOW()),
        (v_l3_fix_session, v_tenant_id, v_type_l3, v_l2_fin_protocols, 'FIX Session & Connectivity', 'classification/fix_session_connectivity', 'FIX session states, sender/target comp IDs', true, '{"tier": 3, "tier_name": "Classification", "breadcrumb": "Trading & Execution (OMS/EMS) > Financial Protocols > FIX Session & Connectivity"}'::jsonb, NOW(), NOW()),
        (v_l3_fix_payload, v_tenant_id, v_type_l3, v_l2_fin_protocols, 'FIX Message Payload', 'classification/fix_message_payload', 'Protocol message types, raw tags, and defaults', true, '{"tier": 3, "tier_name": "Classification", "breadcrumb": "Trading & Execution (OMS/EMS) > Financial Protocols > FIX Message Payload"}'::jsonb, NOW(), NOW()),
        (v_l3_broker_counterparty, v_tenant_id, v_type_l3, v_l2_intermediaries, 'Broker & Counterparty', 'classification/broker_counterparty', 'Executing brokers, clearing firms, broker status', true, '{"tier": 3, "tier_name": "Classification", "breadcrumb": "Trading & Execution (OMS/EMS) > Intermediaries > Broker & Counterparty"}'::jsonb, NOW(), NOW()),
        -- Portfolio & Custody
        (v_l3_acct_ident, v_tenant_id, v_type_l3, v_l2_account_master, 'Account Identification', 'classification/account_identification', 'Internal account codes, account numbers, names', true, '{"tier": 3, "tier_name": "Classification", "breadcrumb": "Portfolio, Accounting & Custody > Account Master > Account Identification"}'::jsonb, NOW(), NOW()),
        (v_l3_custodial_safe, v_tenant_id, v_type_l3, v_l2_account_master, 'Custodial Safekeeping', 'classification/custodial_safekeeping', 'Depository accounts, custodian bank identifiers', true, '{"tier": 3, "tier_name": "Classification", "breadcrumb": "Portfolio, Accounting & Custody > Account Master > Custodial Safekeeping"}'::jsonb, NOW(), NOW()),
        (v_l3_acct_gov, v_tenant_id, v_type_l3, v_l2_account_master, 'Account Governance', 'classification/account_governance', 'Account types, tax treatments, lifecycle states', true, '{"tier": 3, "tier_name": "Classification", "breadcrumb": "Portfolio, Accounting & Custody > Account Master > Account Governance"}'::jsonb, NOW(), NOW()),
        (v_l3_pos_lot_balances, v_tenant_id, v_type_l3, v_l2_pos_valuation, 'Position & Lot Balances', 'classification/position_lot_balances', 'Shares held, base cost, position instances', true, '{"tier": 3, "tier_name": "Classification", "breadcrumb": "Portfolio, Accounting & Custody > Positions & Valuation > Position & Lot Balances"}'::jsonb, NOW(), NOW()),
        (v_l3_monetary_amounts, v_tenant_id, v_type_l3, v_l2_pos_valuation, 'Monetary Amounts', 'classification/monetary_amounts', 'Financial consideration, fees, currencies', true, '{"tier": 3, "tier_name": "Classification", "breadcrumb": "Portfolio, Accounting & Custody > Positions & Valuation > Monetary Amounts"}'::jsonb, NOW(), NOW()),
        (v_l3_corp_action, v_tenant_id, v_type_l3, v_l2_asset_servicing, 'Corporate Action Event', 'classification/corporate_action_event', 'Ex/record/pay dates, corporate action types', true, '{"tier": 3, "tier_name": "Classification", "breadcrumb": "Portfolio, Accounting & Custody > Asset Servicing > Corporate Action Event"}'::jsonb, NOW(), NOW()),
        -- Securities
        (v_l3_symbology, v_tenant_id, v_type_l3, v_l2_security_master, 'Instrument Symbology', 'classification/instrument_symbology', 'CUSIP, ISIN, SEDOL, FIGI, LEI, Tickers', true, '{"tier": 3, "tier_name": "Classification", "breadcrumb": "Securities & Market Data > Security Master > Instrument Symbology"}'::jsonb, NOW(), NOW()),
        (v_l3_mkt_currency_ref, v_tenant_id, v_type_l3, v_l2_security_master, 'Market & Currency Reference', 'classification/market_currency_reference', 'ISO currency codes, curve categories', true, '{"tier": 3, "tier_name": "Classification", "breadcrumb": "Securities & Market Data > Security Master > Market & Currency Reference"}'::jsonb, NOW(), NOW()),
        (v_l3_quotes_pricing, v_tenant_id, v_type_l3, v_l2_pricing_analytics, 'Market Quotes & Pricing', 'classification/market_quotes_pricing', 'Last price, bid/ask, closing valuations', true, '{"tier": 3, "tier_name": "Classification", "breadcrumb": "Securities & Market Data > Pricing & Analytics > Market Quotes & Pricing"}'::jsonb, NOW(), NOW()),
        -- Platform Ops
        (v_l3_audit_temporal, v_tenant_id, v_type_l3, v_l2_gov_ops, 'Audit & Temporal Metadata', 'classification/audit_temporal_metadata', 'Effective dates, created timestamps, as-of dates', true, '{"tier": 3, "tier_name": "Classification", "breadcrumb": "Platform Ops & Data Lake > Governance & Ops > Audit & Temporal Metadata"}'::jsonb, NOW(), NOW()),
        (v_l3_exchange_cal, v_tenant_id, v_type_l3, v_l2_gov_ops, 'Exchange Calendar', 'classification/exchange_calendar', 'Market holidays, exchange settlement schedules', true, '{"tier": 3, "tier_name": "Classification", "breadcrumb": "Platform Ops & Data Lake > Governance & Ops > Exchange Calendar"}'::jsonb, NOW(), NOW()),
        (v_l3_monitoring_rules, v_tenant_id, v_type_l3, v_l2_gov_ops, 'Monitoring & Rules', 'classification/monitoring_rules', 'Business rules, alert messages, threshold breaks', true, '{"tier": 3, "tier_name": "Classification", "breadcrumb": "Platform Ops & Data Lake > Governance & Ops > Monitoring & Rules"}'::jsonb, NOW(), NOW())
    ON CONFLICT (tenant_id, node_name) DO UPDATE SET 
        node_type_id = EXCLUDED.node_type_id, 
        parent_id = EXCLUDED.parent_id,
        properties = EXCLUDED.properties;

    -- 6. Connect Business Terms to L3 Classification Nodes via CLASSIFIED_BY edges
    -- Account Identification
    INSERT INTO catalog_edge (tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at)
    SELECT v_tenant_id, cn.id, v_l3_acct_ident, 'CLASSIFIED_BY', v_edge_type_classified, '{"tier": "L3"}'::jsonb, NOW(), NOW()
    FROM catalog_node cn
    WHERE (cn.tenant_id = v_tenant_id OR cn.tenant_id = v_system_tenant_id)
      AND cn.node_name IN ('Account Code', 'Account Identifier', 'Account Name')
    ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING;

    -- Custodial Safekeeping
    INSERT INTO catalog_edge (tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at)
    SELECT v_tenant_id, cn.id, v_l3_custodial_safe, 'CLASSIFIED_BY', v_edge_type_classified, '{"tier": "L3"}'::jsonb, NOW(), NOW()
    FROM catalog_node cn
    WHERE (cn.tenant_id = v_tenant_id OR cn.tenant_id = v_system_tenant_id)
      AND cn.node_name IN ('Account Custodian Identifier', 'Custodian Identifier', 'Custodial Account Code')
    ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING;

    -- Account Governance
    INSERT INTO catalog_edge (tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at)
    SELECT v_tenant_id, cn.id, v_l3_acct_gov, 'CLASSIFIED_BY', v_edge_type_classified, '{"tier": "L3"}'::jsonb, NOW(), NOW()
    FROM catalog_node cn
    WHERE (cn.tenant_id = v_tenant_id OR cn.tenant_id = v_system_tenant_id)
      AND cn.node_name IN ('Account Status', 'Account Type Code', 'Account Manager Identifier', 'Action Type')
    ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING;

    -- Trade Allocation
    INSERT INTO catalog_edge (tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at)
    SELECT v_tenant_id, cn.id, v_l3_trade_allocation, 'CLASSIFIED_BY', v_edge_type_classified, '{"tier": "L3"}'::jsonb, NOW(), NOW()
    FROM catalog_node cn
    WHERE (cn.tenant_id = v_tenant_id OR cn.tenant_id = v_system_tenant_id)
      AND cn.node_name IN ('Allocation Account Code', 'Allocation Identifier', 'Allocation Order Identifier', 'Allocation Settlement Date', 'Allocation Status')
    ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING;

    -- Order Lifecycle & Execution
    INSERT INTO catalog_edge (tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at)
    SELECT v_tenant_id, cn.id, v_l3_order_lifecycle, 'CLASSIFIED_BY', v_edge_type_classified, '{"tier": "L3"}'::jsonb, NOW(), NOW()
    FROM catalog_node cn
    WHERE (cn.tenant_id = v_tenant_id OR cn.tenant_id = v_system_tenant_id)
      AND cn.node_name IN ('Order Identifier', 'Direction', 'Engine Side')
    ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING;

    INSERT INTO catalog_edge (tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at)
    SELECT v_tenant_id, cn.id, v_l3_trade_execution, 'CLASSIFIED_BY', v_edge_type_classified, '{"tier": "L3"}'::jsonb, NOW(), NOW()
    FROM catalog_node cn
    WHERE (cn.tenant_id = v_tenant_id OR cn.tenant_id = v_system_tenant_id)
      AND cn.node_name IN ('Execution Identifier', 'Execution Order Identifier', 'Execution Status')
    ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING;

    -- FIX Session & Payload
    INSERT INTO catalog_edge (tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at)
    SELECT v_tenant_id, cn.id, v_l3_fix_session, 'CLASSIFIED_BY', v_edge_type_classified, '{"tier": "L3"}'::jsonb, NOW(), NOW()
    FROM catalog_node cn
    WHERE (cn.tenant_id = v_tenant_id OR cn.tenant_id = v_system_tenant_id)
      AND cn.node_name IN ('Fix Session Identifier', 'Fix Session Sender Comp Identifier', 'Fix Session Target Comp Identifier', 'Fix Session Status', 'Fix Version')
    ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING;

    INSERT INTO catalog_edge (tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at)
    SELECT v_tenant_id, cn.id, v_l3_fix_payload, 'CLASSIFIED_BY', v_edge_type_classified, '{"tier": "L3"}'::jsonb, NOW(), NOW()
    FROM catalog_node cn
    WHERE (cn.tenant_id = v_tenant_id OR cn.tenant_id = v_system_tenant_id)
      AND cn.node_name IN ('Fix Message Identifier', 'Fix Message Msg Type', 'Fix Message Session Identifier', 'Fix Message Default Identifier', 'Fix Message Default Msg Type', 'Fix Message Default Session Identifier')
    ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING;

    -- Broker & Counterparty
    INSERT INTO catalog_edge (tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at)
    SELECT v_tenant_id, cn.id, v_l3_broker_counterparty, 'CLASSIFIED_BY', v_edge_type_classified, '{"tier": "L3"}'::jsonb, NOW(), NOW()
    FROM catalog_node cn
    WHERE (cn.tenant_id = v_tenant_id OR cn.tenant_id = v_system_tenant_id)
      AND cn.node_name IN ('Broker Code', 'Bkr Code', 'Broker Identifier', 'Executing Broker Identifier', 'Brokername', 'Brokerstatus', 'Issuer Identifier')
    ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING;

    -- Position & Lot Balances
    INSERT INTO catalog_edge (tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at)
    SELECT v_tenant_id, cn.id, v_l3_pos_lot_balances, 'CLASSIFIED_BY', v_edge_type_classified, '{"tier": "L3"}'::jsonb, NOW(), NOW()
    FROM catalog_node cn
    WHERE (cn.tenant_id = v_tenant_id OR cn.tenant_id = v_system_tenant_id)
      AND cn.node_name IN ('Position Identifier', 'PortfolioHoldingShares', 'Base Cost')
    ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING;

    -- Monetary Amounts
    INSERT INTO catalog_edge (tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at)
    SELECT v_tenant_id, cn.id, v_l3_monetary_amounts, 'CLASSIFIED_BY', v_edge_type_classified, '{"tier": "L3"}'::jsonb, NOW(), NOW()
    FROM catalog_node cn
    WHERE (cn.tenant_id = v_tenant_id OR cn.tenant_id = v_system_tenant_id)
      AND cn.node_name IN ('Financialamount', 'Asset Crrncy Code', 'Currency Code', 'Currency Identifier', 'Currency Name', 'PricingCurrency', 'Annual Revenue')
    ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING;

    -- Corporate Action Event
    INSERT INTO catalog_edge (tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at)
    SELECT v_tenant_id, cn.id, v_l3_corp_action, 'CLASSIFIED_BY', v_edge_type_classified, '{"tier": "L3"}'::jsonb, NOW(), NOW()
    FROM catalog_node cn
    WHERE (cn.tenant_id = v_tenant_id OR cn.tenant_id = v_system_tenant_id)
      AND cn.node_name IN ('Corporate Action Identifier', 'Corporate Action Legacy Ca Identifier', 'Corporate Action Security Identifier', 'Corporateactiontype', 'Corporateactionstatus', 'Corporateactioneffdt', 'Corporateactionexdate', 'Corporateactionrecorddate', 'Corporateactionpaydate', 'Ex Date')
    ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING;

    -- Instrument Symbology
    INSERT INTO catalog_edge (tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at)
    SELECT v_tenant_id, cn.id, v_l3_symbology, 'CLASSIFIED_BY', v_edge_type_classified, '{"tier": "L3"}'::jsonb, NOW(), NOW()
    FROM catalog_node cn
    WHERE (cn.tenant_id = v_tenant_id OR cn.tenant_id = v_system_tenant_id)
      AND cn.node_name IN ('CUSIP', 'ISIN', 'SEDOL', 'FIGI', 'LEI', 'Security Identifier')
    ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING;

    -- Market Quotes & Pricing
    INSERT INTO catalog_edge (tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at)
    SELECT v_tenant_id, cn.id, v_l3_quotes_pricing, 'CLASSIFIED_BY', v_edge_type_classified, '{"tier": "L3"}'::jsonb, NOW(), NOW()
    FROM catalog_node cn
    WHERE (cn.tenant_id = v_tenant_id OR cn.tenant_id = v_system_tenant_id)
      AND cn.node_name IN ('LastPrice', 'Curve Category', 'Exch Code')
    ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING;

    -- Exchange Calendar
    INSERT INTO catalog_edge (tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at)
    SELECT v_tenant_id, cn.id, v_l3_exchange_cal, 'CLASSIFIED_BY', v_edge_type_classified, '{"tier": "L3"}'::jsonb, NOW(), NOW()
    FROM catalog_node cn
    WHERE (cn.tenant_id = v_tenant_id OR cn.tenant_id = v_system_tenant_id)
      AND cn.node_name IN ('Calendar Date', 'Exchangecalendardate', 'Exchange Calendar Identifier', 'Exchange Calendar Market Identifier', 'Exchange Calendar Holiday Name', 'Holiday Name')
    ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING;

    -- Audit & Temporal Metadata
    INSERT INTO catalog_edge (tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at)
    SELECT v_tenant_id, cn.id, v_l3_audit_temporal, 'CLASSIFIED_BY', v_edge_type_classified, '{"tier": "L3"}'::jsonb, NOW(), NOW()
    FROM catalog_node cn
    WHERE (cn.tenant_id = v_tenant_id OR cn.tenant_id = v_system_tenant_id)
      AND cn.node_name IN ('As Of Date', 'Effective Date', 'Created At', 'Audit Timestamp', 'Custom Attributes', 'Category', 'Code', 'Country Code')
    ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING;

    -- Monitoring & Rules
    INSERT INTO catalog_edge (tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at)
    SELECT v_tenant_id, cn.id, v_l3_monitoring_rules, 'CLASSIFIED_BY', v_edge_type_classified, '{"tier": "L3"}'::jsonb, NOW(), NOW()
    FROM catalog_node cn
    WHERE (cn.tenant_id = v_tenant_id OR cn.tenant_id = v_system_tenant_id)
      AND cn.node_name IN ('Alert Message', 'Fix Alert Identifier', 'Fix Alert Message Identifier', 'Rule Identifier')
    ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING;

END $$;

COMMIT;
