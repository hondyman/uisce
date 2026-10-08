-- 20261223_012_phase1_tranche2_rules_and_pack.up.sql
--
-- Post-Trade Phase 1 Tranche 2 (Sovereign, Agency, Muni, CCP, Custodian, Bank Deposit, Sec Lending)
-- 1. Seed 7 Phase 1 Tranche 2 Post-Trade Rules into Gold-Copy Master
-- 2. Seed v1 Snapshots with RFC 8785 Canonical Content Hashes
-- 3. Seed Fourth Licensable Ruleset Pack: POST_TRADE_MONITORING (14 total post-trade rules)

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

    -- 1. POST_TRADE_SOVEREIGN_EXPOSURE_35 (Sovereign Debt Single-Country Concentration Limit 35%)
    v_r1 := compliance.seed_core_rule(
        'POST_TRADE_SOVEREIGN_EXPOSURE_35',
        'Sovereign Debt Single-Country Concentration Limit (35%)',
        'POST',
        'HARD_BLOCK',
        90,
        'UCITS Directive 2009/65/EC Art. 54(1); SEC Rule 2a-7(d)(2); ESMA Guidelines ESMA/2014/937 §43',
        ARRAY['EU', 'US'],
        '{"max_sovereign_exposure_pct": "0.350000", "require_oecd_member": false}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.max_sovereign_exposure_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_sovereign_exposure_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- 2. POST_TRADE_AGENCY_SUPRA_25 (Agency and Supranational Debt Concentration Limit 25%)
    v_r2 := compliance.seed_core_rule(
        'POST_TRADE_AGENCY_SUPRA_25',
        'Agency and Supranational Debt Concentration Limit (25%)',
        'POST',
        'HARD_BLOCK',
        85,
        'ESMA Guidelines on Money Market Funds (ESMA34-49-115) §4.2; UCITS Directive 2009/65/EC Art. 52(2)',
        ARRAY['EU', 'GLOBAL'],
        '{"include_multilateral_dev_banks": true, "max_agency_supra_pct": "0.250000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.max_agency_supra_exposure_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_agency_supra_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- 3. POST_TRADE_MUNI_OBLIGOR_10 (Municipal Bond Single-Obligor Concentration Limit 10%)
    v_r3 := compliance.seed_core_rule(
        'POST_TRADE_MUNI_OBLIGOR_10',
        'Municipal Bond Single-Obligor Concentration Limit (10%)',
        'POST',
        'HARD_BLOCK',
        85,
        '17 CFR § 270.2a-7(d)(3)(i); MSRB Rules G-17 & G-19; US Investment Company Act 1940 §8(b)(1)',
        ARRAY['US'],
        '{"max_muni_obligor_pct": "0.100000", "revenue_bond_lookthrough": true}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.max_muni_obligor_exposure_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_muni_obligor_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- 4. POST_TRADE_CCP_CLEARING_EXPOSURE_15 (Central Counterparty Clearing Exposure Limit 15%)
    v_r4 := compliance.seed_core_rule(
        'POST_TRADE_CCP_CLEARING_EXPOSURE_15',
        'Central Counterparty (CCP) Clearing Exposure Limit (15%)',
        'POST',
        'HARD_BLOCK',
        80,
        'EMIR Regulation (EU) No 648/2012 Art. 47; CSDR Regulation (EU) No 909/2014 Art. 12; CFTC 17 CFR Part 39',
        ARRAY['EU', 'US', 'GLOBAL'],
        '{"max_ccp_exposure_pct": "0.150000", "qualifying_ccp_only": true}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.max_ccp_exposure_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_ccp_exposure_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- 5. POST_TRADE_CUSTODIAN_CONCENTRATION_20 (Custodian Safekeeping Asset Concentration Limit 20%)
    v_r5 := compliance.seed_core_rule(
        'POST_TRADE_CUSTODIAN_CONCENTRATION_20',
        'Custodian Safekeeping Asset Concentration Limit (20%)',
        'POST',
        'HARD_BLOCK',
        80,
        'AIFMD Directive 2011/61/EU Art. 21(8); UCITS Directive 2009/65/EC Art. 22; SEC Custody Rule 17 CFR § 275.206(4)-2',
        ARRAY['EU', 'US'],
        '{"max_custodian_pct": "0.200000", "segregated_sub_custody_exemption": false}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.max_custodian_concentration_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_custodian_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- 6. POST_TRADE_BANK_DEPOSIT_20 (Single-Bank Cash Deposit Limit 20%)
    v_r6 := compliance.seed_core_rule(
        'POST_TRADE_BANK_DEPOSIT_20',
        'General Cash & Bank Deposit Concentration House Limit (20%)',
        'POST',
        'HARD_BLOCK',
        80,
        'Institutional Cash Management Policy & FCA COLL 5.2.10R (House limit applying to aggregate operational and sweep bank cash; cross-ref statutory UCITS_DEPOSIT_20)',
        ARRAY['EU', 'UK'],
        '{"credit_institution_tier1_only": true, "max_bank_deposit_pct": "0.200000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.max_bank_deposit_pct"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_bank_deposit_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- 7. POST_TRADE_SEC_LENDING_COLLATERAL_102 (Securities Lending Minimum Collateral Coverage Floor 102%)
    v_r7 := compliance.seed_core_rule(
        'POST_TRADE_SEC_LENDING_COLLATERAL_102',
        'Securities Lending Minimum Collateral Coverage Floor (102%)',
        'POST',
        'SOFT_WARNING',
        80,
        'ESMA Guidelines on ETFs and other UCITS issues (ESMA/2014/937) §43(e); SEC Rule 17 CFR § 240.15c3-3; ICMA GMSLA Schedule',
        ARRAY['EU', 'US'],
        '{"daily_mark_to_market": true, "min_collateral_ratio": "1.020000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.sec_lending_collateral_ratio"},"operator":"GREATER_THAN_OR_EQUAL","right":{"type":"PARAM","name":"min_collateral_ratio"}}'::jsonb,
        'ACTIVE'
    );

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
            WHEN 'POST_TRADE_SOVEREIGN_EXPOSURE_35'   THEN '383b47f781ad25e13de1f0a7d0d6fba0e90609376a3d0c60d6c0c5d27e4355f3'
            WHEN 'POST_TRADE_AGENCY_SUPRA_25'         THEN '752253e0e4cfe80e3fe0e10c488e79644caf90ab877138a4cfaaf8b3dcce3f60'
            WHEN 'POST_TRADE_MUNI_OBLIGOR_10'         THEN '62e1f7ff199c5defa5773644883c6497b5460f6a59101aa68b0baf53932021d8'
            WHEN 'POST_TRADE_CCP_CLEARING_EXPOSURE_15' THEN 'cd121eb77c1c719967e8e271204ca6bc6c471080fa31295aab64858afc6dacf3'
            WHEN 'POST_TRADE_CUSTODIAN_CONCENTRATION_20' THEN '6a81c7b47a8b3fe1fa63af67282944ffa6c20b7d5f471e983bb28e44406dc875'
            WHEN 'POST_TRADE_BANK_DEPOSIT_20'         THEN '6eb0c014457535bb3351059010abb0547e25d23d6992a9b03c38222cd6d44023'
            WHEN 'POST_TRADE_SEC_LENDING_COLLATERAL_102' THEN 'ca5ae756f71aef6cad6a8ebb578b1dd547e58c4df04a99a26c2e5b3c808a312b'
            ELSE encode(sha256(('v1|' || r.ast_condition::text || '|' || r.parameter_thresholds::text || '|' || COALESCE(r.citation, ''))::bytea), 'hex')
        END,
        COALESCE(encode(sha256(r.compiled_bytecode), 'hex'), encode(sha256(''::bytea), 'hex')),
        'system_seed_v1',
        now()
    FROM compliance.compliance_rule r
    WHERE r.id IN (v_r1, v_r2, v_r3, v_r4, v_r5, v_r6, v_r7)
    ON CONFLICT (rule_id, version) DO NOTHING;

    -- 3. Seed Fourth Licensable Pack: POST_TRADE_MONITORING (all 14 post-trade rules)
    INSERT INTO compliance.compliance_ruleset_membership (ruleset_code, rule_id)
    SELECT 'POST_TRADE_MONITORING', r.id
    FROM compliance.compliance_rule r
    WHERE r.tenant_id = v_gold_tenant
      AND r.rule_code IN (
          'UCITS_5_10_40',
          'SEC_144A_QIB_HOLDING',
          'MARGIN_UTILIZATION_80',
          'POST_TRADE_GROUP_ISSUER_20',
          'POST_TRADE_ISSUER_DEBT_15',
          'POST_TRADE_COUNTERPARTY_PFE_10',
          'POST_TRADE_CASH_MIN_5',
          'POST_TRADE_SOVEREIGN_EXPOSURE_35',
          'POST_TRADE_AGENCY_SUPRA_25',
          'POST_TRADE_MUNI_OBLIGOR_10',
          'POST_TRADE_CCP_CLEARING_EXPOSURE_15',
          'POST_TRADE_CUSTODIAN_CONCENTRATION_20',
          'POST_TRADE_BANK_DEPOSIT_20',
          'POST_TRADE_SEC_LENDING_COLLATERAL_102'
      )
    ON CONFLICT (ruleset_code, rule_id) DO NOTHING;

END $$;
