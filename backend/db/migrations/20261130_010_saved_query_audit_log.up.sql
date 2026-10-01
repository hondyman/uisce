-- Migration: Add audit log table for saved query execution
CREATE TABLE IF NOT EXISTS data_explorer.saved_query_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    query_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    caller_id TEXT,
    params_hash TEXT NOT NULL,
    row_count INT NOT NULL DEFAULT 0,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    status_code INT NOT NULL DEFAULT 200,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_saved_query_audit_tenant_query 
ON data_explorer.saved_query_audit_log (tenant_id, query_id, executed_at DESC);
