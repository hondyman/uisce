-- 20260926_dq_health_scorecards.up.sql
-- Continuous Data Quality Scorecards & Real-Time Health Scoring (DQ-HSC)

CREATE SCHEMA IF NOT EXISTS catalog_dq;

-- 1. Declarative DQ Scorecard Rules (Rule 1: Config-Before-Code)
CREATE TABLE IF NOT EXISTS catalog_dq.scorecard_rules (
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    domain_key VARCHAR(50) NOT NULL,
    dimension VARCHAR(30) NOT NULL,
    weight_pct NUMERIC(5, 2) NOT NULL DEFAULT 25.00,
    threshold_passing_score NUMERIC(5, 2) NOT NULL DEFAULT 95.00,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_domain_dim UNIQUE (tenant_id, domain_key, dimension)
);

-- 2. Real-Time Health Score Snapshots
CREATE TABLE IF NOT EXISTS catalog_dq.health_score_snapshots (
    snapshot_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    domain_key VARCHAR(50) NOT NULL,
    asset_class VARCHAR(50) NOT NULL,
    vendor_source VARCHAR(50) NOT NULL,
    
    completeness_score NUMERIC(5, 2) NOT NULL,
    accuracy_score NUMERIC(5, 2) NOT NULL,
    timeliness_score NUMERIC(5, 2) NOT NULL,
    consistency_score NUMERIC(5, 2) NOT NULL,
    composite_health_score NUMERIC(5, 2) NOT NULL,
    
    evaluated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dq_snapshots_lookup 
ON catalog_dq.health_score_snapshots (tenant_id, domain_key, evaluated_at DESC);
