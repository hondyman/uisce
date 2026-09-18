CREATE TABLE IF NOT EXISTS swift_reconciliation_report (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    custodian_id UUID NOT NULL,
    lookback_start TIMESTAMPTZ NOT NULL,
    lookback_end TIMESTAMPTZ NOT NULL,
    instructions_scanned INT NOT NULL DEFAULT 0,
    mismatches_count INT NOT NULL DEFAULT 0,
    mismatches JSONB NOT NULL DEFAULT '[]',
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    workflow_run_id TEXT
);

CREATE INDEX IF NOT EXISTS swift_recon_report_tenant_time ON swift_reconciliation_report (tenant_id, generated_at DESC);

ALTER TABLE swift_reconciliation_report ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS swift_reconciliation_report_isolation ON swift_reconciliation_report;

CREATE POLICY swift_reconciliation_report_isolation ON swift_reconciliation_report
    USING (
        tenant_id = NULLIF(current_setting('app.tenant_id', 't'), '')::uuid
        OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)
    );

COMMENT ON TABLE swift_reconciliation_report IS 'Persisted output of SWIFTReconciliationWorkflow runs. GSIFI-isolated.';
