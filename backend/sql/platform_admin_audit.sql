-- ============================================================================
-- Platform Admin Audit Table
-- Purpose: Synchronous OLTP audit trail for ALL global-admin impersonation
--          activity. Every row is written in the SAME db transaction as the
--          action that triggered it — no async workers, no message queues.
-- ============================================================================

CREATE TABLE IF NOT EXISTS platform_admin_audit (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),

    -- What happened
    event_type       TEXT        NOT NULL,  -- 'IMPERSONATION_START' | 'IMPERSONATION_END' | 'BREAK_GLASS_ACTION'
    mode             TEXT        NOT NULL,  -- 'read_only' | 'break_glass'

    -- Who did it (immutable — set from the real admin's identity token, never the impersonation token)
    admin_user_id    TEXT        NOT NULL,
    admin_email      TEXT        NOT NULL,
    admin_role       TEXT        NOT NULL DEFAULT 'global_admin', -- 'global_admin' | 'helpdesk' | 'professional_services'

    -- Which tenant was targeted
    target_tenant_id UUID        NOT NULL,

    -- Session linkage — ties START / END / actions for one window together
    session_id       UUID        NOT NULL,

    -- Justification (mandatory for all events)
    reason           TEXT        NOT NULL,
    ticket_reference TEXT,                  -- Required for break_glass mode; optional for read_only

    -- Session window metadata
    duration_minutes INT,
    expires_at       TIMESTAMPTZ,

    -- Network metadata (from the originating HTTP request)
    ip_address       TEXT,
    user_agent       TEXT,

    -- For BREAK_GLASS_ACTION: what Business Object was touched, old/new state
    action_detail    JSONB,

    -- Immutable timestamp
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying by audit consumers / compliance reporters
CREATE INDEX IF NOT EXISTS idx_paa_admin_user   ON platform_admin_audit (admin_user_id);
CREATE INDEX IF NOT EXISTS idx_paa_target_tenant ON platform_admin_audit (target_tenant_id);
CREATE INDEX IF NOT EXISTS idx_paa_session       ON platform_admin_audit (session_id);
CREATE INDEX IF NOT EXISTS idx_paa_event_type    ON platform_admin_audit (event_type);
CREATE INDEX IF NOT EXISTS idx_paa_created       ON platform_admin_audit (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_paa_admin_role    ON platform_admin_audit (admin_role);

-- Comments for documentation
COMMENT ON TABLE platform_admin_audit IS
    'Synchronous OLTP audit trail for global admin impersonation sessions. '
    'Every row is written inside the same DB transaction as the action that generated it. '
    'Do NOT add async writers to this table.';

COMMENT ON COLUMN platform_admin_audit.admin_user_id IS
    'The real admin''s user ID from their primary identity token. Never the impersonation token subject.';

COMMENT ON COLUMN platform_admin_audit.session_id IS
    'Groups all audit rows (START, END, BREAK_GLASS_ACTIONs) for a single impersonation window.';

COMMENT ON COLUMN platform_admin_audit.action_detail IS
    'For BREAK_GLASS_ACTION events: JSON containing business_object_type, business_object_id, '
    'action_type, previous_state (redacted), new_state (redacted), execution_context.';

COMMENT ON COLUMN platform_admin_audit.admin_role IS
    'Role used to initiate the impersonation session: global_admin, helpdesk, or professional_services.';

-- ============================================================================
-- Impersonation Action Audit Table
-- Purpose: Synchronous OLTP micro-audit for each Business Object state change
--          performed during an impersonation window. Written inside the same
--          transaction as the BO mutation (abort on failure).
-- ============================================================================

CREATE TABLE IF NOT EXISTS impersonation_action_audit (
    action_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    impersonation_id   UUID NOT NULL REFERENCES platform_admin_audit(session_id),
    target_tenant_id   UUID NOT NULL,
    bo_key             TEXT NOT NULL,
    bo_instance_id     TEXT NOT NULL,
    state_transition   TEXT NOT NULL,
    payload_snapshot   JSONB NOT NULL,
    executed_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_impersonation_actions
    ON impersonation_action_audit (impersonation_id);

CREATE INDEX IF NOT EXISTS idx_impersonation_actions_tenant
    ON impersonation_action_audit (target_tenant_id, executed_at);

CREATE INDEX IF NOT EXISTS idx_impersonation_actions_bo
    ON impersonation_action_audit (bo_key, bo_instance_id);

COMMENT ON TABLE impersonation_action_audit IS
    'Synchronous OLTP audit trail for individual Business Object actions taken during an impersonation session.';

-- ============================================================================
-- Compliance view: session lineage to BO actions
-- ============================================================================

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
