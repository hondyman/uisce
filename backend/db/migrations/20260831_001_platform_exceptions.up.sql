-- Persistent, tenant-isolated platform exception hub. Replaces the mock
-- ExceptionAggregator with a real table that dedups repeat occurrences of
-- the same underlying problem via a stable fingerprint.
CREATE TABLE IF NOT EXISTS platform_exceptions (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL,
    type               TEXT NOT NULL,
    severity           TEXT NOT NULL,
    source             TEXT NOT NULL,
    description        TEXT NOT NULL,
    evidence           JSONB NOT NULL DEFAULT '[]'::jsonb,
    fingerprint        TEXT NOT NULL,
    occurrence_count   INT NOT NULL DEFAULT 1,
    first_seen         TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen          TIMESTAMPTZ NOT NULL DEFAULT now(),
    status             TEXT NOT NULL DEFAULT 'open',
    resolved_at        TIMESTAMPTZ,
    resolved_by        TEXT,
    closed_by_ai       BOOLEAN NOT NULL DEFAULT false,
    autofix_attempts   JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Only one OPEN row per fingerprint should exist at a time; that's what
-- makes the upsert-by-fingerprint dedup logic in ExceptionAggregator.Publish
-- safe under concurrent writers.
CREATE UNIQUE INDEX IF NOT EXISTS platform_exceptions_open_fingerprint_uq
    ON platform_exceptions (tenant_id, fingerprint)
    WHERE status IN ('open', 'acknowledged', 'auto_fix_pending');

CREATE INDEX IF NOT EXISTS platform_exceptions_tenant_status_idx
    ON platform_exceptions (tenant_id, status);
CREATE INDEX IF NOT EXISTS platform_exceptions_tenant_type_idx
    ON platform_exceptions (tenant_id, type);
CREATE INDEX IF NOT EXISTS platform_exceptions_last_seen_idx
    ON platform_exceptions (last_seen DESC);
