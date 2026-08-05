-- ============================================================================
-- Make RBAC tenant_id columns nullable for gold copy tenant support
-- Gold copy tenant records can have NULL tenant_id (global/super-admin scope)
-- Non-gold-copy records retain their tenant_id (enforced at application level)
-- No backfill: existing records keep their tenant_id values
-- ============================================================================

ALTER TABLE bp_roles ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE bp_permissions ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE bp_field_permissions ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE bp_approval_delegations ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE bp_delegation_usage_log ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE bp_teams ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE bp_team_members ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE bp_team_permissions ALTER COLUMN tenant_id DROP NOT NULL;
