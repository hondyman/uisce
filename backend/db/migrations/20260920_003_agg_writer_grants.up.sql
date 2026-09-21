-- Migration: grant agg_writer role permissions for CDC aggregate consumer
--
-- The aggregate consumer needs a dedicated Postgres role (not postgres superuser)
-- with targeted grants on the tables it writes to. Two deployments:
--   - aggregate-consumer-alpha  → alpha DB
--   - aggregate-consumer-crims  → crims DB
--
-- Prerequisites (run BEFORE this migration):
--   scripts/create_agg_writer_roles.sh
-- This script creates the roles and sets passwords (from .env), which
-- must NOT appear in committed migration files.
--
-- NOTE: scripts/create_agg_writer_roles.sh also grants consumer_dedupe and agg.*
-- access during its bootstrap step (those tables don't exist until the script
-- runs). Keep the two files in sync when the access model changes.
--
-- The consumer writes aggregates to tables that are ALSO in the CDC publication.
-- This is safe because:
--   1. The consumer's dedupe table (consumer_dedupe) is NOT in any publication
--      — its writes never emit CDC events, so there's no feedback loop.
--   2. The consumer's writes to publication tables (e.g. investment_opportunities
--      UPDATE for screening) emit CDC events; the self-write echo filter in
--      screening.go drops those events by checking ChangedColumns — no loop.
--
-- Grant strategy: minimum privileges for each deployment.
--
-- Alpha writes:
--   rating_stats      → process_templates (UPDATE rating_average, rating_count)
--   cube_versions     → cube_custom_model_versions, cube_security_policy_versions (INSERT)
--   template_rbac     → semantic_query_template_permissions (INSERT),
--                        semantic_query_template_versions (INSERT)
--   screening         → investment_opportunities (UPDATE screening_*, current_stage, stage_*)
--   notify_events     → agg.security_fund_access_change (INSERT),
--                        agg.metrics_registry_changed (INSERT)
--   dedupe store      → consumer_dedupe (INSERT + periodic DELETE)
--
-- Crims writes:
--   identifier_cache  → orm.security (UPDATE isin, cusip, ticker)
--   dedupe store      → consumer_dedupe (INSERT + periodic DELETE)

BEGIN;

-- ============================================================================
-- Alpha: agg_writer_alpha
-- ============================================================================

GRANT USAGE  ON SCHEMA public TO agg_writer_alpha;
GRANT USAGE  ON SCHEMA agg   TO agg_writer_alpha;

-- rating_stats handler: only rating_average, rating_count, updated_at
GRANT SELECT ON process_templates TO agg_writer_alpha;
GRANT UPDATE (rating_average, rating_count, updated_at) ON process_templates TO agg_writer_alpha;

GRANT INSERT ON cube_custom_model_versions         TO agg_writer_alpha;
GRANT INSERT ON cube_security_policy_versions      TO agg_writer_alpha;

GRANT INSERT ON semantic_query_template_permissions TO agg_writer_alpha;
GRANT INSERT ON semantic_query_template_versions   TO agg_writer_alpha;

-- screening handler: only the columns it actually writes
GRANT UPDATE (
    screening_passed, screening_reasons, screening_score,
    screening_completed_at, current_stage, stage_updated_at,
    stage_history, updated_at
) ON investment_opportunities TO agg_writer_alpha;

GRANT INSERT ON agg.security_fund_access_change TO agg_writer_alpha;
GRANT INSERT ON agg.metrics_registry_changed   TO agg_writer_alpha;

-- dedupe store: handler writes (INSERT) + Prune() call (DELETE)
GRANT SELECT, INSERT, DELETE ON consumer_dedupe TO agg_writer_alpha;

-- ============================================================================
-- Crims: agg_writer_crims
-- ============================================================================

GRANT USAGE ON SCHEMA orm TO agg_writer_crims;

-- identifier_cache handler: only the columns it actually writes
GRANT UPDATE (isin, cusip, ticker) ON orm.security TO agg_writer_crims;

-- dedupe store
GRANT SELECT, INSERT, DELETE ON consumer_dedupe TO agg_writer_crims;

COMMIT;
