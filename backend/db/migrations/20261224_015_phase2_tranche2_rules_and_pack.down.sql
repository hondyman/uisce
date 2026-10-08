-- 20261224_015_phase2_tranche2_rules_and_pack.down.sql

DO $$
DECLARE
    v_rules TEXT[] := ARRAY[
        'POST_TRADE_SANCTION_LIST_ZERO_TOLERANCE',
        'POST_TRADE_FX_FORWARD_UNHEDGED_30',
        'POST_TRADE_HIGH_YIELD_CEILING_10',
        'POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR',
        'POST_TRADE_ESG_CONTROVERSIAL_WEAPONS_0',
        'POST_TRADE_ESG_THERMAL_COAL_REVENUE_5',
        'POST_TRADE_ESG_TOBACCO_REVENUE_5',
        'POST_TRADE_ILLIQUID_TIER3_ASSETS_10',
        'POST_TRADE_SETTLEMENT_FAIL_CONCENTRATION_5'
    ];
    r_id UUID;
    r_code TEXT;
BEGIN
    FOREACH r_code IN ARRAY v_rules LOOP
        SELECT id INTO r_id FROM compliance.compliance_rule WHERE rule_code = r_code;
        IF r_id IS NOT NULL THEN
            DELETE FROM compliance.compliance_ruleset_membership WHERE rule_id = r_id;
            
            ALTER TABLE compliance.compliance_rule_version DISABLE TRIGGER trg_prevent_rule_version_mutation;
            DELETE FROM compliance.compliance_rule_version WHERE rule_id = r_id;
            ALTER TABLE compliance.compliance_rule_version ENABLE TRIGGER trg_prevent_rule_version_mutation;

            ALTER TABLE compliance.compliance_rule DISABLE TRIGGER trg_enforce_rule_version_snapshot;
            DELETE FROM compliance.compliance_rule WHERE id = r_id;
            ALTER TABLE compliance.compliance_rule ENABLE TRIGGER trg_enforce_rule_version_snapshot;
        END IF;
    END LOOP;
END $$;
