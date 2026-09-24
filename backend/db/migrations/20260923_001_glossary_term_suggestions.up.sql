-- glossary_term_suggestions: caches deterministic derivation output per column
-- so wizard reopen and reject+regenerate cycles hit Postgres instead of recomputing.
--
-- Key: (tenant_id, qualified_path) — unique per column per tenant.
-- ON CONFLICT DO UPDATE: re-running the wizard for the same column refreshes
-- the cached suggestion.
--
-- Column semantics:
--   column_node_name:   denormalized from catalog_node (avoids join on reopen)
--   semantic_name:      post-derivation PascalCase name (e.g., "CustomerIdentifier")
--   business_name:      post-derivation title-case name (e.g., "Customer Identifier")
--   base_generic_term:  non-empty only for address-line / bare-generic cases
--   derived_via:        which derivation path produced the name
--   definition_cache_key: sha256 hash for Commit 3's definition dedup

DO $$
BEGIN
    CREATE SCHEMA IF NOT EXISTS sml;
EXCEPTION
    WHEN insufficient_privilege THEN
        RAISE NOTICE 'sml schema creation skipped (no permission); relying on existing schema';
END
$$;

CREATE TABLE IF NOT EXISTS sml.glossary_term_suggestions (
    tenant_id            uuid NOT NULL,
    datasource_id        uuid NOT NULL,
    qualified_path       text NOT NULL,
    column_node_name     text NOT NULL,
    semantic_name        text NOT NULL,
    business_name        text NOT NULL,
    base_generic_term    text,
    derived_via          text NOT NULL DEFAULT 'deterministic',
    definition_cache_key text NOT NULL DEFAULT '',
    computed_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, qualified_path)
);

CREATE INDEX IF NOT EXISTS idx_glossary_term_suggestions_datasource
    ON sml.glossary_term_suggestions (tenant_id, datasource_id, column_node_name);
