-- 20260908_ast_normalization_and_mv_advisor.up.sql
-- Canonical AST Normalization, Materialized View Advisor & Role-Aware Recommendation Penalties

CREATE SCHEMA IF NOT EXISTS catalog_advisor;

-- 1. Canonical AST Signatures & Semantic Consolidation Ledger
CREATE TABLE IF NOT EXISTS catalog_advisor.canonical_ast_signatures (
    signature_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    canonical_hash VARCHAR(64) NOT NULL,
    simplified_expression TEXT NOT NULL,
    matched_existing_concept_key VARCHAR(150),
    similarity_score NUMERIC(5, 2) DEFAULT 100.00,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_canonical_hash UNIQUE (tenant_id, canonical_hash)
);

-- 2. Materialized View & Performance Recommendations
CREATE TABLE IF NOT EXISTS catalog_advisor.materialized_view_recommendations (
    recommendation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    target_backend VARCHAR(50) NOT NULL, -- STARROCKS_OLAP, SNOWFLAKE, POSTGRES_HOT
    mv_name VARCHAR(150) NOT NULL,
    recommended_ddl TEXT NOT NULL,
    query_frequency_daily BIGINT NOT NULL,
    estimated_latency_reduction_pct NUMERIC(5, 2) NOT NULL DEFAULT 85.00,
    estimated_compute_cost_savings_usd NUMERIC(10, 2) NOT NULL DEFAULT 450.00,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING_DEPLOYMENT', -- PENDING_DEPLOYMENT, DEPLOYED, DISMISSED
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Role & User Dismissal Penalty Tracking
CREATE TABLE IF NOT EXISTS catalog_advisor.dismissal_penalties (
    penalty_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_role VARCHAR(100) NOT NULL,
    source_node_id UUID NOT NULL,
    target_node_id UUID NOT NULL,
    penalty_weight NUMERIC(6, 4) NOT NULL DEFAULT 0.5000,
    dismissed_count INT DEFAULT 1,
    last_dismissed_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_dismissal_penalties_role_node UNIQUE (tenant_id, user_role, source_node_id, target_node_id)
);
