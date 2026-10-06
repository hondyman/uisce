-- 20261224_017_phase2_tranche3_waci_and_sfdr_rules.down.sql

DO $$
DECLARE
    v_rules TEXT[] := ARRAY[
        'POST_TRADE_ESG_WACI_PORTFOLIO_CEILING',
        'POST_TRADE_ESG_SCOPE_1_2_EMISSIONS_CEILING',
        'POST_TRADE_ESG_BOARD_GENDER_DIVERSITY_FLOOR',
        'POST_TRADE_ESG_HAZARDOUS_WASTE_RATIO_CEILING',
        'POST_TRADE_EU_TAXONOMY_GREEN_REVENUE_FLOOR',
        'POST_TRADE_LIQUIDITY_COVERAGE_RATIO_BUFFER'
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

    -- Revert Rule 26 parameter rename
    ALTER TABLE compliance.compliance_rule_version DISABLE TRIGGER trg_prevent_rule_version_mutation;
    ALTER TABLE compliance.compliance_rule DISABLE TRIGGER trg_enforce_rule_version_snapshot;

    UPDATE compliance.compliance_rule
    SET 
        ast_condition = '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.split_rating_worst_grade_rank"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_grade_rank"}}'::jsonb,
        parameter_thresholds = '{"max_grade_rank": 10}'::jsonb,
        updated_at = now()
    WHERE rule_code = 'POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR';

    UPDATE compliance.compliance_rule_version
    SET 
        resolved_ast = '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.split_rating_worst_grade_rank"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_grade_rank"}}'::jsonb,
        parameter_thresholds = '{"max_grade_rank": 10}'::jsonb,
        content_hash = '545cbc38cbf5d7c805d3d17cae733a844a0ea666b4af108610293bc51f72ea9d'
    WHERE rule_id IN (SELECT id FROM compliance.compliance_rule WHERE rule_code = 'POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR');

    ALTER TABLE compliance.compliance_rule_version ENABLE TRIGGER trg_prevent_rule_version_mutation;
    ALTER TABLE compliance.compliance_rule ENABLE TRIGGER trg_enforce_rule_version_snapshot;
END $$;
