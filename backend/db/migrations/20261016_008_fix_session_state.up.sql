-- 20261016_008_fix_session_state.up.sql
-- Per-session sequence number state for the Postgres-backed quickfix
-- MessageStoreFactory. See HANDOFF_FIX_OVER_PIPELINE.md §8 (Amendment 3)
-- and the implementation in backend/internal/fix/postgres_store.go.
--
-- One row per (session_id). Persisted so that MsgSeqNum survives acceptor
-- restarts. Without this, the broker force-logs the session out on restart
-- because quickfix's quickfix.NewMemoryStoreFactory() resets to 1.
--
-- RLS is not needed (no tenant_id; access gated by the admin API on
-- 127.0.0.1:8981 with shared-secret auth, plus BYPASSRLS for the gold-copy
-- sync role for admin queries).
--
-- Why two tables (this + fix_message_store from migration 006)?
-- quickfix's MessageStore interface separates two concerns that don't
-- share access patterns:
--
--   fix_session_state (this):
--     - One row per session
--     - Holds the *next* sender/target sequence numbers + creation time
--     - Read on every session callback to determine MsgSeqNum
--     - Updated on every sent message
--     - Indexed by session_id (PK)
--
--   fix_message_store (migration 006):
--     - N rows per session, one per (session_id, msg_seq_num)
--     - Holds raw bytes for replay on ResendRequest
--     - Read only when the counterparty requests a resend (rare)
--     - Indexed by (session_id, msg_seq_num) (composite PK)
--
-- Consolidating into one table would either mean a full table scan on
-- every send (to find the next sequence number) or a separate denormalized
-- counter table — neither is better than the split. quickfix's own SQL
-- store in newer versions makes the same split.


CREATE TABLE IF NOT EXISTS fix_session_state (
    session_id TEXT PRIMARY KEY,
    sender_msg_seq_num INT NOT NULL DEFAULT 0,
    target_msg_seq_num INT NOT NULL DEFAULT 0,
    creation_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

