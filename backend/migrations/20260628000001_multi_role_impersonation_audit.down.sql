-- =============================================================================
-- Migration rollback: Multi-role impersonation audit infrastructure
-- =============================================================================

DROP VIEW IF EXISTS v_impersonation_activity_feed;

DROP TABLE IF EXISTS impersonation_action_audit;

DROP INDEX IF EXISTS idx_paa_admin_role;

ALTER TABLE platform_admin_audit
    DROP COLUMN IF EXISTS admin_role;
