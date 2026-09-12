-- 20260906_collective_intelligence_and_shadow_replay.up.sql
-- Collective Intelligence Mesh, Differential Privacy Exchange, Shadow Replay & Staleness Decay

CREATE SCHEMA IF NOT EXISTS catalog_collective;

-- 1. Collective Intelligence Exchange (Anonymized Benchmarks & Heuristics)
CREATE TABLE IF NOT EXISTS catalog_collective.anonymized_heuristics (
    heuristic_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_industry_category VARCHAR(100) NOT NULL, -- PRIVATE_CREDIT, HEDGE_FUND, WEALTH_MANAGEMENT, GSIFI
    concept_type VARCHAR(50) NOT NULL, -- OPERATIONAL_RULE, MDM_SURVIVORSHIP, ROUTING_POLICY
    heuristic_title VARCHAR(255) NOT NULL,
    sanitized_ast_payload JSONB NOT NULL,
    adoption_count BIGINT DEFAULT 1,
    simulated_transactions_count BIGINT DEFAULT 100000,
    dp_epsilon_bound NUMERIC(6, 4) NOT NULL DEFAULT 0.0500,
    efficacy_score NUMERIC(5, 2) NOT NULL DEFAULT 98.50, -- % agreement / pass rate
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Historical Shadow Replay & Financial Impact Backtest Runs
CREATE TABLE IF NOT EXISTS catalog_collective.shadow_replay_runs (
    replay_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    proposal_id UUID NOT NULL,
    rule_key VARCHAR(150) NOT NULL,
    backtest_window_days INT NOT NULL DEFAULT 90,
    historical_transactions_evaluated BIGINT NOT NULL,
    nav_impact_bps NUMERIC(8, 4) NOT NULL DEFAULT 0.0000,
    discrepancy_breaks_count INT NOT NULL DEFAULT 0,
    regulatory_impact_flag BOOLEAN DEFAULT FALSE,
    smt_invariant_proof_passed BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Desk-Level Contextual Predicate Scopes
CREATE TABLE IF NOT EXISTS catalog_collective.desk_contextual_scopes (
    scope_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    concept_key VARCHAR(150) NOT NULL,
    desk_name VARCHAR(100) NOT NULL, -- FIXED_INCOME_CREDIT, ENTERPRISE_RISK, QUANT_EXECUTION
    user_role VARCHAR(100),
    contextual_predicate_sql TEXT NOT NULL,
    override_ast_payload JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_concept_desk UNIQUE (tenant_id, concept_key, desk_name)
);

-- 4. Staleness & Rule Decay Tracking Ledger
CREATE TABLE IF NOT EXISTS catalog_collective.rule_staleness_ledger (
    ledger_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    concept_key VARCHAR(150) NOT NULL,
    last_hit_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    execution_hits_last_180d BIGINT DEFAULT 0,
    staleness_score NUMERIC(5, 2) DEFAULT 100.00, -- 100 = Fresh, 0 = Fully Decayed
    lifecycle_status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE, PENDING_DEPRECATION, ARCHIVED
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_concept_staleness UNIQUE (tenant_id, concept_key)
);
