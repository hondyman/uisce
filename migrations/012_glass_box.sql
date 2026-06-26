-- 1. Events Raw (Append-Only Log)
-- Mimics StarRocks behavior in Postgres for now
CREATE TABLE events_raw (
    event_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL,
    seq BIGINT NOT NULL,
    event_type TEXT NOT NULL,
    payload_canon TEXT NOT NULL,
    payload_hash CHAR(64) NOT NULL,
    parent_hash CHAR(64),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_events_run_seq ON events_raw (run_id, seq);

-- 2. Artifacts (Versioned Storage)
-- Mimics Iceberg behavior
CREATE TABLE artifacts (
    artifact_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL,
    type TEXT NOT NULL, -- 'prompt', 'output', 'policy_eval'
    content_hash CHAR(64) NOT NULL,
    schema_version INT NOT NULL,
    content JSONB, -- Storing content directly for now, would be S3/Iceberg ref in prod
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_artifacts_run ON artifacts (run_id);
CREATE INDEX idx_artifacts_hash ON artifacts (content_hash);
