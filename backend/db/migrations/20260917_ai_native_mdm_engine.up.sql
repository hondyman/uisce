-- 20260917_ai_native_mdm_engine.up.sql
-- AI-Native Cognitive MDM Mesh & Neural Survivorship Ledger

CREATE SCHEMA IF NOT EXISTS catalog_mdm_ai;

-- 1. AI Confidence Scoring & Dynamic Vendor Trust Weights
CREATE TABLE IF NOT EXISTS catalog_mdm_ai.vendor_dynamic_trust_weights (
    weight_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    domain_key VARCHAR(50) NOT NULL,
    vendor_source VARCHAR(50) NOT NULL,
    asset_class VARCHAR(50) NOT NULL DEFAULT 'ALL',
    base_trust_score NUMERIC(5, 2) NOT NULL DEFAULT 85.00,
    historical_accuracy_pct NUMERIC(5, 2) NOT NULL DEFAULT 99.20,
    staleness_decay_half_life_sec INT NOT NULL DEFAULT 3600,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_vendor_domain UNIQUE (tenant_id, domain_key, vendor_source, asset_class)
);

-- 2. Semantic Cross-Reference Embeddings
CREATE TABLE IF NOT EXISTS catalog_mdm_ai.entity_semantic_embeddings (
    embedding_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    master_entity_sid VARCHAR(100) NOT NULL,
    domain_key VARCHAR(50) NOT NULL,
    entity_text_context TEXT NOT NULL,
    embedding_vector JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_entity_embedding UNIQUE (tenant_id, master_entity_sid)
);

-- 3. AI Exception Triage & Agentic Recommendations
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

CREATE INDEX IF NOT EXISTS idx_ai_triage_status 
ON catalog_mdm_ai.agentic_triage_proposals (tenant_id, status, ai_confidence_score DESC);
