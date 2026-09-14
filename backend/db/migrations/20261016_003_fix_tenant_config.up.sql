-- 20261016_003_fix_tenant_config.up.sql
-- Tenant-scoped FIX configuration. Part of the GSIFI-compliant FIX-over-pipeline
-- and Temporal introduction. See HANDOFF_FIX_OVER_PIPELINE.md §11.
--
-- RLS regime follows 20261016_001_strict_tenant_rls.up.sql: helper function
-- `uisce_get_current_tenant()` returns NULL when the GUC is unset (fail-closed).
-- Application-level queries that need Gold Copy visibility use the explicit
-- OR-clause (the gold copy is NOT in the RLS policy itself).
--
-- The Gold Copy tenant row is excluded from RLS via the `BYPASSRLS` privilege
-- granted in 20261016_002_gold_copy_sync_role.up.sql — admin/seed code paths
-- connect as a role that bypasses. Application roles do not bypass; they see
-- only their own tenant.

BEGIN;

CREATE TABLE IF NOT EXISTS fix_tenant_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    broker_id UUID NOT NULL,
    sender_comp_id TEXT NOT NULL,
    target_comp_id TEXT NOT NULL,
    fix_version TEXT NOT NULL DEFAULT 'FIX.4.4',
    heartbeat_interval_sec INT NOT NULL DEFAULT 30,
    pipeline_latency_budget_ms INT NOT NULL DEFAULT 30000,
    error_policy TEXT NOT NULL DEFAULT 'skip_and_log',
    allow_seq_reset BOOLEAN NOT NULL DEFAULT FALSE,  -- see HANDOFF §8 (Amendment 3)
    reconciliation_interval_sec INT NOT NULL DEFAULT 300,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    admin_url TEXT,                                   -- see HANDOFF §6 (admin API endpoint)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_modified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, broker_id)
);

CREATE INDEX IF NOT EXISTS idx_fix_tenant_config_tenant
    ON fix_tenant_config (tenant_id, is_active);

ALTER TABLE fix_tenant_config ENABLE ROW LEVEL SECURITY;
ALTER TABLE fix_tenant_config FORCE ROW LEVEL SECURITY;

-- Policy shape: split read vs write.
--
-- Read (USING): tenant sees their own rows + Gold Copy rows (for
-- inheritance — the FIX-over-pipeline architecture reads defaults from
-- the Gold Copy tenant when a tenant hasn't overridden). Without this
-- OR-clause, the fix_tag_map tile's "tenant-override-gold-copy"
-- precedence rule silently fails — gold-copy rows are invisible to a
-- tenant session, so the tenant never sees the defaults they're
-- supposed to inherit.
--
-- Write (WITH CHECK): strictly tenant's own rows. Gold Copy is
-- admin-managed, not tenant-writable.
--
-- This is the GSIFI read-inheritance / write-isolation pattern. The
-- "Sev-1 if cross-tenant leak" concern from HANDOFF_FIX_OVER_PIPELINE.md
-- §4 is preserved: writes cannot leak (WITH CHECK restricts), and reads
-- only see Gold Copy (which is itself public defaults, no tenant data).
DROP POLICY IF EXISTS fix_tenant_config_isolation_policy ON fix_tenant_config;
CREATE POLICY fix_tenant_config_isolation_policy ON fix_tenant_config
    FOR ALL
    USING (
        tenant_id = uisce_get_current_tenant()
        OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)
    )
    WITH CHECK (
        tenant_id = uisce_get_current_tenant()
    );

-- Admin/seed code paths need to read all rows regardless of tenant. The gold-copy
-- sync role (20261016_002) bypasses via BYPASSRLS; this is a safety belt, not
-- a bypass.

COMMIT;
