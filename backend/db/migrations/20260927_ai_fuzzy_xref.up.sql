-- 20260927_ai_fuzzy_xref.up.sql
-- Generative Entity Resolution & Fuzzy Cross-Matching (AI-Powered XREF)

CREATE SCHEMA IF NOT EXISTS catalog_mdm_ai;

-- 1. Entity Matching Configuration Rules (Rule 1: Config-Before-Code)
CREATE TABLE IF NOT EXISTS catalog_mdm_ai.entity_matching_rules (
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    domain_key VARCHAR(50) NOT NULL, -- SECURITY, PARTY, PRICING
    auto_merge_threshold NUMERIC(5, 4) NOT NULL DEFAULT 0.9500,
    review_threshold NUMERIC(5, 4) NOT NULL DEFAULT 0.8200,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_domain_match_rule UNIQUE (tenant_id, domain_key)
);

-- 2. Semantic Entity Embeddings (pgvector HNSW)
CREATE TABLE IF NOT EXISTS catalog_mdm_ai.entity_semantic_embeddings (
    embedding_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    golden_id UUID NOT NULL,
    domain_key VARCHAR(50) NOT NULL,
    entity_text_context TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_golden_embedding UNIQUE (tenant_id, golden_id)
);

-- 3. Staged Fuzzy Match Proposals
CREATE TABLE IF NOT EXISTS catalog_mdm_ai.fuzzy_match_proposals (
    proposal_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    inbound_payload_json JSONB NOT NULL,
    matched_golden_id UUID,
    cosine_similarity NUMERIC(6, 4) NOT NULL,
    match_status VARCHAR(30) NOT NULL DEFAULT 'PENDING_REVIEW',
    rationale_narrative TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fuzzy_proposals_status 
ON catalog_mdm_ai.fuzzy_match_proposals (tenant_id, match_status, cosine_similarity DESC);
