-- 20260208_create_ops_error_fingerprints.up.sql
-- Error fingerprinting for noise reduction and error grouping

CREATE TABLE IF NOT EXISTS ops_error_fingerprints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fingerprint TEXT UNIQUE NOT NULL,
    path TEXT,
    status_code INTEGER,
    sample_message TEXT,
    first_seen TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen TIMESTAMPTZ NOT NULL DEFAULT now(),
    count BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ops_error_fingerprints_fingerprint ON ops_error_fingerprints(fingerprint);
CREATE INDEX idx_ops_error_fingerprints_last_seen ON ops_error_fingerprints(last_seen DESC);
CREATE INDEX idx_ops_error_fingerprints_status_code ON ops_error_fingerprints(status_code);

CREATE TABLE IF NOT EXISTS ops_error_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fingerprint_id UUID NOT NULL REFERENCES ops_error_fingerprints(id) ON DELETE CASCADE,
    tenant_id UUID,
    endpoint TEXT,
    status_code INTEGER,
    message TEXT,
    request_id TEXT,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ops_error_events_fingerprint_id ON ops_error_events(fingerprint_id);
CREATE INDEX idx_ops_error_events_tenant_id ON ops_error_events(tenant_id);
CREATE INDEX idx_ops_error_events_occurred_at ON ops_error_events(occurred_at DESC);
