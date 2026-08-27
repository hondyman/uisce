-- Migration: 20260910_finops_and_governance_extensions.sql
-- Prepaid Wallets, Cost Centers, AI Metering, What-If Exceptions & Two-Tone Audit Ledger

CREATE SCHEMA IF NOT EXISTS finops;
CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS compliance;

-- 1. Prepaid Credit Wallets & Compute Reserves
CREATE TABLE IF NOT EXISTS finops.tenant_credit_wallets (
    wallet_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    balance_credits NUMERIC(14, 4) NOT NULL DEFAULT 0.0000,
    credit_multiplier NUMERIC(6, 4) NOT NULL DEFAULT 1.0000,
    warning_threshold_credits NUMERIC(14, 4) NOT NULL DEFAULT 100.0000,
    hard_stop_enabled BOOLEAN DEFAULT TRUE,
    auto_replenish_enabled BOOLEAN DEFAULT FALSE,
    auto_replenish_amount NUMERIC(14, 4) DEFAULT 1000.0000,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_wallet UNIQUE (tenant_id)
);

-- 2. Cost Center & Strategy Attribution Allocations
CREATE TABLE IF NOT EXISTS finops.cost_center_allocations (
    allocation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    cost_center_code VARCHAR(50) NOT NULL,
    cost_center_name VARCHAR(100) NOT NULL,
    budget_limit_monthly NUMERIC(12, 2) NOT NULL DEFAULT 10000.00,
    current_month_spend NUMERIC(12, 4) NOT NULL DEFAULT 0.0000,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_cost_center UNIQUE (tenant_id, cost_center_code)
);

-- 3. AI Token Metering & Deduplication Ledger (Cardinal Rule 8)
CREATE TABLE IF NOT EXISTS audit.ai_query_metering_ledger (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    proposal_hash VARCHAR(64) NOT NULL,                  -- Dedup Key (Rule 8)
    driving_node_id UUID REFERENCES public.catalog_node(id),
    prompt_type VARCHAR(50) NOT NULL,                    -- BO_SUGGEST, FORMULA_GEN, WHAT_IF_REASONING
    tokens_prompt INT NOT NULL DEFAULT 0,
    tokens_completion INT NOT NULL DEFAULT 0,
    tokens_total INT NOT NULL DEFAULT 0,
    is_heuristic_estimation BOOLEAN DEFAULT FALSE,       -- True if provider failed to emit tokens
    retry_count INT NOT NULL DEFAULT 0,                  -- Retry penalty accounting
    was_successful BOOLEAN DEFAULT TRUE,
    total_token_cost_usd NUMERIC(10, 6) NOT NULL DEFAULT 0.000000,
    executed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_meter_dedup 
ON audit.ai_query_metering_ledger (tenant_id, driving_node_id, proposal_hash);

-- 4. Dynamic Exception Radar & Compliance Breach Lifecycle
CREATE TABLE IF NOT EXISTS compliance.compliance_exception_instances (
    instance_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    rule_code VARCHAR(100) NOT NULL,
    account_id VARCHAR(50) NOT NULL,
    portfolio_id VARCHAR(50) NOT NULL,
    scope_mode VARCHAR(20) NOT NULL DEFAULT 'POS',       -- POS, POSEXEC, POSORD
    severity VARCHAR(20) NOT NULL,                       -- HARD_BLOCK, SOFT_WARNING
    status VARCHAR(50) NOT NULL DEFAULT 'NEW',           -- NEW, OPEN, CLOSED_CORRECTED, CLOSED_NO_ACTION, REOPENED
    reopen_reason VARCHAR(100),
    evaluated_ratio NUMERIC(18, 6) NOT NULL,
    threshold_limit NUMERIC(18, 6) NOT NULL,
    breach_delta NUMERIC(18, 6) NOT NULL,
    market_value_at_breach NUMERIC(18, 4) NOT NULL,
    constituent_fingerprint VARCHAR(64) NOT NULL,
    is_what_if_simulation BOOLEAN DEFAULT FALSE,         -- True if generated from draft order blotter
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exception_lifecycle 
ON compliance.compliance_exception_instances (tenant_id, portfolio_id, status);

-- 5. Two-Tone Audit Trail Ledger (System Red vs User Blue)
CREATE TABLE IF NOT EXISTS audit.governance_two_tone_comments (
    comment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,                    -- EXCEPTION_INSTANCE, STATEMENT_BURST, BO_MAPPING
    entity_id UUID NOT NULL,
    author_type VARCHAR(20) NOT NULL,                    -- SYSTEM (Red), USER (Blue)
    author_identity VARCHAR(100) NOT NULL,
    action_taken VARCHAR(100) NOT NULL,                  -- STATUS_CHANGE, DISPOSITION_OVERRIDE, DRIFT_ALERT
    comment_text TEXT NOT NULL,
    color_tone VARCHAR(10) NOT NULL DEFAULT 'BLUE',      -- RED, BLUE
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_two_tone_lookup 
ON audit.governance_two_tone_comments (tenant_id, entity_id, created_at ASC);
