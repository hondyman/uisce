-- 20260829_differential_privacy_cleanrooms.up.sql
-- Differential Privacy Clean Rooms & Privacy Loss Ledger

CREATE SCHEMA IF NOT EXISTS catalog_dp;

-- 1. Clean Room Federation Policy
CREATE TABLE IF NOT EXISTS catalog_dp.cleanroom_federation (
    cleanroom_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cleanroom_key VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    min_participating_tenants INT NOT NULL DEFAULT 5,
    max_epsilon_per_query NUMERIC(6, 4) NOT NULL DEFAULT 0.5000,
    target_delta NUMERIC(10, 9) NOT NULL DEFAULT 0.000010000,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Tenant Privacy Loss Budgets (Depletion Ledger)
CREATE TABLE IF NOT EXISTS catalog_dp.tenant_privacy_budgets (
    budget_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    cleanroom_id UUID NOT NULL REFERENCES catalog_dp.cleanroom_federation(cleanroom_id) ON DELETE CASCADE,
    allocated_epsilon NUMERIC(8, 4) NOT NULL DEFAULT 10.0000, -- Total epsilon allowance
    consumed_epsilon NUMERIC(8, 4) NOT NULL DEFAULT 0.0000,
    budget_reset_interval VARCHAR(20) NOT NULL DEFAULT 'MONTHLY', -- DAILY, WEEKLY, MONTHLY
    last_reset_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_cleanroom_budget UNIQUE(tenant_id, cleanroom_id),
    CONSTRAINT chk_budget_exhaustion CHECK (consumed_epsilon <= allocated_epsilon)
);

-- 3. Immutable DP Execution Audit Outbox (Merkle Passport)
CREATE TABLE IF NOT EXISTS catalog_dp.dp_query_audit_ledger (
    audit_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cleanroom_id UUID NOT NULL REFERENCES catalog_dp.cleanroom_federation(cleanroom_id),
    requesting_tenant_id UUID NOT NULL,
    query_ast_sha256 VARCHAR(64) NOT NULL,
    participating_tenant_count INT NOT NULL,
    epsilon_spent NUMERIC(6, 4) NOT NULL,
    delta_spent NUMERIC(10, 9) NOT NULL,
    computed_sensitivity NUMERIC(12, 4) NOT NULL,
    raw_aggregate_value NUMERIC(18, 6),
    noise_injected NUMERIC(18, 6) NOT NULL,
    final_perturbed_value NUMERIC(18, 6) NOT NULL,
    merkle_execution_passport VARCHAR(64) NOT NULL,
    executed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dp_audit_cleanroom 
    ON catalog_dp.dp_query_audit_ledger(cleanroom_id, requesting_tenant_id, executed_at DESC);
