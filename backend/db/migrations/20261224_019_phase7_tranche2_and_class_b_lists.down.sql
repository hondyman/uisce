-- 20261224_019_phase7_tranche2_and_class_b_lists.down.sql

DO $$
DECLARE
    v_rules TEXT[] := ARRAY['POST_TRADE_SEC_SCHEDULE_13G_PASSIVE', 'POST_TRADE_ERISA_PLAN_ASSET_25PCT'];
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

            ALTER TABLE compliance.compliance_evaluation_event DISABLE TRIGGER trg_prevent_eval_mutation;
            ALTER TABLE compliance.compliance_rule DISABLE TRIGGER trg_enforce_rule_version_snapshot;
            DELETE FROM compliance.compliance_rule WHERE id = r_id;
            ALTER TABLE compliance.compliance_rule ENABLE TRIGGER trg_enforce_rule_version_snapshot;
            ALTER TABLE compliance.compliance_evaluation_event ENABLE TRIGGER trg_prevent_eval_mutation;
        END IF;
    END LOOP;

    -- Remove rule family
    DELETE FROM compliance.compliance_rule_family
    WHERE family_code = 'FIDUCIARY_AND_PLAN_ASSETS';

    -- Drop investor classification table
    DROP TABLE IF EXISTS compliance.compliance_investor_classification CASCADE;

END $$;
