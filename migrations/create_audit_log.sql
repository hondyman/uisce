-- Audit log table for immutable Q&A trail
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    question TEXT NOT NULL,
    answer TEXT NOT NULL,
    context_ids TEXT[] DEFAULT '{}',
    provider TEXT NOT NULL,
    response_hash TEXT NOT NULL,
    sources TEXT[] DEFAULT '{}',
    calculation_steps JSONB DEFAULT '[]',
    caveats TEXT[] DEFAULT '{}',
    confidence TEXT,
    version TEXT,
    ip_address TEXT,
    user_agent TEXT,
    session_id TEXT,
    status_code INTEGER,
    latency_ms BIGINT,
    metadata JSONB DEFAULT '{}'
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_audit_log_tenant ON audit_log(tenant_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_user ON audit_log(user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON audit_log(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_session ON audit_log(session_id);

-- Make table append-only (prevent updates/deletes for compliance)
CREATE OR REPLACE FUNCTION prevent_audit_log_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit log is immutable - updates and deletes are not allowed';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_log_immutable_update
    BEFORE UPDATE ON audit_log
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_log_modification();

CREATE TRIGGER audit_log_immutable_delete
    BEFORE DELETE ON audit_log
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_log_modification();
