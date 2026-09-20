-- semantic_term_rejections: per-(tenant, datasource, column, rejected_name) rejection set
-- for the glossary wizard "never suggest this name for this column" feature.
--
-- Key: (tenant_id, datasource_id, qualified_path, rejected_name) — level 2 (per-datasource column-level).
-- Pattern-level (level 3) is out of scope for this table; it belongs in a future
-- learned-negative-mappings table that unifies with the abbreviation dictionary.
--
-- Rejection records are permanent — no updated_at, no soft-delete. A mis-click is
-- corrected by recording the correct name, not by un-rejecting the wrong one.
-- The DELETE endpoint exists for explicit undo within a session, not as a general
-- escape hatch.
--
-- Storage contract: preview loads the tenant+datasource rejection set once per request
-- (WHERE tenant_id=$1 AND datasource_id=$2) and filters in-memory. The composite
-- index supports that lookup efficiently.

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
