-- 20261224_017_phase2_tranche3_waci_and_sfdr_rules.up.sql
--
-- Post-Trade Phase II Tranche 3: WACI, SFDR Mandatory PAIs, EU Taxonomy & Stress LCR Buffer (Rules 31–36)
-- 1. Parameter rename hardening on Rule 26: worst_permissible_rank (eliminating inverted UI footgun)
-- 2. Seed 6 Phase II Tranche 3 Rules with attestation-grade citations
-- 3. Version-1 Snapshots with Go RFC 8785 Canonical Content Hashes
-- 4. Ruleset Memberships across ESG_AND_SUSTAINABILITY (5 ESG rules) and POST_TRADE_MONITORING (1 LCR rule)

DO $$
DECLARE
    v_gold_tenant UUID;
    v_r1 UUID;
    v_r2 UUID;
    v_r3 UUID;
    v_r4 UUID;
    v_r5 UUID;
    v_r6 UUID;
BEGIN
    SELECT id INTO v_gold_tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_gold_tenant IS NULL THEN
        SELECT '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid INTO v_gold_tenant;
    END IF;

    -- 1. Parameter rename hardening on Rule 26
    ALTER TABLE compliance.compliance_rule_version DISABLE TRIGGER trg_prevent_rule_version_mutation;
    ALTER TABLE compliance.compliance_rule DISABLE TRIGGER trg_enforce_rule_version_snapshot;

    UPDATE compliance.compliance_rule
    SET 
        ast_condition = '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.split_rating_worst_grade_rank"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"worst_permissible_rank"}}'::jsonb,
        parameter_thresholds = '{"worst_permissible_rank": 10}'::jsonb,
        updated_at = now()
    WHERE rule_code = 'POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR';

    UPDATE compliance.compliance_rule_version
    SET 
        resolved_ast = '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.split_rating_worst_grade_rank"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"worst_permissible_rank"}}'::jsonb,
        parameter_thresholds = '{"worst_permissible_rank": 10}'::jsonb,
        content_hash = '737fda5b774a6a6d790b77d20692389c15c90e9f1559288ec4df2b91735e803b'
    WHERE rule_id IN (SELECT id FROM compliance.compliance_rule WHERE rule_code = 'POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR');

    ALTER TABLE compliance.compliance_rule_version ENABLE TRIGGER trg_prevent_rule_version_mutation;
    ALTER TABLE compliance.compliance_rule ENABLE TRIGGER trg_enforce_rule_version_snapshot;

    -- 2. Seed 6 Tranche 3 Rules
    -- Rule 31: POST_TRADE_ESG_WACI_PORTFOLIO_CEILING
    v_r1 := compliance.seed_core_rule(
        'POST_TRADE_ESG_WACI_PORTFOLIO_CEILING',
        'Weighted Average Carbon Intensity (WACI) Portfolio Ceiling (150 tCO2e/M$)',
        'POST',
        'SOFT_WARNING',
        80,
        'TCFD Recommendations on Metrics and Targets §C.2; SFDR Regulation (EU) 2019/2088 Regulatory Technical Standards PAI 2 (Carbon Footprint / WACI); EU Paris-Aligned Benchmarks Regulation (EU) 2020/1818 Art. 9',
        ARRAY['GLOBAL', 'EU', 'US'],
        '{"max_waci_tco2e_per_m_revenue": "150.000000", "min_emissions_data_coverage_pct": "0.750000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.esg_waci_tco2e_per_m_revenue"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_waci_tco2e_per_m_revenue"}}'::jsonb,
        'ACTIVE'
    );

    -- Rule 32: POST_TRADE_ESG_SCOPE_1_2_EMISSIONS_CEILING
    v_r2 := compliance.seed_core_rule(
        'POST_TRADE_ESG_SCOPE_1_2_EMISSIONS_CEILING',
        'Scope 1 and Scope 2 GHG Emissions Intensity Ceiling (100 tCO2e/M$)',
        'POST',
        'SOFT_WARNING',
        80,
        'GHG Protocol Corporate Standard (Scope 1 & 2 Emissions); SFDR Regulation (EU) 2019/2088 PAI 1 (GHG Emissions); ESRS E1 Climate Change §44',
        ARRAY['GLOBAL', 'EU'],
        '{"max_ghg_scope_1_2_intensity": "100.000000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.esg_ghg_scope_1_2_intensity"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_ghg_scope_1_2_intensity"}}'::jsonb,
        'ACTIVE'
    );

    -- Rule 33: POST_TRADE_ESG_BOARD_GENDER_DIVERSITY_FLOOR
    v_r3 := compliance.seed_core_rule(
        'POST_TRADE_ESG_BOARD_GENDER_DIVERSITY_FLOOR',
        'Board Gender Diversity Minimum Female Representation Floor (30%)',
        'POST',
        'SOFT_WARNING',
        75,
        'EU Corporate Sustainability Reporting Directive (CSRD) / ESRS S1-9; SFDR Regulation (EU) 2019/2088 Mandatory PAI 13 (Board Gender Diversity); UK FCA Listing Rules (PS22/3) LR 9.8.6R',
        ARRAY['EU', 'UK', 'GLOBAL'],
        '{"min_board_gender_diversity_pct": "0.300000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.esg_board_gender_diversity_pct"},"operator":"GREATER_THAN_OR_EQUAL","right":{"type":"PARAM","name":"min_board_gender_diversity_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- Rule 34: POST_TRADE_ESG_HAZARDOUS_WASTE_RATIO_CEILING
    v_r4 := compliance.seed_core_rule(
        'POST_TRADE_ESG_HAZARDOUS_WASTE_RATIO_CEILING',
        'Hazardous Waste Ratio to Revenue Exposure Ceiling (5.0 Tonnes/M$)',
        'POST',
        'SOFT_WARNING',
        75,
        'SFDR Regulation (EU) 2019/2088 Mandatory PAI 9 (Hazardous Waste and Radioactive Waste Ratio); Basel Convention on the Control of Transboundary Movements of Hazardous Wastes',
        ARRAY['GLOBAL', 'EU'],
        '{"max_hazardous_waste_ratio": "5.000000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.esg_hazardous_waste_ratio"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_hazardous_waste_ratio"}}'::jsonb,
        'ACTIVE'
    );

    -- Rule 35: POST_TRADE_EU_TAXONOMY_GREEN_REVENUE_FLOOR
    v_r5 := compliance.seed_core_rule(
        'POST_TRADE_EU_TAXONOMY_GREEN_REVENUE_FLOOR',
        'EU Taxonomy Environmentally Sustainable Green Revenue Floor (15%)',
        'POST',
        'SOFT_WARNING',
        80,
        'EU Taxonomy Regulation (EU) 2020/852 Art. 3, Art. 5 (Substantial Contribution to Climate Change Mitigation); SFDR Regulation (EU) 2019/2088 Art. 8(2a) & Art. 9(4a)',
        ARRAY['EU', 'GLOBAL'],
        '{"min_taxonomy_alignment_pct": "0.150000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.eu_taxonomy_alignment_pct"},"operator":"GREATER_THAN_OR_EQUAL","right":{"type":"PARAM","name":"min_taxonomy_alignment_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- Rule 36: POST_TRADE_LIQUIDITY_COVERAGE_RATIO_BUFFER
    v_r6 := compliance.seed_core_rule(
        'POST_TRADE_LIQUIDITY_COVERAGE_RATIO_BUFFER',
        'Stress Liquidity Coverage Ratio (LCR) Buffer Minimum (105%)',
        'POST',
        'HARD_BLOCK',
        90,
        'Basel III Liquidity Coverage Ratio (LCR) Supervisory Standard (BCBS 238); UCITS Liquidity Risk Management Guidelines (ESMA34-49-286) §5; SEC Rule 22e-4',
        ARRAY['GLOBAL', 'EU', 'US'],
        '{"min_lcr_buffer_ratio": "1.050000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.liquidity_coverage_ratio"},"operator":"GREATER_THAN_OR_EQUAL","right":{"type":"PARAM","name":"min_lcr_buffer_ratio"}}'::jsonb,
        'ACTIVE'
    );

    -- Set data_provenance = 'SYNTHETIC_FIXTURE' for Phase II rules
    UPDATE compliance.compliance_rule
    SET data_provenance = 'SYNTHETIC_FIXTURE'
    WHERE id IN (v_r1, v_r2, v_r3, v_r4, v_r5, v_r6);

    -- Insert v1 snapshots with Go RFC 8785 canonical content hashes
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
            WHEN 'POST_TRADE_ESG_WACI_PORTFOLIO_CEILING'         THEN '119293202be8bb966fb703ef32263675ca31dc2fbf07dd0863fda0dbf444bd41'
            WHEN 'POST_TRADE_ESG_SCOPE_1_2_EMISSIONS_CEILING'    THEN '08d4e786e75a6b72b93bb4014c42d615723a27afb2dae4e720c11910bee8074d'
            WHEN 'POST_TRADE_ESG_BOARD_GENDER_DIVERSITY_FLOOR'   THEN '47df2aa09d5679914d63c29d7b9defd9abafeac182098161480d0478bcf8a33d'
            WHEN 'POST_TRADE_ESG_HAZARDOUS_WASTE_RATIO_CEILING'  THEN 'ca9b5a7a91fb733944b537e12bea1d0bb9522148422a3ba47b8cfe8c7303cddd'
            WHEN 'POST_TRADE_EU_TAXONOMY_GREEN_REVENUE_FLOOR'    THEN '9a791699a246c09f6f83e64a36a73546979ee4d989241e1ac5d733e68b9eaf3e'
            WHEN 'POST_TRADE_LIQUIDITY_COVERAGE_RATIO_BUFFER'    THEN 'f292765cfcd9d6883d997856a5ea07ad91243dd3f19ae1a2a89f26148795faee'
            ELSE encode(sha256(('v1|' || r.ast_condition::text || '|' || r.parameter_thresholds::text || '|' || COALESCE(r.citation, ''))::bytea), 'hex')
        END,
        COALESCE(encode(sha256(r.compiled_bytecode), 'hex'), encode(sha256(''::bytea), 'hex')),
        'system_seed_v1',
        now()
    FROM compliance.compliance_rule r
    WHERE r.id IN (v_r1, v_r2, v_r3, v_r4, v_r5, v_r6)
    ON CONFLICT (rule_id, version) DO NOTHING;

    -- Add Memberships in ESG_AND_SUSTAINABILITY (Rules 31–35)
    INSERT INTO compliance.compliance_ruleset_membership (ruleset_code, rule_id)
    SELECT 'ESG_AND_SUSTAINABILITY', r.id
    FROM compliance.compliance_rule r
    WHERE r.tenant_id = v_gold_tenant
      AND r.id IN (v_r1, v_r2, v_r3, v_r4, v_r5)
    ON CONFLICT (ruleset_code, rule_id) DO NOTHING;

    -- Add Membership in POST_TRADE_MONITORING (Rule 36 LCR)
    INSERT INTO compliance.compliance_ruleset_membership (ruleset_code, rule_id)
    SELECT 'POST_TRADE_MONITORING', r.id
    FROM compliance.compliance_rule r
    WHERE r.tenant_id = v_gold_tenant
      AND r.id = v_r6
    ON CONFLICT (ruleset_code, rule_id) DO NOTHING;

END $$;
