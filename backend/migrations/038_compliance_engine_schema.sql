-- Migration 038: Compliance Engine Schema
-- Bi-temporal, RLS-enabled, multi-tenant compliance system
-- Aligns with Whitepaper §7: Rules Engine uses Semantic Terms
-- Aligns with Usice Architecture §6.2: Multi-Tenant Enforcement Layers

CREATE SCHEMA IF NOT EXISTS edm;

-- ============================================
-- ROOT: Compliance Rule (SCD2 for versioning)
-- ============================================
CREATE TABLE IF NOT EXISTS edm.compliance_rule (
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Core Identity
    rule_code TEXT NOT NULL UNIQUE,
    rule_name TEXT NOT NULL,
    description TEXT,
    
    -- Scope (Portfolio, Strategy, Global)
    scope_type TEXT CHECK (scope_type IN ('PORTFOLIO', 'STRATEGY', 'GLOBAL')),
    scope_value TEXT,
    
    -- Rule Expression (DSL referencing semantic terms)
    expression TEXT NOT NULL,
    expression_type TEXT DEFAULT 'DSL', -- DSL, SQL, WASM
    
    -- Thresholds
    threshold_value NUMERIC(28,10),
    threshold_operator TEXT CHECK (threshold_operator IN ('<=', '>=', '<', '>', '=', '!=')),
    
    -- Severity & Status
    severity TEXT CHECK (severity IN ('HARD', 'SOFT', 'WARNING', 'ALERT')),
    status TEXT DEFAULT 'ACTIVE', -- ACTIVE, INACTIVE, DRAFT, ARCHIVED
    
    -- Effective Dates (SCD2)
    effective_from DATE NOT NULL,
    effective_to DATE,
    
    -- Bi-temporal Versioning (Semantic Design §6.3)
    valid_from TIMESTAMPTZ DEFAULT NOW(),
    valid_to TIMESTAMPTZ DEFAULT 'infinity',
    system_from TIMESTAMPTZ DEFAULT NOW(),
    system_to TIMESTAMPTZ DEFAULT 'infinity',
    
    -- Multi-tenant Lineage (Usice Architecture §6.2)
    tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    core_id UUID,
    
    -- Audit
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    UNIQUE (rule_code, valid_from)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_compliance_rule_scope ON edm.compliance_rule (scope_type, scope_value);
CREATE INDEX IF NOT EXISTS idx_compliance_rule_status ON edm.compliance_rule (status, effective_from);
CREATE INDEX IF NOT EXISTS idx_compliance_rule_tenant ON edm.compliance_rule (tenant_id);
CREATE INDEX IF NOT EXISTS idx_compliance_rule_valid ON edm.compliance_rule (valid_from, valid_to) WHERE valid_to = 'infinity';

-- RLS Policies
ALTER TABLE edm.compliance_rule ENABLE ROW LEVEL SECURITY;

CREATE POLICY compliance_rule_tenant_isolation ON edm.compliance_rule
    FOR ALL USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);

-- ============================================
-- EVALUATION: Compliance Evaluation Results
-- ============================================
CREATE TABLE IF NOT EXISTS edm.compliance_evaluation (
    evaluation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Keys
    rule_id UUID NOT NULL REFERENCES edm.compliance_rule(rule_id),
    portfolio_id UUID NOT NULL,
    valuation_date DATE NOT NULL,
    
    -- Results
    metric_value NUMERIC(28,10),
    threshold_value NUMERIC(28,10),
    result TEXT CHECK (result IN ('PASS', 'FAIL', 'WARNING')),
    details JSONB,
    
    -- Performance
    evaluation_time_ms INTEGER,
    
    -- Audit
    evaluated_at TIMESTAMPTZ DEFAULT NOW(),
    tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    
    UNIQUE (rule_id, portfolio_id, valuation_date)
);

CREATE INDEX IF NOT EXISTS idx_compliance_eval_portfolio ON edm.compliance_evaluation (portfolio_id, valuation_date);
CREATE INDEX IF NOT EXISTS idx_compliance_eval_result ON edm.compliance_evaluation (result, valuation_date);

-- ============================================
-- BREACH: Compliance Breach Records
-- ============================================
CREATE TABLE IF NOT EXISTS edm.compliance_breach (
    breach_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Keys
    evaluation_id UUID NOT NULL REFERENCES edm.compliance_evaluation(evaluation_id),
    rule_id UUID NOT NULL REFERENCES edm.compliance_rule(rule_id),
    portfolio_id UUID NOT NULL,
    valuation_date DATE NOT NULL,
    
    -- Breach Details
    severity TEXT NOT NULL,
    metric_value NUMERIC(28,10),
    threshold_value NUMERIC(28,10),
    deviation NUMERIC(28,10),
    message TEXT,
    
    -- Lifecycle
    status TEXT DEFAULT 'OPEN', -- OPEN, ACKNOWLEDGED, RESOLVED, WAIVED
    priority TEXT DEFAULT 'HIGH', -- HIGH, MEDIUM, LOW
    assigned_to UUID,
    
    -- Resolution
    resolved_at TIMESTAMPTZ,
    resolved_by UUID,
    resolution_notes TEXT,
    waiver_expiry DATE,
    
    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    
    UNIQUE (rule_id, portfolio_id, valuation_date)
);

CREATE INDEX IF NOT EXISTS idx_compliance_breach_portfolio ON edm.compliance_breach (portfolio_id, valuation_date);
CREATE INDEX IF NOT EXISTS idx_compliance_breach_status ON edm.compliance_breach (status, priority);
CREATE INDEX IF NOT EXISTS idx_compliance_breach_tenant ON edm.compliance_breach (tenant_id);

-- RLS
ALTER TABLE edm.compliance_breach ENABLE ROW LEVEL SECURITY;
CREATE POLICY compliance_breach_tenant_isolation ON edm.compliance_breach
    FOR ALL USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);

-- ============================================
-- TRACE: Compliance Evaluation Lineage (Whitepaper §9)
-- ============================================
CREATE TABLE IF NOT EXISTS edm.compliance_lineage (
    lineage_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    evaluation_id UUID NOT NULL REFERENCES edm.compliance_evaluation(evaluation_id),
    source_domain TEXT NOT NULL, -- POSITION, CASH, SECURITY, PORTFOLIO
    source_table TEXT NOT NULL,
    source_record_id UUID,
    contribution_type TEXT,
    contribution_amount NUMERIC(28,10),
    processed_at TIMESTAMPTZ DEFAULT NOW(),
    tenant_id UUID NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_compliance_lineage_eval ON edm.compliance_lineage (evaluation_id);
