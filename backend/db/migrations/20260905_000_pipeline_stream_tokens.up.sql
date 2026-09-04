-- Pipeline stream tokens: single-use, time-limited tokens for SSE auth.
-- A token is a random 32-byte hex string (64 hex chars). The raw token is
-- returned once on creation; only the SHA-256 hash is stored. Consuming
-- the token (via SSE GET) atomically sets used_at so it cannot be replayed.
--
-- Tokens are scoped to a run_id + tenant_id so a token for one tenant's
-- run cannot be used to stream another tenant's run.

CREATE TABLE IF NOT EXISTS pipeline_stream_tokens (
    id              uuid        NOT NULL DEFAULT gen_random_uuid(),
    run_id          uuid        NOT NULL,
    tenant_id       uuid        NOT NULL,
    token_hash      text        NOT NULL,
    expires_at      timestamptz NOT NULL,
    used_at         timestamptz,
    created_at      timestamptz NOT NULL DEFAULT NOW(),

    CONSTRAINT pipeline_stream_tokens_pkey PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS pipeline_stream_tokens_token_hash_active
    ON pipeline_stream_tokens (token_hash)
    WHERE used_at IS NULL;

CREATE INDEX IF NOT EXISTS pipeline_stream_tokens_run_id
    ON pipeline_stream_tokens (run_id);
