-- Migration: 20260729000010_data_contract_violations.up.sql
-- Description: Data Contract Violations table for CI/CD schema-change gatekeeping

CREATE TABLE IF NOT EXISTS public.data_contract_violations (
    violation_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         TEXT NOT NULL,
    contract_id       TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'PENDING_REVIEW',
    severity          TEXT NOT NULL,
    target_table      TEXT NOT NULL,
    datasource_id     TEXT NOT NULL,
    proposed_diff     JSONB NOT NULL,
    violations        JSONB NOT NULL,
    detected_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_by       TEXT,
    reviewed_at       TIMESTAMPTZ,
    ticket_id         TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dcv_tenant_status ON public.data_contract_violations(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_dcv_target       ON public.data_contract_violations(tenant_id, target_table);
CREATE INDEX IF NOT EXISTS idx_dcv_datasource   ON public.data_contract_violations(tenant_id, datasource_id);
CREATE INDEX IF NOT EXISTS idx_dcv_detected     ON public.data_contract_violations(tenant_id, detected_at DESC);
