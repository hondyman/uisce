-- definition_cache: deduplicates LLM-generated definitions across columns that
-- derive the same semantic name. Two columns with the same derived name and
-- same derivation context share one definition.
--
-- The cache_key is sha256(namingLogicVersion || abbreviationsVersion ||
-- derivedTokensJoined || contextPart). Bump the Go constants when the
-- abbreviation dictionary or naming logic changes.

DO $$
BEGIN
    CREATE SCHEMA IF NOT EXISTS sml;
EXCEPTION
    WHEN insufficient_privilege THEN
        RAISE NOTICE 'sml schema creation skipped (no permission); relying on existing schema';
END
$$;

CREATE TABLE IF NOT EXISTS sml.glossary_term_definition_cache (
    cache_key       text NOT NULL,
    semantic_name   text NOT NULL,
    definition      text NOT NULL,
    definition_source text NOT NULL DEFAULT 'llm',
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (cache_key)
);

CREATE INDEX IF NOT EXISTS idx_glossary_term_definition_cache_name
    ON sml.glossary_term_definition_cache (semantic_name);
