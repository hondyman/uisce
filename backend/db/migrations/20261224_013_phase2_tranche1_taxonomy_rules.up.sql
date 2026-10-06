-- 20261224_013_phase2_tranche1_taxonomy_rules.up.sql
--
-- Post-Trade Phase II Tranche 1: Sector, Industry, Country & Classification Rules (Rules 16–22)
-- 1. Data Provenance Column on compliance.compliance_rule ('VENDOR_PROVEN' vs 'SYNTHETIC_FIXTURE')
-- 2. Seed 7 Phase II Tranche 1 Rules with attestation-grade citations
-- 3. Version-1 Snapshots with Go RFC 8785 Canonical Content Hashes
-- 4. Ruleset Membership in POST_TRADE_MONITORING pack

-- 1. Add data_provenance column
ALTER TABLE compliance.compliance_rule
    ADD COLUMN IF NOT EXISTS data_provenance TEXT NOT NULL DEFAULT 'VENDOR_PROVEN'
        CHECK (data_provenance IN ('VENDOR_PROVEN', 'SYNTHETIC_FIXTURE'));

DO $$
DECLARE
    v_gold_tenant UUID;
    v_r1 UUID;
    v_r2 UUID;
    v_r3 UUID;
    v_r4 UUID;
    v_r5 UUID;
    v_r6 UUID;
    v_r7 UUID;
