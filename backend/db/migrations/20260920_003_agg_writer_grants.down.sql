-- Revoke all column-scoped grants from agg_writer roles.
-- The roles themselves are dropped separately via:
--   DROP ROLE IF EXISTS agg_writer_alpha, agg_writer_crims;
-- (run manually after verifying no active connections remain)
--
-- Note: REVOKE must mirror the column-scoped grants from the up migration.
-- A table-level REVOKE does NOT undo a column-level GRANT.

BEGIN;

-- Alpha
REVOKE USAGE  ON SCHEMA public FROM agg_writer_alpha;
REVOKE USAGE  ON SCHEMA agg   FROM agg_writer_alpha;

REVOKE SELECT ON process_templates FROM agg_writer_alpha;
REVOKE UPDATE (rating_average, rating_count, updated_at) ON process_templates FROM agg_writer_alpha;

REVOKE INSERT ON cube_custom_model_versions         FROM agg_writer_alpha;
REVOKE INSERT ON cube_security_policy_versions      FROM agg_writer_alpha;

REVOKE INSERT ON semantic_query_template_permissions FROM agg_writer_alpha;
REVOKE INSERT ON semantic_query_template_versions   FROM agg_writer_alpha;

REVOKE UPDATE (screening_passed, screening_reasons, screening_score,
               screening_completed_at, current_stage, stage_updated_at,
               stage_history, updated_at) ON investment_opportunities FROM agg_writer_alpha;

REVOKE INSERT ON agg.security_fund_access_change FROM agg_writer_alpha;
REVOKE INSERT ON agg.metrics_registry_changed   FROM agg_writer_alpha;

REVOKE SELECT, INSERT, DELETE ON consumer_dedupe FROM agg_writer_alpha;

-- Crims
REVOKE USAGE ON SCHEMA orm FROM agg_writer_crims;

REVOKE UPDATE (isin, cusip, ticker) ON orm.security FROM agg_writer_crims;

REVOKE SELECT, INSERT, DELETE ON consumer_dedupe FROM agg_writer_crims;

COMMIT;
