-- Create sml.semantic_term_rejections ahead of 20260923_003, which alters it.
--
-- The table's own migration is 20261021_002_semantic_term_rejections, which
-- sorts AFTER 20260923_003_semantic_term_rejections_preferred_name. alpha applied
-- them in the other order (2026-09-20, then 2026-09-25), but on a fresh database
-- (backend-gated-tests) the runner reaches 20260923_003 first and fails with
-- relation "sml.semantic_term_rejections" does not exist. Both files are already
-- applied, so neither can be edited or renamed; this one sorts between
-- 20260923_002 and 20260923_003 and carries 20261021_002's DDL verbatim.
-- Everything is IF NOT EXISTS: a no-op on alpha, and 20261021_002 becomes a
-- no-op on fresh databases.

-- Idempotent: sml schema may not exist yet in fresh environments.
DO $$
BEGIN
    CREATE SCHEMA IF NOT EXISTS sml;
EXCEPTION
    WHEN insufficient_privilege THEN
        RAISE NOTICE 'sml schema creation skipped (no permission); relying on existing schema';
END
$$;

CREATE TABLE IF NOT EXISTS sml.semantic_term_rejections (
    id              uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       uuid NOT NULL,
    datasource_id   uuid NOT NULL,
    qualified_path  text NOT NULL,
    rejected_name   text NOT NULL,
    rejected_at     timestamptz NOT NULL DEFAULT now(),
    rejected_by     text,
    UNIQUE (tenant_id, datasource_id, qualified_path, rejected_name)
);

CREATE INDEX IF NOT EXISTS idx_semantic_term_rejections_lookup
    ON sml.semantic_term_rejections (tenant_id, datasource_id, qualified_path);
