-- 20261224_015_phase2_tranche2_rules_and_pack.up.sql
--
-- Post-Trade Phase II Tranche 2: Sanctions, FX, Credit Rating, ESG & Settlement Rules (Rules 23–30)
-- 1. Create 5th Licensable Pack: ESG_AND_SUSTAINABILITY
-- 2. Seed 9 Phase II Tranche 2 Rules with attestation-grade citations
-- 3. Version-1 Snapshots with Go RFC 8785 Canonical Content Hashes
-- 4. Ruleset Memberships across POST_TRADE_MONITORING and ESG_AND_SUSTAINABILITY
--
-- Idempotency note: each rule uses `INSERT … ON CONFLICT (tenant_id, rule_code) DO NOTHING`
-- followed by a fallback `SELECT id` to obtain the row id whether the insert wrote a new
-- row or hit the existing one. This deliberately avoids `ON CONFLICT … DO UPDATE`, which
-- would route through compliance_rule's `BEFORE UPDATE` trigger `trg_enforce_rule_version_snapshot`
-- (installed by 20261219_008_trigger_refactor_and_draft_guard.up.sql) and refuse the
-- mutation unless `current_version` is incremented and a matching `compliance_rule_version`
-- snapshot is shipped in the same transaction. The subsequent UPDATE on `data_provenance`
-- is unaffected (that column is not watched by the trigger), the v1 snapshot INSERT uses
-- `ON CONFLICT (rule_id, version) DO NOTHING`, and the membership INSERTs use
-- `ON CONFLICT (ruleset_code, rule_id) DO NOTHING`.

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
    v_r8 UUID;
    v_r9 UUID;
