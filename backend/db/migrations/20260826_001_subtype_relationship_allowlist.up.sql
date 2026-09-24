-- 20260826_001_subtype_relationship_allowlist.up.sql
-- Adds relationship_allowlist JSONB column to oms.subtype_registry so that
-- subtype-first polymorphic join scoping can be driven entirely from the
-- catalog registry (Rule 1: Config-Before-Code).

-- Default empty array preserves backward compatibility: existing subtypes
-- default to root-only join paths.

ALTER TABLE oms.subtype_registry
  ADD COLUMN IF NOT EXISTS relationship_allowlist JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN oms.subtype_registry.relationship_allowlist IS
  'Array of relationship edge keys (from catalog_edge.rel_key) that are valid for this subtype. '
  'An empty array means only root-level joins are reachable from this subtype. '
  'Mirrors field_allowlist structure; kept in sync by catalog-admin tooling.';

-- GIN index for efficient containment queries when resolving allowed joins
CREATE INDEX IF NOT EXISTS idx_subtype_registry_rel_allowlist_gin
  ON oms.subtype_registry USING GIN (relationship_allowlist);

-- Seed example: institutional account may use benchmark + mandate schedule joins
-- (Actual values are managed by catalog-admin; this is a documented example.)
-- UPDATE oms.subtype_registry
-- SET relationship_allowlist = '["account_to_benchmark","account_to_mandate_schedule"]'::jsonb
-- WHERE root_object = 'account' AND subtype_code = 'institutional';
