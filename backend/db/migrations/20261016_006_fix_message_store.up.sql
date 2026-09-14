-- 20261016_006_fix_message_store.up.sql
-- Postgres-backed message store for quickfix. Replaces the in-memory store
-- (backend/internal/fix/adapter.go:124) so that MsgSeqNum persists across
-- acceptor restarts.
--
-- See HANDOFF_FIX_OVER_PIPELINE.md §8 (Amendment 3). Without this, any
-- acceptor restart resets sequence numbers to 1 and the broker force-logs
-- the session out.
--
-- Schema mirrors quickfix's MessageStoreFactory contract:
--   - (session_id, msg_seq_num) is the primary key
--   - message is the full raw bytes (so quickfix can replay on resend requests)
--   - created_at is for audit; quickfix itself doesn't need it
--
-- This table does NOT need RLS — it is keyed on session_id, not tenant_id.
-- Access is gated by the admin API (127.0.0.1:8981) which is itself
-- localhost-only with shared-secret auth. The gold-copy sync role bypasses
-- for admin/audit queries.

BEGIN;

CREATE TABLE IF NOT EXISTS fix_message_store (
    session_id TEXT NOT NULL,
    msg_seq_num BIGINT NOT NULL,
    message BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (session_id, msg_seq_num)
);

CREATE INDEX IF NOT EXISTS idx_fix_message_store_session_created
    ON fix_message_store (session_id, created_at DESC);

COMMIT;
