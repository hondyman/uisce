CREATE TABLE IF NOT EXISTS swift_session_log (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    custodian_id UUID NOT NULL,
    msg_type TEXT NOT NULL,
    transaction_ref TEXT,
    uetr TEXT,
    event_type TEXT NOT NULL,
    raw_excerpt TEXT,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS swift_session_log_tenant_time ON swift_session_log (tenant_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS swift_session_log_uetr ON swift_session_log (uetr) WHERE uetr IS NOT NULL;
CREATE INDEX IF NOT EXISTS swift_session_log_txref ON swift_session_log (tenant_id, transaction_ref) WHERE transaction_ref IS NOT NULL;

COMMENT ON TABLE swift_session_log IS 'Append-only SWIFT message audit log. Not a Temporal workflow source. GSIFI-scoped by tenant_id index; no RLS needed (append-only, no cross-tenant read risk from a known tenant_id column).';
