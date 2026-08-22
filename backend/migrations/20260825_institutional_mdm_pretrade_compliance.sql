-- 20260825_institutional_mdm_pretrade_compliance.sql
-- MDM Survivorship Rules, Compliance Mandates, Look-Through Constituents & Pre-Trade Audit Ledger

-- 1. Multi-Source Golden Record Survivorship Rules
CREATE TABLE IF NOT EXISTS public.mdm_survivorship_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,      -- SECURITY_PRICE, SECURITY_MASTER, ENTITY_LEGAL
    field_name VARCHAR(100) NOT NULL,      -- px_last, credit_rating, esg_score
    strategy VARCHAR(50) NOT NULL,         -- SOURCE_PRIORITY, MOST_RECENT, CONFIDENCE_SCORE, CONSERVATIVE_MIN
    priority_list JSONB DEFAULT '[]'::jsonb, -- ["BLOOMBERG", "REFINITIV", "CRIMS", "INTERNAL"]
    staleness_threshold_sec INT DEFAULT 86400,
    confidence_weights JSONB DEFAULT '{}'::jsonb,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_mdm_survivorship UNIQUE (tenant_id, entity_type, field_name)
);

-- 2. Declarative Compliance Rule Definitions
CREATE TABLE IF NOT EXISTS public.compliance_rule_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    rule_code VARCHAR(100) NOT NULL,
    rule_name TEXT NOT NULL,
    rule_category VARCHAR(50) NOT NULL,    -- CONCENTRATION, ALLOCATION, RATING_MATRIX, BENCHMARK_VARIANCE
    severity VARCHAR(20) NOT NULL,         -- HARD_BLOCK, SOFT_WARNING
    numerator_ast TEXT NOT NULL,           -- e.g., "PositionOrderMarketValue"
    denominator_type VARCHAR(50) NOT NULL, -- ACCT_MARKET_VALUE, TOTAL_AUM, BENCHMARK_WEIGHT
    grouping_dimension VARCHAR(100),       -- bloomberg_industry_sector, country_of_risk, issuer_id
    threshold_operator VARCHAR(10) NOT NULL, -- >, <, >=, <=, BETWEEN
    threshold_val NUMERIC(18, 6) NOT NULL,
    tolerance_band NUMERIC(18, 6) DEFAULT 0.0,
    is_lookthrough_enabled BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_compliance_rule UNIQUE (tenant_id, rule_code)
);

-- 3. Look-Through Fund Constituents Store
CREATE TABLE IF NOT EXISTS public.compliance_lookthrough_constituents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    fund_security_id VARCHAR(50) NOT NULL,
    constituent_security_id VARCHAR(50) NOT NULL,
    weight_pct NUMERIC(8, 6) NOT NULL,     -- e.g. 0.045000 for 4.5%
    as_of_date DATE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_fund_constituent UNIQUE (tenant_id, fund_security_id, constituent_security_id, as_of_date)
);

-- 4. Pre-Trade Compliance Audit & Breach Ledger
CREATE TABLE IF NOT EXISTS public.compliance_pretrade_audit_ledger (
    audit_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    trade_ticket_id VARCHAR(100) NOT NULL,
    account_id VARCHAR(50) NOT NULL,
    security_id VARCHAR(50) NOT NULL,
    rule_code VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    evaluated_numerator NUMERIC(18, 4) NOT NULL,
    evaluated_denominator NUMERIC(18, 4) NOT NULL,
    evaluated_ratio NUMERIC(18, 6) NOT NULL,
    threshold_limit NUMERIC(18, 6) NOT NULL,
    breach_delta NUMERIC(18, 6) NOT NULL,
    compliance_status VARCHAR(50) NOT NULL, -- PASSED, HARD_BLOCKED, SOFT_WARNING_OVERRIDDEN
    execution_time_ns BIGINT NOT NULL,
    pm_override_reason TEXT,
    pm_override_actor TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_compliance_audit_search 
ON public.compliance_pretrade_audit_ledger (tenant_id, account_id, created_at DESC);
