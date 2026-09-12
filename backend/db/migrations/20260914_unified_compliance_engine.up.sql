-- 20260914_unified_compliance_engine.up.sql
-- Unified 5-Stage Institutional Investment Compliance Engine

CREATE SCHEMA IF NOT EXISTS catalog_compliance;

-- 1. Declarative Compliance Mandate Definitions (Rule 1: Config-Before-Code)
CREATE TABLE IF NOT EXISTS catalog_compliance.mandate_definitions (
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    rule_code VARCHAR(100) NOT NULL,
    rule_name VARCHAR(200) NOT NULL,
    rule_category VARCHAR(50) NOT NULL, -- CONCENTRATION, ALLOCATION, RATING_MATRIX, BENCHMARK_VARIANCE
    evaluation_mode VARCHAR(50) NOT NULL DEFAULT 'PRE_TRADE_AND_IN_TRADE', -- PRE_TRADE, IN_TRADE, POST_TRADE, ALL
    severity VARCHAR(20) NOT NULL DEFAULT 'HARD_BLOCK', -- HARD_BLOCK, SOFT_WARNING
    
    numerator_ast JSONB NOT NULL DEFAULT '{}'::jsonb,
    denominator_type VARCHAR(50) NOT NULL DEFAULT 'ACCT_AUM',
    grouping_dimension VARCHAR(100),
    operator VARCHAR(10) NOT NULL,
    threshold_val NUMERIC(18, 6) NOT NULL,
    warning_tolerance_band NUMERIC(18, 6) DEFAULT 0.000000,
    
    is_lookthrough_enabled BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_compliance_rule UNIQUE (tenant_id, rule_code)
);

-- 2. Look-Through Fund Constituent Mappings
CREATE TABLE IF NOT EXISTS catalog_compliance.fund_lookthrough_weights (
    mapping_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    fund_security_node_id UUID NOT NULL,
    constituent_security_node_id UUID NOT NULL,
    weight_pct NUMERIC(8, 6) NOT NULL,
    as_of_date DATE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_fund_constituent UNIQUE (tenant_id, fund_security_node_id, constituent_security_node_id, as_of_date)
);

-- 3. SEC Rule 17a-4 Pre-Trade Audit & Breach Ledger (WORM Storage)
CREATE TABLE IF NOT EXISTS catalog_compliance.pretrade_audit_ledger (
    audit_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    trade_ticket_id VARCHAR(100) NOT NULL,
    portfolio_node_id UUID NOT NULL,
    security_node_id UUID NOT NULL,
    rule_code VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    
    evaluated_numerator NUMERIC(28, 4) NOT NULL,
    evaluated_denominator NUMERIC(28, 4) NOT NULL,
    evaluated_ratio_pct NUMERIC(18, 6) NOT NULL,
    threshold_limit_pct NUMERIC(18, 6) NOT NULL,
    breach_delta NUMERIC(18, 6) NOT NULL,
    compliance_status VARCHAR(50) NOT NULL,
    
    execution_time_ns BIGINT NOT NULL,
    pm_override_reason TEXT,
    pm_override_user_id VARCHAR(100),
    merkle_leaf_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 4. Exception & Break Triage Ledger (Enhanced Reopen Logic)
CREATE TABLE IF NOT EXISTS catalog_compliance.exception_triage_instances (
    instance_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    rule_id UUID NOT NULL REFERENCES catalog_compliance.mandate_definitions(rule_id) ON DELETE CASCADE,
    portfolio_node_id UUID NOT NULL,
    security_node_id UUID,
    
    exception_status VARCHAR(30) NOT NULL DEFAULT 'OPEN',
    last_evaluated_ratio NUMERIC(18, 6) NOT NULL,
    root_cause_narrative TEXT,
    suggested_remediation_json JSONB,
    reopen_count INT DEFAULT 0,
    last_reopened_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_compliance_audit_search 
ON catalog_compliance.pretrade_audit_ledger (tenant_id, portfolio_node_id, created_at DESC);
