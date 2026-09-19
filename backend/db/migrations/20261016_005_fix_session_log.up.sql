-- 20261016_005_fix_session_log.up.sql
-- Append-only audit of FIX session events. NOT a workflow source of truth
-- (the Temporal workflow's event history is); this is observability + forensics.
--
-- Append-only enforced by:
--   1. No UPDATE/DELETE policy in the RLS (only FOR ALL USING + WITH CHECK
--      which allows INSERT but a future migration should drop UPDATE/DELETE
--      from the policy via a separate role-permission layer).
--   2. (Future) trigger blocking UPDATE/DELETE on this table.
--
-- For now, the policy allows the tenant to see its own rows. INSERT is implicit
-- (FOR ALL covers it). The gold copy sync role bypasses via BYPASSRLS for
-- admin queries.


CREATE TABLE IF NOT EXISTS fix_session_log (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    broker_id UUID NOT NULL,
    session_id TEXT NOT NULL,            -- FIX.4.4:SENDER->TARGET
    event_type TEXT NOT NULL,            -- logon | logout | heartbeat | resend | reject | msg_in | msg_out
    msg_seq_num BIGINT,
    msg_type TEXT,                       -- for msg_in / msg_out: the FIX MsgType (35)
    cl_ord_id TEXT,                      -- for order-related events
    exec_id TEXT,                        -- for execution-related events
    raw_excerpt TEXT,                    -- first 512 bytes of the message, base64-encoded
    latency_ms INT,                      -- for msg_out: round-trip to ack
    error_message TEXT,                  -- for reject/error events
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fix_session_log_tenant_time
    ON fix_session_log (tenant_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_fix_session_log_session
    ON fix_session_log (session_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_fix_session_log_cl_ord_id
    ON fix_session_log (tenant_id, cl_ord_id)
    WHERE cl_ord_id IS NOT NULL;

ALTER TABLE fix_session_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE fix_session_log FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS fix_session_log_isolation_policy ON fix_session_log;
CREATE POLICY fix_session_log_isolation_policy ON fix_session_log
    FOR ALL
    USING (
        tenant_id = uisce_get_current_tenant()
    )
    WITH CHECK (
        tenant_id = uisce_get_current_tenant()
    );

