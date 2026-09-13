-- 20260907_unsupervised_mining_and_recommendations.up.sql
-- Unsupervised Query Telemetry Mining & Autonomous Recommendation Engine

CREATE SCHEMA IF NOT EXISTS catalog_mining;
CREATE SCHEMA IF NOT EXISTS catalog_rec;

-- 1. Ingested Query Signatures & Abstracted AST Patterns
CREATE TABLE IF NOT EXISTS catalog_mining.discovered_patterns (
    pattern_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    pattern_type VARCHAR(50) NOT NULL, -- IMPLICIT_JOIN, METRIC_FORMULA, TEMPORAL_CONVENTION
    pattern_signature VARCHAR(64) NOT NULL,
    source_dialect VARCHAR(30) NOT NULL, -- SNOWFLAKE, STARROCKS, POSTGRES
    raw_expression TEXT NOT NULL,
    normalized_ast JSONB NOT NULL,
    execution_frequency BIGINT DEFAULT 1,
    unique_user_count INT DEFAULT 1,
    first_detected_at TIMESTAMPTZ DEFAULT NOW(),
    last_detected_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_pattern UNIQUE (tenant_id, pattern_signature)
);

-- 2. Staged OKF Candidate Proposals for Steward Signoff
CREATE TABLE IF NOT EXISTS catalog_mining.candidate_proposals (
    proposal_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    pattern_id UUID NOT NULL REFERENCES catalog_mining.discovered_patterns(pattern_id) ON DELETE CASCADE,
    proposed_concept_type VARCHAR(50) NOT NULL, -- concept/attested-calculation, concept/business-object-relationship
    proposed_key VARCHAR(150) NOT NULL,
    confidence_score NUMERIC(5, 2) NOT NULL, -- 0.00 to 100.00%
    staged_okf_payload JSONB NOT NULL,
    staged_markdown TEXT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING_REVIEW', -- PENDING_REVIEW, APPROVED, REJECTED
    reviewed_by VARCHAR(100),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Entity & Term Co-Occurrence Matrix (Frequent Association Graph)
CREATE TABLE IF NOT EXISTS catalog_rec.entity_cooccurrence_matrix (
    source_node_id UUID NOT NULL,
    target_node_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    association_type VARCHAR(50) NOT NULL, -- CO_QUERIED, NEXT_VISIT, CALCULATION_INPUT
    frequency_count BIGINT DEFAULT 1,
    positive_feedback_count INT DEFAULT 0,
    negative_feedback_count INT DEFAULT 0,
    confidence_weight NUMERIC(6, 4) NOT NULL DEFAULT 1.0000,
    last_reinforced_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (tenant_id, source_node_id, target_node_id, association_type)
);

-- 4. Proactive Insights & Anomaly Notifications Queue
CREATE TABLE IF NOT EXISTS catalog_rec.proactive_insights (
    insight_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    entity_node_id UUID,
    insight_type VARCHAR(50) NOT NULL, -- CONCENTRATION_SKEW, METRIC_ANOMALY, SCHEMA_DRIFT, OPTIMIZATION
    severity VARCHAR(20) NOT NULL DEFAULT 'INFO', -- INFO, WARNING, CRITICAL
    headline VARCHAR(255) NOT NULL,
    explanation_body TEXT NOT NULL,
    recommended_action_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_dismissed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ DEFAULT NOW() + INTERVAL '7 days'
);
