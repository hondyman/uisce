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
