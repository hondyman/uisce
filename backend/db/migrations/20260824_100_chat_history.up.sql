-- Chat History telemetry ledger (ai_telemetry schema).
-- Stores chat sessions + per-message transcripts for the Uuisce Chat History view.

CREATE SCHEMA IF NOT EXISTS ai_telemetry;

CREATE TABLE IF NOT EXISTS ai_telemetry.chat_session (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL,
    conversation_id  UUID NOT NULL,
    agent_id         TEXT NOT NULL,
    agent_version    TEXT,
    user_id          TEXT NOT NULL,
    user_email       TEXT,
    view_type        TEXT NOT NULL CHECK (view_type IN ('end_user', 'admin')),
    embedded         BOOLEAN NOT NULL DEFAULT FALSE,
    embed_surface    TEXT CHECK (embed_surface IN ('studio', 'iframe', 'slack', 'teams', 'mcp') OR embed_surface IS NULL),
    started_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at         TIMESTAMPTZ,
    message_count    INTEGER NOT NULL DEFAULT 0,
    feedback_score   SMALLINT CHECK (feedback_score IN (-1, 1) OR feedback_score IS NULL),
    feedback_comment TEXT,
    metadata         JSONB NOT NULL DEFAULT '{}'::jsonb,
    trace_id         TEXT,
    previous_hash    TEXT,
    current_hash     TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_chat_session_tenant_conv UNIQUE (tenant_id, conversation_id)
);

CREATE INDEX IF NOT EXISTS idx_chat_session_tenant_started ON ai_telemetry.chat_session (tenant_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_session_tenant_agent   ON ai_telemetry.chat_session (tenant_id, agent_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_session_feedback       ON ai_telemetry.chat_session (tenant_id, feedback_score) WHERE feedback_score IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_chat_session_trace          ON ai_telemetry.chat_session (trace_id) WHERE trace_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS ai_telemetry.chat_message (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   UUID NOT NULL REFERENCES ai_telemetry.chat_session(id) ON DELETE CASCADE,
    tenant_id    UUID NOT NULL,
    seq          INTEGER NOT NULL,
    role         TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system', 'tool')),
    content      TEXT NOT NULL,
    content_json JSONB,
    tool_calls   JSONB,
    chart_spec   JSONB,
    trace_steps  JSONB,
    latency_ms   INTEGER,
    token_in     INTEGER,
    token_out    INTEGER,
    trace_id     TEXT,
    span_id      TEXT,
    error        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_chat_message_session_seq UNIQUE (session_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_chat_message_session ON ai_telemetry.chat_message (session_id, seq);
CREATE INDEX IF NOT EXISTS idx_chat_message_tenant  ON ai_telemetry.chat_message (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_message_trace   ON ai_telemetry.chat_message (trace_id) WHERE trace_id IS NOT NULL;