BEGIN
    SELECT id INTO v_gold_tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_gold_tenant IS NULL THEN
        SELECT '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid INTO v_gold_tenant;
    END IF;

    -- Rule 16: POST_TRADE_UNCLASSIFIED_CEILING_5 (Unclassified Securities Exposure Ceiling 5%)
    v_r1 := compliance.seed_core_rule(
        'POST_TRADE_UNCLASSIFIED_CEILING_5',
        'Unclassified Securities Exposure Ceiling (5%)',
        'POST',
        'HARD_BLOCK',
        95,
        'Institutional Risk Policy & ESMA Guidelines on Portfolio Transparency (ESMA34-49-115) §6; SEC Rule 22e-4',
        ARRAY['GLOBAL', 'EU', 'US'],
        '{"max_unclassified_pct": "0.050000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.unclassified_securities_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_unclassified_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- Rule 17: POST_TRADE_SECTOR_CONCENTRATION_25 (GICS/ICB Sector Concentration Limit 25%)
    v_r2 := compliance.seed_core_rule(
        'POST_TRADE_SECTOR_CONCENTRATION_25',
        'GICS / ICB Sector Concentration Limit (25%)',
        'POST',
        'HARD_BLOCK',
        90,
        'Institutional Mandate Concentration Standards; UCITS Directive 2009/65/EC Art. 52(1); SEC Investment Company Act §8(b)(1)',
        ARRAY['GLOBAL', 'EU', 'US'],
        '{"max_sector_pct": "0.250000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.max_sector_exposure_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_sector_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- Rule 18: POST_TRADE_INDUSTRY_GROUP_15 (Industry Group Concentration Limit 15%)
    v_r3 := compliance.seed_core_rule(
        'POST_TRADE_INDUSTRY_GROUP_15',
        'Industry Group Concentration Limit (15%)',
        'POST',
        'HARD_BLOCK',
        85,
        'Institutional Investment Mandate Standard; US Investment Company Act 1940 §8(b)(1); 17 CFR § 270.8b-16',
        ARRAY['US', 'GLOBAL'],
        '{"max_industry_group_pct": "0.150000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.max_industry_group_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_industry_group_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- Rule 19: POST_TRADE_CYCLICAL_SECTOR_35 (Cyclical Sector Aggregate Exposure Limit 35%)
    v_r4 := compliance.seed_core_rule(
        'POST_TRADE_CYCLICAL_SECTOR_35',
        'Cyclical Sector Aggregate Exposure Limit (35%)',
        'POST',
        'SOFT_WARNING',
        80,
        'Institutional Macro Risk Policy; ESMA Guidelines on Liquidity Stress Testing (ESMA34-49-286) §4',
        ARRAY['GLOBAL', 'EU'],
        '{"max_cyclical_sector_pct": "0.350000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.cyclical_sectors_aggregate_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_cyclical_sector_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- Rule 20: POST_TRADE_EMERGING_MARKET_20 (Emerging Market Country Exposure Limit 20%)
    v_r5 := compliance.seed_core_rule(
        'POST_TRADE_EMERGING_MARKET_20',
        'Emerging Market Country Exposure Limit (20%)',
        'POST',
        'HARD_BLOCK',
        85,
        'MSCI Emerging Markets Classification Standard; UCITS Directive 2009/65/EC Art. 50(1)(f); ESMA Guidelines 2014/937',
        ARRAY['GLOBAL', 'EU', 'US'],
        '{"max_emerging_market_pct": "0.200000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.emerging_markets_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_emerging_market_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- Rule 21: POST_TRADE_NON_OECD_EXPOSURE_10 (Non-OECD Country Aggregate Exposure Ceiling 10%)
    v_r6 := compliance.seed_core_rule(
        'POST_TRADE_NON_OECD_EXPOSURE_10',
        'Non-OECD Country Aggregate Exposure Ceiling (10%)',
        'POST',
        'HARD_BLOCK',
        80,
        'OECD Code of Liberalisation of Capital Movements; UCITS Directive 2009/65/EC Art. 50(1)(d)',
        ARRAY['EU', 'GLOBAL'],
        '{"max_non_oecd_pct": "0.100000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.non_oecd_exposure_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_non_oecd_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- Rule 22: POST_TRADE_FRONTIER_MARKET_5 (Frontier Market Country Sub-Ceiling 5%)
    v_r7 := compliance.seed_core_rule(
        'POST_TRADE_FRONTIER_MARKET_5',
        'Frontier Market Country Sub-Ceiling (5%)',
        'POST',
        'HARD_BLOCK',
        80,
        'MSCI Frontier Markets Index Methodology §2; ESMA Liquidity Risk Management Guidelines',
        ARRAY['GLOBAL', 'EU'],
        '{"max_frontier_market_pct": "0.050000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.frontier_markets_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_frontier_market_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- Set data_provenance = 'SYNTHETIC_FIXTURE' for Phase II rules
    UPDATE compliance.compliance_rule
    SET data_provenance = 'SYNTHETIC_FIXTURE'
    WHERE id IN (v_r1, v_r2, v_r3, v_r4, v_r5, v_r6, v_r7);

    -- Insert v1 snapshots for each seeded rule
    INSERT INTO compliance.compliance_rule_version (
        rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
        citation, effective_from, effective_to, content_hash, compiled_bytecode_hash,
        created_by, created_at
    )
    SELECT 
        r.id,
        COALESCE(r.current_version, 1),
        r.tenant_id,
        r.ast_condition,
        r.parameter_thresholds,
        r.citation,
        r.effective_from,
        r.effective_to,
        CASE r.rule_code
            WHEN 'POST_TRADE_UNCLASSIFIED_CEILING_5'  THEN '76fab86d8503e46b06969a7270f53d64a24cf14b9ebe9be17eaaa2dd6e58fd8d'
            WHEN 'POST_TRADE_SECTOR_CONCENTRATION_25' THEN '9b4402b858ece2be84407d47a40b61378abb89fbec310cb384fe1256db1bbf03'
            WHEN 'POST_TRADE_INDUSTRY_GROUP_15'       THEN '1b98ec605db454ab7b317e32107669cd4a0c2811d57f8d01291cd02360567e47'
            WHEN 'POST_TRADE_CYCLICAL_SECTOR_35'      THEN 'c57d61d200dc8a6730c87bf7c4815297928a3217fb60fd56ee37102241f6e6e0'
            WHEN 'POST_TRADE_EMERGING_MARKET_20'      THEN 'fae409df08b3a2a39d937eba94528c87caa47bb789028aedfcde97426eac0c55'
            WHEN 'POST_TRADE_NON_OECD_EXPOSURE_10'    THEN '1bffc4d36b350446463e576f8d62f19dac53ca0f71da40ff31fcd55efba7d1e2'
            WHEN 'POST_TRADE_FRONTIER_MARKET_5'       THEN '48d0e2175848c2fa842eb4ee1324c081207acd35ab7718679066ed6c19e53b55'
            ELSE encode(sha256(('v1|' || r.ast_condition::text || '|' || r.parameter_thresholds::text || '|' || COALESCE(r.citation, ''))::bytea), 'hex')
        END,
        COALESCE(encode(sha256(r.compiled_bytecode), 'hex'), encode(sha256(''::bytea), 'hex')),
        'system_seed_v1',
        now()
    FROM compliance.compliance_rule r
    WHERE r.id IN (v_r1, v_r2, v_r3, v_r4, v_r5, v_r6, v_r7)
    ON CONFLICT (rule_id, version) DO NOTHING;

    -- Seed memberships into POST_TRADE_MONITORING
    INSERT INTO compliance.compliance_ruleset_membership (ruleset_code, rule_id)
    SELECT 'POST_TRADE_MONITORING', r.id
    FROM compliance.compliance_rule r
    WHERE r.tenant_id = v_gold_tenant
      AND r.id IN (v_r1, v_r2, v_r3, v_r4, v_r5, v_r6, v_r7)
    ON CONFLICT (ruleset_code, rule_id) DO NOTHING;
END $$;
