-- 20260921_mdm_exception_studio.up.sql
-- MDM Exception Queue, Variance Deltas & AI Triage Proposals

CREATE SCHEMA IF NOT EXISTS catalog_mdm;
CREATE SCHEMA IF NOT EXISTS catalog_mdm_ai;

-- 1. Ensure Table Structure for Universal Exception Queue
CREATE TABLE IF NOT EXISTS catalog_mdm.universal_exception_queue (
    exception_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    domain_key VARCHAR(50) NOT NULL,
    golden_id UUID,
    master_entity_sid VARCHAR(100) NOT NULL,
    field_name VARCHAR(100) NOT NULL,
    competing_values JSONB NOT NULL,
    deviation_pct NUMERIC(8, 4) NOT NULL DEFAULT 0.0000,
    status VARCHAR(30) NOT NULL DEFAULT 'OPEN', -- OPEN, PENDING_APPROVAL, RESOLVED_AUTO, OVERRIDDEN_MANUAL, DISMISSED
    created_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

-- 2. Ensure AI Agentic Triage Proposals Table
CREATE TABLE IF NOT EXISTS catalog_mdm_ai.agentic_triage_proposals (
    proposal_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    exception_id UUID NOT NULL,
    master_entity_sid VARCHAR(100) NOT NULL,
    field_name VARCHAR(100) NOT NULL,
    
    winning_vendor_recommendation VARCHAR(50) NOT NULL,
    recommended_value JSONB NOT NULL,
    ai_confidence_score NUMERIC(5, 4) NOT NULL,
    explain_why_diagnostic TEXT NOT NULL,
    
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING_APPROVAL',
    approved_by VARCHAR(100),
    approved_at TIMESTAMPTZ,
    merkle_leaf_seal VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
