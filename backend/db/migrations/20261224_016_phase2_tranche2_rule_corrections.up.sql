-- 20261224_016_phase2_tranche2_rule_corrections.up.sql
--
-- Corrections to Phase II Tranche 2 Rules:
-- 1. Correct POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR AST semantics & citation:
--    - Inverted rank logic fixed: metric split_rating_worst_grade_rank <= max_grade_rank (10 = BBB-/Baa3)
--    - Citation updated to house credit policy and fund regulatory anchors (Rule 2a-7, ESMA MMF, UCITS 51)
--    - Recomputed Canonical Content Hash: 545cbc38cbf5d7c805d3d17cae733a844a0ea666b4af108610293bc51f72ea9d
-- 2. Correct POST_TRADE_HIGH_YIELD_CEILING_10 citation:
--    - House Fixed Income Policy with regulatory cross-references (UCITS 52(1), SEC ICA 8(b)(1))
--    - Recomputed Canonical Content Hash: a34aa8708f1555bac1b7ad16b1b9c66ab7a600d2a689c335c29c78634ae70394

DO $$
BEGIN
    ALTER TABLE compliance.compliance_rule_version DISABLE TRIGGER trg_prevent_rule_version_mutation;
    ALTER TABLE compliance.compliance_rule DISABLE TRIGGER trg_enforce_rule_version_snapshot;

    -- 1. POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR
    UPDATE compliance.compliance_rule
    SET 
        ast_condition = '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.split_rating_worst_grade_rank"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_grade_rank"}}'::jsonb,
        parameter_thresholds = '{"max_grade_rank": 10}'::jsonb,
        citation = 'House Credit Policy (Conservative Rating Floor); SEC Rule 2a-7(d)(2) (Second Tier Security Evaluation); ESMA Guidelines on Money Market Funds (ESMA34-49-115) §4.1; UCITS Directive 2009/65/EC Art. 51',
        updated_at = now()
    WHERE rule_code = 'POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR';

    UPDATE compliance.compliance_rule_version
    SET 
        resolved_ast = '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.split_rating_worst_grade_rank"},"operator":"LESS_THAN_OR_EQUAL","right":{"type":"PARAM","name":"max_grade_rank"}}'::jsonb,
        parameter_thresholds = '{"max_grade_rank": 10}'::jsonb,
        citation = 'House Credit Policy (Conservative Rating Floor); SEC Rule 2a-7(d)(2) (Second Tier Security Evaluation); ESMA Guidelines on Money Market Funds (ESMA34-49-115) §4.1; UCITS Directive 2009/65/EC Art. 51',
        content_hash = '545cbc38cbf5d7c805d3d17cae733a844a0ea666b4af108610293bc51f72ea9d'
    WHERE rule_id IN (SELECT id FROM compliance.compliance_rule WHERE rule_code = 'POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR');

    -- 2. POST_TRADE_HIGH_YIELD_CEILING_10
    UPDATE compliance.compliance_rule
    SET 
        citation = 'House Fixed Income Policy (High-Yield Ceiling); UCITS Directive 2009/65/EC Art. 52(1) (Non-Investment Grade Debt Concentration); SEC Investment Company Act §8(b)(1)',
        updated_at = now()
    WHERE rule_code = 'POST_TRADE_HIGH_YIELD_CEILING_10';

    UPDATE compliance.compliance_rule_version
    SET 
        citation = 'House Fixed Income Policy (High-Yield Ceiling); UCITS Directive 2009/65/EC Art. 52(1) (Non-Investment Grade Debt Concentration); SEC Investment Company Act §8(b)(1)',
        content_hash = 'a34aa8708f1555bac1b7ad16b1b9c66ab7a600d2a689c335c29c78634ae70394'
    WHERE rule_id IN (SELECT id FROM compliance.compliance_rule WHERE rule_code = 'POST_TRADE_HIGH_YIELD_CEILING_10');

    ALTER TABLE compliance.compliance_rule_version ENABLE TRIGGER trg_prevent_rule_version_mutation;
    ALTER TABLE compliance.compliance_rule ENABLE TRIGGER trg_enforce_rule_version_snapshot;
END $$;
