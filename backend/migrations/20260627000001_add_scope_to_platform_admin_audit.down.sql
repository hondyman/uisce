-- =============================================================================
-- Down migration: Remove scope fields from platform_admin_audit
-- Purpose: Revert the scope_kind and scope_id columns added in the up migration.
-- =============================================================================

DROP INDEX IF EXISTS idx_paa_scope;

ALTER TABLE platform_admin_audit
    DROP COLUMN IF EXISTS scope_id,
    DROP COLUMN IF EXISTS scope_kind;
