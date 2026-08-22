-- Migration: Seed Semantic Term Relationship Types, Hierarchies, and Differentiators for Gold Copy
-- Purpose: Provide core ontology relationships and differentiator metadata for Semantic Mapper & AI Context Feeding
-- Date: 2026-08-21

BEGIN;

DO $$
DECLARE
    v_gold_tenant_id UUID;
    v_business_term_type_id UUID := '21645d21-de5f-4feb-af99-99273ea75626';

    v_edge_spec_id UUID;
    v_edge_gen_id UUID;
    v_edge_peer_id UUID;
    v_edge_diff_id UUID;
    v_edge_rel_id UUID;

    -- Account Family Term Node IDs
    v_node_acct_code UUID := 'b1000000-0000-0000-0000-000000000001';
    v_node_alloc_acct UUID := 'b1000000-0000-0000-0000-000000000002';
    v_node_cust_acct UUID := 'b1000000-0000-0000-0000-000000000003';
    v_node_gl_acct UUID := 'b1000000-0000-0000-0000-000000000004';
    v_node_client_acct UUID := 'b1000000-0000-0000-0000-000000000005';
    v_node_sleeve_acct UUID := 'b1000000-0000-0000-0000-000000000006';

    -- Symbology Family Term Node IDs
    v_node_isin UUID := 'b2000000-0000-0000-0000-000000000001';
    v_node_cusip UUID := 'b2000000-0000-0000-0000-000000000002';
    v_node_sedol UUID := 'b2000000-0000-0000-0000-000000000003';
    v_node_figi UUID := 'b2000000-0000-0000-0000-000000000004';
    v_node_lei UUID := 'b2000000-0000-0000-0000-000000000005';
    v_node_ticker UUID := 'b2000000-0000-0000-0000-000000000006';

    -- Date Family Term Node IDs
    v_node_trade_dt UUID := 'b3000000-0000-0000-0000-000000000001';
    v_node_settle_dt UUID := 'b3000000-0000-0000-0000-000000000002';
    v_node_exec_dt UUID := 'b3000000-0000-0000-0000-000000000003';
    v_node_val_dt UUID := 'b3000000-0000-0000-0000-000000000004';
    v_node_mat_dt UUID := 'b3000000-0000-0000-0000-000000000005';
