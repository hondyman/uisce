CREATE TABLE IF NOT EXISTS swift_tenant_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    custodian_id UUID NOT NULL,
    bic_sender TEXT NOT NULL,
    bic_receiver TEXT NOT NULL,
    swift_version TEXT NOT NULL DEFAULT 'MT',
    default_message_type TEXT NOT NULL DEFAULT 'MT541',
    settlement_currency TEXT NOT NULL DEFAULT 'USD',
    pipeline_latency_budget_ms INT NOT NULL DEFAULT 60000,
    settlement_calendar_id UUID,
    error_policy TEXT NOT NULL DEFAULT 'skip_and_log',
    allow_partial_settlement BOOLEAN NOT NULL DEFAULT FALSE,
    reconciliation_interval_sec INT NOT NULL DEFAULT 300,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_modified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, custodian_id)
);

ALTER TABLE swift_tenant_config ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS swift_tenant_config_isolation ON swift_tenant_config;

CREATE POLICY swift_tenant_config_isolation ON swift_tenant_config
    USING (
        tenant_id = NULLIF(current_setting('app.tenant_id', 't'), '')::uuid
        OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)
    );

COMMENT ON TABLE swift_tenant_config IS 'Per-tenant SWIFT gateway configuration. GSIFI-isolated. No per-tenant code; topology changes are data operations.';
