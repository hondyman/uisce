-- =============================================================================
-- Migration: Multi-role impersonation audit infrastructure
-- Purpose:
--   1. Add admin_role to platform_admin_audit so every session records which
--      role (global_admin, helpdesk, professional_services) initiated it.
--   2. Create impersonation_action_audit for synchronous, transaction-bound
--      micro-audit of Business Object state changes performed during an
--      impersonation window.
--   3. Add a compliance view joining session metadata to individual actions.
-- =============================================================================

-- 1. Role column on the session-level audit table.
ALTER TABLE platform_admin_audit
    ADD COLUMN IF NOT EXISTS admin_role TEXT NOT NULL DEFAULT 'global_admin';

COMMENT ON COLUMN platform_admin_audit.admin_role IS
    'Role used to initiate the impersonation session: global_admin, helpdesk, or professional_services.';

CREATE INDEX IF NOT EXISTS idx_paa_admin_role
    ON platform_admin_audit (admin_role);

-- 2. Per-BO action audit table (micro context).
CREATE TABLE IF NOT EXISTS impersonation_action_audit (
    action_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    impersonation_id   UUID NOT NULL,
    target_tenant_id   UUID NOT NULL,
    bo_key             TEXT NOT NULL,
    bo_instance_id     TEXT NOT NULL,
    state_transition   TEXT NOT NULL,
    payload_snapshot   JSONB NOT NULL,
    executed_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE impersonation_action_audit IS
    'Synchronous OLTP audit trail for individual Business Object actions taken during an impersonation session.';

CREATE INDEX IF NOT EXISTS idx_impersonation_actions
    ON impersonation_action_audit (impersonation_id);

CREATE INDEX IF NOT EXISTS idx_impersonation_actions_tenant
    ON impersonation_action_audit (target_tenant_id, executed_at);

CREATE INDEX IF NOT EXISTS idx_impersonation_actions_bo
    ON impersonation_action_audit (bo_key, bo_instance_id);

-- 3. Compliance officer view: session + action lineage.
CREATE OR REPLACE VIEW v_impersonation_activity_feed AS
SELECT
    s.session_id AS impersonation_id,
    s.admin_email,
    s.admin_role,
    s.ticket_reference,
    s.target_tenant_id,
    s.scope_kind,
    s.scope_id,
    a.bo_key,
    a.bo_instance_id,
    a.state_transition,
    a.payload_snapshot,
    a.executed_at AS action_timestamp
FROM platform_admin_audit s
JOIN impersonation_action_audit a ON s.session_id = a.impersonation_id
WHERE s.event_type = 'IMPERSONATION_START'
ORDER BY a.executed_at DESC;

COMMENT ON VIEW v_impersonation_activity_feed IS
    'Cross-reference between impersonation session metadata and the BO actions performed during each session.';
