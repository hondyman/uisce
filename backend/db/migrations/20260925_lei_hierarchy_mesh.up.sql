-- 20260925_lei_hierarchy_mesh.up.sql
-- Autonomous LEI Hierarchy & Family Tree Mesh (GLEIF Integration)

CREATE SCHEMA IF NOT EXISTS catalog_lei;

-- 1. Declarative Hierarchy Rules (Rule 1: Config-Before-Code)
CREATE TABLE IF NOT EXISTS catalog_lei.hierarchy_rules (
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    hierarchy_type VARCHAR(50) NOT NULL DEFAULT 'OWNERSHIP_ULTIMATE',
    max_traversal_depth INT NOT NULL DEFAULT 10,
    confidence_match_threshold NUMERIC(5, 2) NOT NULL DEFAULT 85.00,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_hierarchy_rule UNIQUE (tenant_id, hierarchy_type)
);

-- 2. Staged GLEIF Raw Feed Records
CREATE TABLE IF NOT EXISTS catalog_lei.gleif_staged_payloads (
    stage_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    lei VARCHAR(20) NOT NULL,
    entity_name VARCHAR(255) NOT NULL,
    direct_parent_lei VARCHAR(20),
    ultimate_parent_lei VARCHAR(20),
    relationship_status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    raw_payload JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. AI Hierarchy Confidence Match Log
CREATE TABLE IF NOT EXISTS catalog_lei.hierarchy_confidence_scores (
    match_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    subsidiary_node_id UUID NOT NULL,
    suggested_parent_lei VARCHAR(20) NOT NULL,
    cosine_similarity NUMERIC(6, 4) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'SUGGESTED',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gleif_lei_lookup 
ON catalog_lei.gleif_staged_payloads (tenant_id, lei);
