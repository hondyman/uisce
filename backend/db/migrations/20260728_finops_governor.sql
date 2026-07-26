-- Uisce Multi-Cloud FinOps Cost Governor Schema
-- Rule 7 (Security Mandate)

BEGIN;

-- 1. Tenant FinOps Budget & Cost Thresholds Table
CREATE TABLE IF NOT EXISTS platform.finops_tenant_budget (
    budget_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    max_query_cost_usd NUMERIC(10, 4) DEFAULT 5.00,
    current_month_spend_usd NUMERIC(12, 4) DEFAULT 0.00,
    monthly_limit_usd NUMERIC(12, 4) DEFAULT 1000.00,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id)
);

-- 2. Query Execution Cost Ledger
CREATE TABLE IF NOT EXISTS platform.finops_query_cost_log (
    log_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    query_hash VARCHAR(64) NOT NULL,
    estimated_cost_usd NUMERIC(10, 4) NOT NULL,
    scanned_bytes BIGINT NOT NULL,
    execution_target VARCHAR(50) NOT NULL, -- STARROCKS, TRINO_ICEBERG, POSTGRES
    status VARCHAR(50) DEFAULT 'EXECUTED', -- EXECUTED, REJECTED_BUDGET_EXCEEDED
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_finops_tenant ON platform.finops_query_cost_log(tenant_id, created_at);

COMMIT;