BEGIN
    SELECT id INTO v_gold_tenant_id FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_gold_tenant_id IS NULL THEN
        SELECT id INTO v_gold_tenant_id FROM public.tenants ORDER BY created_at LIMIT 1;
    END IF;

    IF v_gold_tenant_id IS NOT NULL THEN
        -- 1. Ensure Standard Relationship Edge Types exist in catalog_edge_type
        -- IS_SPECIALIZATION_OF
        INSERT INTO catalog_edge_type (tenant_id, edge_type_name, description, is_active, created_at, updated_at)
        VALUES (v_gold_tenant_id::text, 'IS_SPECIALIZATION_OF', 'Sub-type or contextual specialization of a parent business term', true, NOW(), NOW())
        ON CONFLICT (tenant_id, edge_type_name) DO UPDATE SET description = EXCLUDED.description, is_active = EXCLUDED.is_active;

        -- IS_GENERALIZATION_OF
        INSERT INTO catalog_edge_type (tenant_id, edge_type_name, description, is_active, created_at, updated_at)
        VALUES (v_gold_tenant_id::text, 'IS_GENERALIZATION_OF', 'Parent or umbrella business term overarching specialized terms', true, NOW(), NOW())
        ON CONFLICT (tenant_id, edge_type_name) DO UPDATE SET description = EXCLUDED.description, is_active = EXCLUDED.is_active;

        -- IS_PEER_IDENTIFIER_OF
        INSERT INTO catalog_edge_type (tenant_id, edge_type_name, description, is_active, created_at, updated_at)
        VALUES (v_gold_tenant_id::text, 'IS_PEER_IDENTIFIER_OF', 'Peer symbology or alternate identifier for the same underlying entity', true, NOW(), NOW())
        ON CONFLICT (tenant_id, edge_type_name) DO UPDATE SET description = EXCLUDED.description, is_active = EXCLUDED.is_active;

        -- DIFFERENTIATED_FROM
        INSERT INTO catalog_edge_type (tenant_id, edge_type_name, description, is_active, created_at, updated_at)
        VALUES (v_gold_tenant_id::text, 'DIFFERENTIATED_FROM', 'Explicit semantic distinction and disambiguation rule between commonly confused terms', true, NOW(), NOW())
        ON CONFLICT (tenant_id, edge_type_name) DO UPDATE SET description = EXCLUDED.description, is_active = EXCLUDED.is_active;

        -- RELATES_TO
        INSERT INTO catalog_edge_type (tenant_id, edge_type_name, description, is_active, created_at, updated_at)
        VALUES (v_gold_tenant_id::text, 'RELATES_TO', 'Associative semantic relationship in the business domain', true, NOW(), NOW())
        ON CONFLICT (tenant_id, edge_type_name) DO UPDATE SET description = EXCLUDED.description, is_active = EXCLUDED.is_active;

        -- Fetch edge type IDs
        SELECT id INTO v_edge_spec_id FROM catalog_edge_type WHERE tenant_id = v_gold_tenant_id::text AND edge_type_name = 'IS_SPECIALIZATION_OF';
        SELECT id INTO v_edge_gen_id FROM catalog_edge_type WHERE tenant_id = v_gold_tenant_id::text AND edge_type_name = 'IS_GENERALIZATION_OF';
        SELECT id INTO v_edge_peer_id FROM catalog_edge_type WHERE tenant_id = v_gold_tenant_id::text AND edge_type_name = 'IS_PEER_IDENTIFIER_OF';
        SELECT id INTO v_edge_diff_id FROM catalog_edge_type WHERE tenant_id = v_gold_tenant_id::text AND edge_type_name = 'DIFFERENTIATED_FROM';
        SELECT id INTO v_edge_rel_id FROM catalog_edge_type WHERE tenant_id = v_gold_tenant_id::text AND edge_type_name = 'RELATES_TO';

        -- -------------------------------------------------------------
        -- 2. SEED CORE BUSINESS TERMS IN GOLD COPY
        -- -------------------------------------------------------------

        -- 2A. Account Family
        INSERT INTO catalog_node (id, tenant_id, node_name, name, qualified_path, node_type_id, description, properties, is_alpha, is_active, created_at, updated_at)
        VALUES
        (
            v_node_acct_code, v_gold_tenant_id, 'Account Code', 'Account Code', 'business_term/Account Code', v_business_term_type_id,
            'Generic account identifier representing a financial or operational account entity.',
            '{"category": "Account & Entity", "data_type": "string", "domain": "Accounting & Finance", "role": "Parent", "is_pii": false, "differentiator_notes": "General umbrella identifier for any account hierarchy level."}'::jsonb,
            true, true, NOW(), NOW()
        ),
        (
            v_node_alloc_acct, v_gold_tenant_id, 'Allocation Account Code', 'Allocation Account Code', 'business_term/Allocation Account Code', v_business_term_type_id,
            'Sub-account identifier used for allocating executed block trades across individual portfolio sleeves or funds.',
            '{"category": "Account & Entity", "data_type": "string", "domain": "Trading & Portfolio Management", "role": "Specialization", "is_pii": false, "differentiator_notes": "Specific to post-trade execution and block allocation; distinct from custodial safekeeping accounts."}'::jsonb,
            true, true, NOW(), NOW()
        ),
        (
            v_node_cust_acct, v_gold_tenant_id, 'Custodial Account Code', 'Custodial Account Code', 'business_term/Custodial Account Code', v_business_term_type_id,
            'Account number assigned by a third-party depository or custodian bank holding client securities and cash in safekeeping.',
            '{"category": "Account & Entity", "data_type": "string", "domain": "Custody & Settlement", "role": "Specialization", "is_pii": false, "differentiator_notes": "Identifies the external bank custodian safekeeping account (e.g. BNY Mellon, State Street, Euroclear), not internal portfolio sleeves."}'::jsonb,
            true, true, NOW(), NOW()
        ),
        (
            v_node_gl_acct, v_gold_tenant_id, 'GL Account Code', 'GL Account Code', 'business_term/GL Account Code', v_business_term_type_id,
            'General Ledger chart-of-accounts number used for financial accounting, balance sheets, and journal vouchers.',
            '{"category": "Account & Entity", "data_type": "string", "domain": "Financial Reporting", "role": "Specialization", "is_pii": false, "differentiator_notes": "Used exclusively for double-entry GL ledger posting and balance sheet reporting."}'::jsonb,
            true, true, NOW(), NOW()
        ),
        (
            v_node_client_acct, v_gold_tenant_id, 'Client Account Number', 'Client Account Number', 'business_term/Client Account Number', v_business_term_type_id,
            'Client-facing identifier for an investor, wealth management client, or institutional relationship.',
            '{"category": "Account & Entity", "data_type": "string", "domain": "Client Servicing", "role": "Specialization", "is_pii": true, "differentiator_notes": "Represents the client master entity account used for client statements and KYC/AML."}'::jsonb,
            true, true, NOW(), NOW()
        ),
        (
            v_node_sleeve_acct, v_gold_tenant_id, 'Sleeve Account Code', 'Sleeve Account Code', 'business_term/Sleeve Account Code', v_business_term_type_id,
            'Sub-account representing a specific asset manager or strategy sleeve within a Unified Managed Account (UMA).',
            '{"category": "Account & Entity", "data_type": "string", "domain": "Wealth Management", "role": "Specialization", "is_pii": false, "differentiator_notes": "Strategy-specific sleeve partition inside a multi-manager UMA portfolio."}'::jsonb,
            true, true, NOW(), NOW()
        )
        ON CONFLICT (id) DO UPDATE SET
            node_name = EXCLUDED.node_name,
            description = EXCLUDED.description,
            properties = EXCLUDED.properties,
            updated_at = NOW();

        -- 2B. Symbology Family (ISIN, CUSIP, SEDOL, FIGI, LEI, Ticker)
        INSERT INTO catalog_node (id, tenant_id, node_name, name, qualified_path, node_type_id, description, properties, is_alpha, is_active, created_at, updated_at)
        VALUES
        (
            v_node_isin, v_gold_tenant_id, 'ISIN', 'ISIN', 'business_term/ISIN', v_business_term_type_id,
            'International Securities Identification Number (ISO 6166), 12-character alphanumeric global instrument identifier.',
            '{"category": "Symbology & Identifiers", "data_type": "string", "domain": "Capital Markets", "standard": "ISO 6166", "format_pattern": "^[A-Z]{2}[A-Z0-9]{9}[0-9]$", "length": 12, "differentiator_notes": "12-character global standard with 2-char country prefix, 9-char NSIN (e.g. CUSIP/SEDOL base), and 1 check digit."}'::jsonb,
            true, true, NOW(), NOW()
        ),
        (
            v_node_cusip, v_gold_tenant_id, 'CUSIP', 'CUSIP', 'business_term/CUSIP', v_business_term_type_id,
            'Committee on Uniform Securities Identification Procedures, 9-character North American security identifier.',
            '{"category": "Symbology & Identifiers", "data_type": "string", "domain": "Capital Markets", "standard": "ANSI X9.6", "format_pattern": "^[0-9A-Z]{9}$", "length": 9, "differentiator_notes": "9-character identifier for US and Canadian securities. Embeds inside US/CA ISINs as the NSIN."}'::jsonb,
            true, true, NOW(), NOW()
        ),
        (
            v_node_sedol, v_gold_tenant_id, 'SEDOL', 'SEDOL', 'business_term/SEDOL', v_business_term_type_id,
            'Stock Exchange Daily Official List, 7-character UK and European security identifier issued by LSEG.',
            '{"category": "Symbology & Identifiers", "data_type": "string", "domain": "Capital Markets", "standard": "LSEG SEDOL", "format_pattern": "^[0-9B-DF-HJ-NP-TV-Z]{6}[0-9]$", "length": 7, "differentiator_notes": "7-character UK/Ireland identifier (6 alphanumeric characters without vowels + 1 check digit)."}'::jsonb,
            true, true, NOW(), NOW()
        ),
        (
            v_node_figi, v_gold_tenant_id, 'FIGI', 'FIGI', 'business_term/FIGI', v_business_term_type_id,
            'Financial Instrument Global Identifier (OMG standard / Bloomberg Open Symbology), 12-character persistent identifier.',
            '{"category": "Symbology & Identifiers", "data_type": "string", "domain": "Capital Markets", "standard": "OMG FIGI", "format_pattern": "^BBG[0-9A-Z]{9}$", "length": 12, "differentiator_notes": "12-character open identifier starting with BBG prefix, covering instruments across multiple share classes and venues."}'::jsonb,
            true, true, NOW(), NOW()
        ),
        (
            v_node_lei, v_gold_tenant_id, 'LEI', 'LEI', 'business_term/LEI', v_business_term_type_id,
            'Legal Entity Identifier (ISO 17442), 20-character global legal identifier for financial counterparties and issuers.',
            '{"category": "Entities & Counterparties", "data_type": "string", "domain": "Regulatory Compliance", "standard": "ISO 17442", "format_pattern": "^[0-9A-Z]{18}[0-9]{2}$", "length": 20, "differentiator_notes": "Identifies legal corporate entities (issuers/counterparties), NOT the individual traded securities/instruments."}'::jsonb,
            true, true, NOW(), NOW()
        ),
        (
            v_node_ticker, v_gold_tenant_id, 'Primary Ticker', 'Primary Ticker', 'business_term/Primary Ticker', v_business_term_type_id,
            'Exchange-assigned short trading mnemonic (e.g. AAPL, MSFT, VOD.L).',
            '{"category": "Symbology & Identifiers", "data_type": "string", "domain": "Exchange Trading", "format_pattern": "^[A-Z0-9.]{1,10}$", "differentiator_notes": "Human-readable exchange ticker; non-unique globally without exchange/MIC qualifier."}'::jsonb,
            true, true, NOW(), NOW()
        )
        ON CONFLICT (id) DO UPDATE SET
            node_name = EXCLUDED.node_name,
            description = EXCLUDED.description,
            properties = EXCLUDED.properties,
            updated_at = NOW();

        -- 2C. Date Family
        INSERT INTO catalog_node (id, tenant_id, node_name, name, qualified_path, node_type_id, description, properties, is_alpha, is_active, created_at, updated_at)
        VALUES
        (
            v_node_trade_dt, v_gold_tenant_id, 'Trade Date', 'Trade Date', 'business_term/Trade Date', v_business_term_type_id,
            'Date and time when an order or trade agreement was executed in the market.',
            '{"category": "Dates & Timestamps", "data_type": "date", "domain": "Trading", "role": "Lifecycle", "differentiator_notes": "Transaction execution date (T); establishes pricing and ownership commitment."}'::jsonb,
            true, true, NOW(), NOW()
        ),
        (
            v_node_settle_dt, v_gold_tenant_id, 'Settlement Date', 'Settlement Date', 'business_term/Settlement Date', v_business_term_type_id,
            'Date on which securities and cash are officially exchanged and finalized (e.g. T+1, T+2).',
            '{"category": "Dates & Timestamps", "data_type": "date", "domain": "Clearing & Settlement", "role": "Lifecycle", "differentiator_notes": "Final settlement and cash/security delivery date; typically T+1 in modern US/EU markets."}'::jsonb,
            true, true, NOW(), NOW()
        ),
        (
            v_node_exec_dt, v_gold_tenant_id, 'Execution Date', 'Execution Date', 'business_term/Execution Date', v_business_term_type_id,
            'Exact execution timestamp at which the trade filled on the venue.',
            '{"category": "Dates & Timestamps", "data_type": "timestamp", "domain": "Order Routing", "role": "Lifecycle", "differentiator_notes": "High-precision electronic fill timestamp at exchange matching engine."}'::jsonb,
            true, true, NOW(), NOW()
        ),
        (
            v_node_val_dt, v_gold_tenant_id, 'Value Date', 'Value Date', 'business_term/Value Date', v_business_term_type_id,
            'Date on which financial interest or cash flow calculation begins taking effect.',
            '{"category": "Dates & Timestamps", "data_type": "date", "domain": "Treasury & Cash", "role": "Lifecycle", "differentiator_notes": "Effective date for cash interest accrual and FX value calculations."}'::jsonb,
            true, true, NOW(), NOW()
        ),
        (
            v_node_mat_dt, v_gold_tenant_id, 'Maturity Date', 'Maturity Date', 'business_term/Maturity Date', v_business_term_type_id,
            'Final expiration date on which the principal of a fixed income instrument is repaid.',
            '{"category": "Dates & Timestamps", "data_type": "date", "domain": "Fixed Income", "role": "Lifecycle", "differentiator_notes": "Bond/instrument expiration and principal redemption date."}'::jsonb,
            true, true, NOW(), NOW()
        )
        ON CONFLICT (id) DO UPDATE SET
            node_name = EXCLUDED.node_name,
            description = EXCLUDED.description,
            properties = EXCLUDED.properties,
            updated_at = NOW();

        -- -------------------------------------------------------------
        -- 3. SEED RELATIONSHIP EDGES IN CATALOG_EDGE
        -- -------------------------------------------------------------

        -- 3A. Account Family Specializations
        INSERT INTO catalog_edge (id, tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at)
        VALUES
        (
            'c1000000-0000-0000-0000-000000000001', v_gold_tenant_id, v_node_alloc_acct, v_node_acct_code, 'IS_SPECIALIZATION_OF', v_edge_spec_id,
            '{"relationship": "Specialization", "direction": "Sub-to-Parent", "differentiation": "Allocation Account Code specifically routes post-trade sleeve fills, whereas Account Code is the general account entity."}'::jsonb,
            NOW()
        ),
        (
            'c1000000-0000-0000-0000-000000000002', v_gold_tenant_id, v_node_cust_acct, v_node_acct_code, 'IS_SPECIALIZATION_OF', v_edge_spec_id,
            '{"relationship": "Specialization", "direction": "Sub-to-Parent", "differentiation": "Custodial Account Code identifies external safekeeping custodian banks, whereas Account Code is general."}'::jsonb,
            NOW()
        ),
        (
            'c1000000-0000-0000-0000-000000000003', v_gold_tenant_id, v_node_gl_acct, v_node_acct_code, 'IS_SPECIALIZATION_OF', v_edge_spec_id,
            '{"relationship": "Specialization", "direction": "Sub-to-Parent", "differentiation": "GL Account Code is strictly for double-entry financial ledger accounting."}'::jsonb,
            NOW()
        ),
        (
            'c1000000-0000-0000-0000-000000000004', v_gold_tenant_id, v_node_client_acct, v_node_acct_code, 'IS_SPECIALIZATION_OF', v_edge_spec_id,
            '{"relationship": "Specialization", "direction": "Sub-to-Parent", "differentiation": "Client Account Number represents the investor legal customer account entity."}'::jsonb,
            NOW()
        ),
        (
            'c1000000-0000-0000-0000-000000000005', v_gold_tenant_id, v_node_sleeve_acct, v_node_alloc_acct, 'IS_SPECIALIZATION_OF', v_edge_spec_id,
            '{"relationship": "Specialization", "direction": "Sub-to-Parent", "differentiation": "Sleeve Account Code is a sub-account within a multi-manager UMA allocation model."}'::jsonb,
            NOW()
        ),
        -- Account Differentiator (Allocation vs Custodial)
        (
            'c1000000-0000-0000-0000-000000000006', v_gold_tenant_id, v_node_alloc_acct, v_node_cust_acct, 'DIFFERENTIATED_FROM', v_edge_diff_id,
            '{"relationship": "Differentiation", "key_distinction": "Allocation accounts are internal order/sleeve execution targets; Custodial accounts are external bank depositories where assets physically settle and reside."}'::jsonb,
            NOW()
        ),

        -- 3B. Symbology Family (Peer Identifiers & Differentiators)
        (
            'c2000000-0000-0000-0000-000000000001', v_gold_tenant_id, v_node_isin, v_node_cusip, 'IS_PEER_IDENTIFIER_OF', v_edge_peer_id,
            '{"relationship": "Peer Symbology", "cross_reference": "US/CA CUSIPs embed directly into ISINs with US/CA prefix and checksum digit.", "differentiator": "ISIN is 12-char global; CUSIP is 9-char North American."}'::jsonb,
            NOW()
        ),
        (
            'c2000000-0000-0000-0000-000000000002', v_gold_tenant_id, v_node_isin, v_node_sedol, 'IS_PEER_IDENTIFIER_OF', v_edge_peer_id,
            '{"relationship": "Peer Symbology", "cross_reference": "UK/IE SEDOLs embed into GB/IE ISINs.", "differentiator": "ISIN is 12-char global; SEDOL is 7-char UK/European."}'::jsonb,
            NOW()
        ),
        (
            'c2000000-0000-0000-0000-000000000003', v_gold_tenant_id, v_node_cusip, v_node_sedol, 'IS_PEER_IDENTIFIER_OF', v_edge_peer_id,
            '{"relationship": "Peer Symbology", "differentiator": "CUSIP (9-char North America) vs SEDOL (7-char UK/Ireland)."}'::jsonb,
            NOW()
        ),
        (
            'c2000000-0000-0000-0000-000000000004', v_gold_tenant_id, v_node_figi, v_node_isin, 'IS_PEER_IDENTIFIER_OF', v_edge_peer_id,
            '{"relationship": "Peer Symbology", "differentiator": "FIGI is 12-char persistent Bloomberg Open Symbology (starts with BBG); ISIN is ISO 6166 national-numbering-agency assigned."}'::jsonb,
            NOW()
        ),
        (
            'c2000000-0000-0000-0000-000000000005', v_gold_tenant_id, v_node_lei, v_node_isin, 'DIFFERENTIATED_FROM', v_edge_diff_id,
            '{"relationship": "Differentiation", "key_distinction": "LEI (ISO 17442, 20-char) identifies the legal company / issuer, whereas ISIN/CUSIP/SEDOL identify the specific tradable security issued by that entity."}'::jsonb,
            NOW()
        ),

        -- 3C. Dates Family (Lifecycle & Differentiators)
        (
            'c3000000-0000-0000-0000-000000000001', v_gold_tenant_id, v_node_trade_dt, v_node_settle_dt, 'DIFFERENTIATED_FROM', v_edge_diff_id,
            '{"relationship": "Lifecycle Differentiation", "key_distinction": "Trade Date is market execution agreement date (T); Settlement Date is when cash and securities are delivered (T+1/T+2)."}'::jsonb,
            NOW()
        ),
        (
            'c3000000-0000-0000-0000-000000000002', v_gold_tenant_id, v_node_trade_dt, v_node_exec_dt, 'RELATES_TO', v_edge_rel_id,
            '{"relationship": "Temporal Relation", "key_distinction": "Execution Date includes timestamp precision of order match; Trade Date is the official business accounting day."}'::jsonb,
            NOW()
        )
        ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO UPDATE SET
            properties = EXCLUDED.properties;

    END IF;
END $$;

COMMIT;
