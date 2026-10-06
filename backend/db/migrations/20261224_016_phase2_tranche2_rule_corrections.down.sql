-- 20261224_016_phase2_tranche2_rule_corrections.down.sql

DO $$
BEGIN
    ALTER TABLE compliance.compliance_rule_version DISABLE TRIGGER trg_prevent_rule_version_mutation;
    ALTER TABLE compliance.compliance_rule DISABLE TRIGGER trg_enforce_rule_version_snapshot;

    -- Revert POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR
    UPDATE compliance.compliance_rule
    SET 
        ast_condition = '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.split_rating_conservative_grade_rank"},"operator":"GREATER_THAN_OR_EQUAL","right":{"type":"PARAM","name":"min_grade_rank"}}'::jsonb,
        parameter_thresholds = '{"min_grade_rank": 10}'::jsonb,
        citation = 'Basel III Supervisory Framework CRE20 §15; Institutional Fixed Income Credit Quality Policy §2.4; SEC Rule 2a-7 Credit Quality Evaluation',
        updated_at = now()
    WHERE rule_code = 'POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR';

    UPDATE compliance.compliance_rule_version
    SET 
        resolved_ast = '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.split_rating_conservative_grade_rank"},"operator":"GREATER_THAN_OR_EQUAL","right":{"type":"PARAM","name":"min_grade_rank"}}'::jsonb,
        parameter_thresholds = '{"min_grade_rank": 10}'::jsonb,
        citation = 'Basel III Supervisory Framework CRE20 §15; Institutional Fixed Income Credit Quality Policy §2.4; SEC Rule 2a-7 Credit Quality Evaluation',
        content_hash = '7e6e81b266d8c6128eda006be7a63056717c2d3502a66cff70c9e98dab297049'
    WHERE rule_id IN (SELECT id FROM compliance.compliance_rule WHERE rule_code = 'POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR');

    -- Revert POST_TRADE_HIGH_YIELD_CEILING_10
    UPDATE compliance.compliance_rule
    SET 
        citation = 'Institutional Investment Grade Mandate Standard §3.1; UCITS Non-Investment Grade Fixed Income Risk Guidelines',
        updated_at = now()
    WHERE rule_code = 'POST_TRADE_HIGH_YIELD_CEILING_10';

    UPDATE compliance.compliance_rule_version
    SET 
        citation = 'Institutional Investment Grade Mandate Standard §3.1; UCITS Non-Investment Grade Fixed Income Risk Guidelines',
        content_hash = '0af91c5c973cebbe4f31d30ca591b4468fd6de7de9631629a7d4c5089a78a0be'
    WHERE rule_id IN (SELECT id FROM compliance.compliance_rule WHERE rule_code = 'POST_TRADE_HIGH_YIELD_CEILING_10');

    ALTER TABLE compliance.compliance_rule_version ENABLE TRIGGER trg_prevent_rule_version_mutation;
    ALTER TABLE compliance.compliance_rule ENABLE TRIGGER trg_enforce_rule_version_snapshot;
END $$;
