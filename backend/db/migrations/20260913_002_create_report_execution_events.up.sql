-- =============================================================================
-- 20260913_002_create_report_execution_events.up.sql
--
-- Creates the report_execution_events append-only audit trail for the monitoring
-- feature (Phase 1).
--
-- Design decisions documented inline:
--   - actor_id vocabulary: defined per writer in the column comment
--   - tenant_id denormalized: required for RLS + CDC partitioning
--   - FORCE ROW LEVEL SECURITY: matches report_executions posture
--   - WITH CHECK on inserts: catches writer bugs that set wrong tenant_id
--   - ON DELETE CASCADE: events deleted with their execution (soft-delete only
--     in current model; cascade is intentional and documented)
--   - Insert-only enforcement: app role gets INSERT+SELECT; PUBLIC revoked
--
-- Named constants (use in application code, not magic values):
--   PHASE1_GO_LIVE_DATE = '2026-09-13T00:00:00Z'
--     Used in SweepStaleExecutions broken-chain date cutoff.
--     Executions created before this date are pre-instrumentation;
--     excluded from broken-chain detection (not migrated, filtered at query time).
-- =============================================================================

CREATE TABLE IF NOT EXISTS public.report_execution_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id    UUID NOT NULL REFERENCES public.report_executions(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL,

    -- Event vocabulary
    event           TEXT NOT NULL,
    --   'CREATED'         : execution inserted (writer 1)
    --   'STARTED'         : execution moved to running (writer 2)
    --   'COMPLETED'       : execution succeeded (writer 4)
    --   'FAILED'          : execution failed with error (writer 3)
    --   'SWEEP_RECONCILED': swept as stale by reconciler (writer 5)
    --   'DISPATCH_FAILED'  : trigger dispatch failed (writer 1 caller context)

    from_status     TEXT NULL,   -- nullable for CREATED; the status being transitioned FROM
    to_status       TEXT NOT NULL,

    -- actor_id vocabulary (by writer):
    --   Writer 1 (pending insert):  user_id of the API caller (from JWT / context)
    --   Writer 2 (running):         'system:executor' (the temporal executor service)
    --   Writer 3 (markFailed):       'system:executor' (same)
    --   Writer 4 (StoreResult):      'system:activity' (the Temporal activity)
    --   Writer 5 (SweepStale):       'system:sweep'
    --   DISPATCH_FAILED:             user_id of the trigger caller (from JWT / context)
    actor_id        TEXT NOT NULL,

    detail          JSONB NULL,
    --   CREATED:        {}
    --   STARTED:        {}
    --   COMPLETED:      {output_url, rows_processed, execution_time_ms}
    --   FAILED:         {error_message}
    --   SWEEP_RECONCILED: {reason: "stale_timeout"}
    --   DISPATCH_FAILED: {error_message}

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for common query shapes
CREATE INDEX IF NOT EXISTS idx_ree_execution_id_created_at
    ON public.report_execution_events(execution_id, created_at);

CREATE INDEX IF NOT EXISTS idx_ree_created_at
    ON public.report_execution_events(created_at);

CREATE INDEX IF NOT EXISTS idx_ree_tenant_id_created_at
    ON public.report_execution_events(tenant_id, created_at);

-- RLS: FORCE to match report_executions posture (owner bypasses non-FORCE RLS silently)
ALTER TABLE public.report_execution_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.report_execution_events FORCE ROW LEVEL SECURITY;

-- Tenant isolation: same policy shape as report_executions
-- USING clause: governs SELECT (read filter)
-- WITH CHECK clause: governs INSERT (write filter — catches writer bugs setting wrong tenant_id)
CREATE POLICY ree_tenant_isolation ON public.report_execution_events
    USING (tenant_id = current_setting('uisce.current_tenant', true)::UUID)
    WITH CHECK (tenant_id = current_setting('uisce.current_tenant', true)::UUID);

-- Grant block (uncommented — runner does raw execution, role name must be literal):
--
-- App role: INSERT+SELECT only — no UPDATE or DELETE
-- The explicit REVOKE is self-documenting; without it, the absence of grant is the constraint.
GRANT INSERT, SELECT ON public.report_execution_events TO app_user;
REVOKE UPDATE, DELETE ON public.report_execution_events FROM PUBLIC;
REVOKE UPDATE, DELETE ON public.report_execution_events FROM app_user;

DO $$ BEGIN
    RAISE NOTICE 'report_execution_events table created';
    RAISE NOTICE '  indexes: idx_ree_execution_id_created_at, idx_ree_created_at, idx_ree_tenant_id_created_at';
    RAISE NOTICE '  RLS: FORCE ROW LEVEL SECURITY with tenant isolation policy + WITH CHECK';
    RAISE NOTICE '  grants: INSERT+SELECT to app_user; UPDATE+DELETE revoked from PUBLIC and app_user';
    RAISE NOTICE 'Named constant: PHASE1_GO_LIVE_DATE = 2026-09-13T00:00:00Z';
END $$;
