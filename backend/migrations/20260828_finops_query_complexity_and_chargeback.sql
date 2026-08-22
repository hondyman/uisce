-- 20260828_finops_query_complexity_and_chargeback.sql
-- Pre-Flight Query Complexity Scoring, Execution Plan History & Tenant FinOps Chargeback

CREATE SCHEMA IF NOT EXISTS finops;

-- 1. Tenant Compute Budgets & Circuit Breaker Thresholds
CREATE TABLE IF NOT EXISTS finops.tenant_compute_quotas (
    quota_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    billing_period VARCHAR(7) NOT NULL,              -- YYYY-MM
    monthly_budget_usd NUMERIC(12, 2) NOT NULL DEFAULT 5000.00,
    current_spend_usd NUMERIC(12, 4) NOT NULL DEFAULT 0.0000,
    max_complexity_score_threshold INT DEFAULT 80,   -- Queries exceeding this score are blocked
    max_scanned_bytes_per_query BIGINT DEFAULT 10737418240, -- 10 GB limit per single query
    cpu_ms_rate_usd NUMERIC(10, 8) DEFAULT 0.000035, -- $0.035 per 1,000 CPU seconds
    scanned_gb_rate_usd NUMERIC(10, 6) DEFAULT 0.005000, -- $5.00 per TB scanned
    is_hard_budget_enforced BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_quota_period UNIQUE (tenant_id, billing_period)
);

-- 2. Query Execution Plans & Node-by-Node DAG Records
CREATE TABLE IF NOT EXISTS finops.semantic_query_execution_plans (
    plan_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    query_fingerprint VARCHAR(64) NOT NULL,          -- SHA-256 of normalized AST
    complexity_score INT NOT NULL,                   -- 0 to 100+
    cost_band VARCHAR(20) NOT NULL,                  -- LOW, MODERATE, EXPENSIVE, FORBIDDEN
    estimated_scanned_bytes BIGINT NOT NULL,
    actual_scanned_bytes BIGINT DEFAULT 0,
    total_latency_ms INT NOT NULL,
    cpu_duration_ms INT NOT NULL,
    attributed_cost_usd NUMERIC(10, 6) NOT NULL,
    execution_dag JSONB NOT NULL,                    -- Array of DAG nodes and edges
    optimization_recommendations JSONB DEFAULT '[]'::jsonb,
    is_blocked_by_circuit_breaker BOOLEAN DEFAULT FALSE,
    executed_by_user_id VARCHAR(100),
    executed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_query_plans_lookup 
ON finops.semantic_query_execution_plans (tenant_id, query_fingerprint, executed_at DESC);

-- 3. Tenant Chargeback Itemized Ledger
CREATE TABLE IF NOT EXISTS finops.tenant_chargeback_ledger (
    ledger_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES finops.semantic_query_execution_plans(plan_id) ON DELETE CASCADE,
    engine_type VARCHAR(50) NOT NULL,                -- STARROCKS, ICEBERG, WASM_VECTOR, POSTGRES
    scanned_bytes BIGINT NOT NULL,
    cpu_milliseconds INT NOT NULL,
    line_item_cost_usd NUMERIC(10, 6) NOT NULL,
    charged_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chargeback_tenant_period 
ON finops.tenant_chargeback_ledger (tenant_id, charged_at);