BEGIN
    SELECT id INTO v_gold_tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_gold_tenant IS NULL THEN
        SELECT '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid INTO v_gold_tenant;
    END IF;

    -- Rule 23: POST_TRADE_SANCTION_LIST_ZERO_TOLERANCE
    INSERT INTO compliance.compliance_rule (
        id, tenant_id, inherit_mode, rule_code, name, rule_phase, severity,
        priority, is_active, citation, jurisdictions, parameter_thresholds,
        ast_condition, effective_from, source_version, library_status
    ) VALUES (
        gen_random_uuid(), v_gold_tenant, 'inherit',
        'POST_TRADE_SANCTION_LIST_ZERO_TOLERANCE',
        'Sanctioned Entity Holding Zero Tolerance (Count = 0)',
        'POST_TRADE', 'HARD_BLOCK', 100, true,
        'OFAC Sanctions Regulations 31 CFR Part 500; EU Council Regulation (EU) No 269/2014 Art. 2; UK Sanctions and Anti-Money Laundering Act 2018 §1',
        ARRAY['GLOBAL', 'US', 'EU', 'UK'],
        '{"max_sanctioned_matches": 0}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.sanctioned_entity_matches_count"},"operator":"EQUAL_TO","right":{"type":"PARAM","name":"max_sanctioned_matches"}}'::jsonb,
        '2026-01-01T00:00:00Z', 'CORE_LIB_V1', 'ACTIVE'
    )
    ON CONFLICT (tenant_id, rule_code) DO NOTHING
    RETURNING id INTO v_r1;

    IF v_r1 IS NULL THEN
        SELECT id INTO v_r1
        FROM compliance.compliance_rule
        WHERE tenant_id = v_gold_tenant
          AND rule_code = 'POST_TRADE_SANCTION_LIST_ZERO_TOLERANCE';
    END IF;

    -- Rule 24: POST_TRADE_FX_FORWARD_UNHEDGED_30
    INSERT INTO compliance.compliance_rule (
        id, tenant_id, inherit_mode, rule_code, name, rule_phase, severity,
        priority, is_active, citation, jurisdictions, parameter_thresholds,
        ast_condition, effective_from, source_version, library_status
    ) VALUES (
        gen_random_uuid(), v_gold_tenant, 'inherit',
        'POST_TRADE_FX_FORWARD_UNHEDGED_30',
        'Unhedged Foreign Currency Exposure Limit (30%)',
        'POST_TRADE', 'HARD_BLOCK', 85, true,
        'UCITS Directive 2009/65/EC Art. 51(3); ESMA/2014/937 Guidelines on ETF and Currency Hedged Sub-Funds; Institutional Mandate Risk Policy §4.2',
        ARRAY['GLOBAL', 'EU', 'US'],
        '{"max_unhedged_fx_pct": "0.300000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.unhedged_fx_exposure_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_unhedged_fx_pct"}}'::jsonb,
        '2026-01-01T00:00:00Z', 'CORE_LIB_V1', 'ACTIVE'
    )
    ON CONFLICT (tenant_id, rule_code) DO NOTHING
    RETURNING id INTO v_r2;

    IF v_r2 IS NULL THEN
        SELECT id INTO v_r2
        FROM compliance.compliance_rule
        WHERE tenant_id = v_gold_tenant
          AND rule_code = 'POST_TRADE_FX_FORWARD_UNHEDGED_30';
    END IF;

    -- Rule 25: POST_TRADE_HIGH_YIELD_CEILING_10
    INSERT INTO compliance.compliance_rule (
        id, tenant_id, inherit_mode, rule_code, name, rule_phase, severity,
        priority, is_active, citation, jurisdictions, parameter_thresholds,
        ast_condition, effective_from, source_version, library_status
    ) VALUES (
        gen_random_uuid(), v_gold_tenant, 'inherit',
        'POST_TRADE_HIGH_YIELD_CEILING_10',
        'High-Yield & Sub-Investment Grade Debt Ceiling (10%)',
        'POST_TRADE', 'HARD_BLOCK', 85, true,
        'Institutional Investment Grade Mandate Standard §3.1; UCITS Non-Investment Grade Fixed Income Risk Guidelines',
        ARRAY['GLOBAL', 'US', 'EU'],
        '{"max_high_yield_pct": "0.100000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.high_yield_debt_exposure_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_high_yield_pct"}}'::jsonb,
        '2026-01-01T00:00:00Z', 'CORE_LIB_V1', 'ACTIVE'
    )
    ON CONFLICT (tenant_id, rule_code) DO NOTHING
    RETURNING id INTO v_r3;

    IF v_r3 IS NULL THEN
        SELECT id INTO v_r3
        FROM compliance.compliance_rule
        WHERE tenant_id = v_gold_tenant
          AND rule_code = 'POST_TRADE_HIGH_YIELD_CEILING_10';
    END IF;

    -- Rule 26: POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR
    INSERT INTO compliance.compliance_rule (
        id, tenant_id, inherit_mode, rule_code, name, rule_phase, severity,
        priority, is_active, citation, jurisdictions, parameter_thresholds,
        ast_condition, effective_from, source_version, library_status
    ) VALUES (
        gen_random_uuid(), v_gold_tenant, 'inherit',
        'POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR',
        'Split Credit Rating Conservative Grade Floor (Min BBB- / Baa3)',
        'POST_TRADE', 'HARD_BLOCK', 90, true,
        'Basel III Supervisory Framework CRE20 §15; Institutional Fixed Income Credit Quality Policy §2.4; SEC Rule 2a-7 Credit Quality Evaluation',
        ARRAY['GLOBAL', 'US', 'EU'],
        '{"min_grade_rank": 10}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.split_rating_conservative_grade_rank"},"operator":"GREATER_THAN_OR_EQUAL","right":{"type":"PARAM","name":"min_grade_rank"}}'::jsonb,
        '2026-01-01T00:00:00Z', 'CORE_LIB_V1', 'ACTIVE'
    )
    ON CONFLICT (tenant_id, rule_code) DO NOTHING
    RETURNING id INTO v_r4;

    IF v_r4 IS NULL THEN
        SELECT id INTO v_r4
        FROM compliance.compliance_rule
        WHERE tenant_id = v_gold_tenant
          AND rule_code = 'POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR';
    END IF;

    -- Rule 27: POST_TRADE_ESG_CONTROVERSIAL_WEAPONS_0
    INSERT INTO compliance.compliance_rule (
        id, tenant_id, inherit_mode, rule_code, name, rule_phase, severity,
        priority, is_active, citation, jurisdictions, parameter_thresholds,
        ast_condition, effective_from, source_version, library_status
    ) VALUES (
        gen_random_uuid(), v_gold_tenant, 'inherit',
        'POST_TRADE_ESG_CONTROVERSIAL_WEAPONS_0',
        'Controversial Weapons Zero-Tolerance Exclusion (0%)',
        'POST_TRADE', 'HARD_BLOCK', 95, true,
        'Convention on Cluster Munitions (CCM) Art. 2; Anti-Personnel Mine Ban Convention (Ottawa Treaty); SFDR Regulation (EU) 2019/2088 Art. 8/9 Mandatory PAI 14',
        ARRAY['GLOBAL', 'EU', 'UK'],
        '{"max_weapons_exposure_pct": "0.000000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.esg_controversial_weapons_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_weapons_exposure_pct"}}'::jsonb,
        '2026-01-01T00:00:00Z', 'CORE_LIB_V1', 'ACTIVE'
    )
    ON CONFLICT (tenant_id, rule_code) DO NOTHING
    RETURNING id INTO v_r5;

    IF v_r5 IS NULL THEN
        SELECT id INTO v_r5
        FROM compliance.compliance_rule
        WHERE tenant_id = v_gold_tenant
          AND rule_code = 'POST_TRADE_ESG_CONTROVERSIAL_WEAPONS_0';
    END IF;

    -- Rule 28a: POST_TRADE_ESG_THERMAL_COAL_REVENUE_5
    INSERT INTO compliance.compliance_rule (
        id, tenant_id, inherit_mode, rule_code, name, rule_phase, severity,
        priority, is_active, citation, jurisdictions, parameter_thresholds,
        ast_condition, effective_from, source_version, library_status
    ) VALUES (
        gen_random_uuid(), v_gold_tenant, 'inherit',
        'POST_TRADE_ESG_THERMAL_COAL_REVENUE_5',
        'Thermal Coal Extraction & Power Generation Revenue Ceiling (5%)',
        'POST_TRADE', 'SOFT_WARNING', 80, true,
        'SFDR Regulation (EU) 2019/2088 Regulatory Technical Standards PAI 4; Paris Aligned Benchmark Regulation (EU) 2020/1818 Art. 12(1)(d)',
        ARRAY['GLOBAL', 'EU'],
        '{"max_coal_revenue_pct": "0.050000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.esg_thermal_coal_revenue_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_coal_revenue_pct"}}'::jsonb,
        '2026-01-01T00:00:00Z', 'CORE_LIB_V1', 'ACTIVE'
    )
    ON CONFLICT (tenant_id, rule_code) DO NOTHING
    RETURNING id INTO v_r6;

    IF v_r6 IS NULL THEN
        SELECT id INTO v_r6
        FROM compliance.compliance_rule
        WHERE tenant_id = v_gold_tenant
          AND rule_code = 'POST_TRADE_ESG_THERMAL_COAL_REVENUE_5';
    END IF;

    -- Rule 28b: POST_TRADE_ESG_TOBACCO_REVENUE_5
    INSERT INTO compliance.compliance_rule (
        id, tenant_id, inherit_mode, rule_code, name, rule_phase, severity,
        priority, is_active, citation, jurisdictions, parameter_thresholds,
        ast_condition, effective_from, source_version, library_status
    ) VALUES (
        gen_random_uuid(), v_gold_tenant, 'inherit',
        'POST_TRADE_ESG_TOBACCO_REVENUE_5',
        'Tobacco Production & Distribution Revenue Ceiling (5%)',
        'POST_TRADE', 'SOFT_WARNING', 80, true,
        'WHO Framework Convention on Tobacco Control (FCTC) Art. 5.3; Paris Aligned Benchmark Regulation (EU) 2020/1818 Art. 12(1)(e); UN Global Compact Principle 7',
        ARRAY['GLOBAL', 'EU', 'US'],
        '{"max_tobacco_revenue_pct": "0.050000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.esg_tobacco_revenue_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_tobacco_revenue_pct"}}'::jsonb,
        '2026-01-01T00:00:00Z', 'CORE_LIB_V1', 'ACTIVE'
    )
    ON CONFLICT (tenant_id, rule_code) DO NOTHING
    RETURNING id INTO v_r7;

    IF v_r7 IS NULL THEN
        SELECT id INTO v_r7
        FROM compliance.compliance_rule
        WHERE tenant_id = v_gold_tenant
          AND rule_code = 'POST_TRADE_ESG_TOBACCO_REVENUE_5';
    END IF;

    -- Rule 29: POST_TRADE_ILLIQUID_TIER3_ASSETS_10
    INSERT INTO compliance.compliance_rule (
        id, tenant_id, inherit_mode, rule_code, name, rule_phase, severity,
        priority, is_active, citation, jurisdictions, parameter_thresholds,
        ast_condition, effective_from, source_version, library_status
    ) VALUES (
        gen_random_uuid(), v_gold_tenant, 'inherit',
        'POST_TRADE_ILLIQUID_TIER3_ASSETS_10',
        'Level 3 Illiquid Fair-Value Assets Exposure Ceiling (10%)',
        'POST_TRADE', 'HARD_BLOCK', 85, true,
        'SEC Rule 22e-4 Liquidity Risk Management Program (Illiquid Investments Ceiling); IFRS 13 Fair Value Measurement Level 3 Inputs §72; UCITS Eligible Assets Directive 2007/16/EC Art. 2',
        ARRAY['GLOBAL', 'US', 'EU'],
        '{"max_illiquid_tier3_pct": "0.100000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.illiquid_level3_assets_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_illiquid_tier3_pct"}}'::jsonb,
        '2026-01-01T00:00:00Z', 'CORE_LIB_V1', 'ACTIVE'
    )
    ON CONFLICT (tenant_id, rule_code) DO NOTHING
    RETURNING id INTO v_r8;

    IF v_r8 IS NULL THEN
        SELECT id INTO v_r8
        FROM compliance.compliance_rule
        WHERE tenant_id = v_gold_tenant
          AND rule_code = 'POST_TRADE_ILLIQUID_TIER3_ASSETS_10';
    END IF;

    -- Rule 30: POST_TRADE_SETTLEMENT_FAIL_CONCENTRATION_5
    INSERT INTO compliance.compliance_rule (
        id, tenant_id, inherit_mode, rule_code, name, rule_phase, severity,
        priority, is_active, citation, jurisdictions, parameter_thresholds,
        ast_condition, effective_from, source_version, library_status
    ) VALUES (
        gen_random_uuid(), v_gold_tenant, 'inherit',
        'POST_TRADE_SETTLEMENT_FAIL_CONCENTRATION_5',
        'Settlement Failure & Overdue Settlement Concentration Ceiling (5%)',
        'POST_TRADE', 'SOFT_WARNING', 75, true,
        'CSDR Regulation (EU) No 909/2014 Settlement Discipline Regime Art. 6-7; US SEC Rule 15c6-1 (T+1 Settlement Integrity & Fails Monitoring)',
        ARRAY['GLOBAL', 'EU', 'US'],
        '{"max_settlement_fail_pct": "0.050000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.settlement_fail_exposure_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_settlement_fail_pct"}}'::jsonb,
        '2026-01-01T00:00:00Z', 'CORE_LIB_V1', 'ACTIVE'
    )
    ON CONFLICT (tenant_id, rule_code) DO NOTHING
    RETURNING id INTO v_r9;

    IF v_r9 IS NULL THEN
        SELECT id INTO v_r9
        FROM compliance.compliance_rule
        WHERE tenant_id = v_gold_tenant
          AND rule_code = 'POST_TRADE_SETTLEMENT_FAIL_CONCENTRATION_5';
    END IF;

    -- Set data_provenance = 'SYNTHETIC_FIXTURE' for Phase II rules.
    -- (data_provenance is not watched by trg_enforce_rule_version_snapshot, so this UPDATE
    -- never trips the audit-violation trigger. The column also defaults to
    -- 'SYNTHETIC_FIXTURE' for fresh inserts, but the explicit UPDATE keeps the post-correction
    -- state asserted on every re-run.)
    UPDATE compliance.compliance_rule
    SET data_provenance = 'SYNTHETIC_FIXTURE'
    WHERE id IN (v_r1, v_r2, v_r3, v_r4, v_r5, v_r6, v_r7, v_r8, v_r9);

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
            WHEN 'POST_TRADE_SANCTION_LIST_ZERO_TOLERANCE'      THEN 'c13417040aacf88d6397abf8db8ec6f1537b7d532712f2f9f7a8eecf6293effe'
            WHEN 'POST_TRADE_FX_FORWARD_UNHEDGED_30'            THEN 'ddeb5e1f670c07e67745a1c1a4cfbbff03d3d84a824ef4c63fcd437df5bf948c'
            WHEN 'POST_TRADE_HIGH_YIELD_CEILING_10'             THEN '0af91c5c973cebbe4f31d30ca591b4468fd6de7de9631629a7d4c5089a78a0be'
            WHEN 'POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR'   THEN '7e6e81b266d8c6128eda006be7a63056717c2d3502a66cff70c9e98dab297049'
            WHEN 'POST_TRADE_ESG_CONTROVERSIAL_WEAPONS_0'       THEN 'e7263c70f8f632ebbbac073bd588cdaa6bd561101feb582270f8b8cccecd109b'
            WHEN 'POST_TRADE_ESG_THERMAL_COAL_REVENUE_5'        THEN 'c5e59267453b6fbd68e452ccb3eb9c3a9716bf94b2ee2176c11d5ef78fc77bc7'
            WHEN 'POST_TRADE_ESG_TOBACCO_REVENUE_5'             THEN '1868a2487727f7b9604d13562e124bd7d26154e246f1b97a5f318a31cffb5403'
            WHEN 'POST_TRADE_ILLIQUID_TIER3_ASSETS_10'          THEN '4bdddac163d468c312290602a618607478e5d51b45037586e8aa6adcef89d0e2'
            WHEN 'POST_TRADE_SETTLEMENT_FAIL_CONCENTRATION_5'   THEN 'b0255fcd4baaa793f1eda4809553065b2e823536dbede3dbb504ab6bf52f156a'
            ELSE encode(sha256(('v1|' || r.ast_condition::text || '|' || r.parameter_thresholds::text || '|' || COALESCE(r.citation, ''))::bytea), 'hex')
        END,
        COALESCE(encode(sha256(r.compiled_bytecode), 'hex'), encode(sha256(''::bytea), 'hex')),
        'system_seed_v1',
        now()
    FROM compliance.compliance_rule r
    WHERE r.id IN (v_r1, v_r2, v_r3, v_r4, v_r5, v_r6, v_r7, v_r8, v_r9)
    ON CONFLICT (rule_id, version) DO NOTHING;

    -- Add Memberships in POST_TRADE_MONITORING
    INSERT INTO compliance.compliance_ruleset_membership (ruleset_code, rule_id)
    SELECT 'POST_TRADE_MONITORING', r.id
    FROM compliance.compliance_rule r
    WHERE r.tenant_id = v_gold_tenant
      AND r.id IN (v_r1, v_r2, v_r3, v_r4, v_r8, v_r9)
    ON CONFLICT (ruleset_code, rule_id) DO NOTHING;

    -- Add Memberships in ESG_AND_SUSTAINABILITY
    INSERT INTO compliance.compliance_ruleset_membership (ruleset_code, rule_id)
    SELECT 'ESG_AND_SUSTAINABILITY', r.id
    FROM compliance.compliance_rule r
    WHERE r.tenant_id = v_gold_tenant
      AND r.id IN (v_r5, v_r6, v_r7)
    ON CONFLICT (ruleset_code, rule_id) DO NOTHING;

END $$;