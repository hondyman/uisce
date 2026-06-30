-- =============================================================================
-- Migration: Add scope fields to platform_admin_audit
-- Purpose: Allow impersonation sessions to be scoped to a specific instance,
--          product, or datasource within a tenant, not just tenant-wide.
--          Also add composite index for forensic scope queries.
-- =============================================================================

-- Add scope_kind: narrows the impersonation to a specific resource type
ALTER TABLE platform_admin_audit
    ADD COLUMN IF NOT EXISTS scope_kind TEXT
    CHECK (scope_kind IS NULL OR scope_kind IN ('tenant', 'instance', 'product', 'datasource'));

COMMENT ON COLUMN platform_admin_audit.scope_kind IS
    'Granularity of impersonation scope: ''tenant'' (full tenant, default), '
    '''instance'' (specific tenant instance), ''product'' (specific product), '
    '''datasource'' (specific datasource). NULL = tenant-wide (pre-migration compatible).';

-- Add scope_id: the concrete UUID of the scoped resource
ALTER TABLE platform_admin_audit
    ADD COLUMN IF NOT EXISTS scope_id UUID;

COMMENT ON COLUMN platform_admin_audit.scope_id IS
    'The concrete UUID of the scoped resource (instance/product/datasource). '
    'NULL when scope_kind is NULL or ''tenant'' (full tenant has no narrower id).';

-- Composite index for forensic scope queries: "show all audit events for datasource X"
CREATE INDEX IF NOT EXISTS idx_paa_scope
    ON platform_admin_audit (scope_kind, scope_id)
    WHERE scope_kind IS NOT NULL;

-- Index on session_id is already created in the base schema (idx_paa_session).
-- Verify it exists:
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes WHERE indexname = 'idx_paa_session'
    ) THEN
        CREATE INDEX IF NOT EXISTS idx_paa_session ON platform_admin_audit (session_id);
    END IF;
END $$;

-- =============================================================================
-- Backfill strategy: existing rows have scope_kind = NULL (tenant-wide) by
-- design. No data migration needed.
-- =============================================================================
