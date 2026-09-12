-- 20260930_110_collective_intelligence_mesh.up.sql
-- Privacy-Preserving Cross-Tenant Intelligence, Cohorts, and Sentiment Telemetry

CREATE SCHEMA IF NOT EXISTS ai_intelligence;

-- 1. Tenant Anonymized Cohort Classification
CREATE TABLE IF NOT EXISTS ai_intelligence.tenant_cohort_profile (
    tenant_id            UUID PRIMARY KEY,
    cohort_hash          VARCHAR(64) NOT NULL,
    institution_tier     VARCHAR(50) NOT NULL,
    regulatory_region    VARCHAR(50) NOT NULL,
    primary_domain       VARCHAR(50) NOT NULL,
    allow_cohort_sharing BOOLEAN NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cohort_profile_hash 
ON ai_intelligence.tenant_cohort_profile (cohort_hash, allow_cohort_sharing);

-- 2. Anonymized Cross-Tenant Query Traversal & Pattern Heuristics
CREATE TABLE IF NOT EXISTS ai_intelligence.anonymized_graph_heuristics (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cohort_hash          VARCHAR(64) NOT NULL,
    source_bo_key        VARCHAR(100) NOT NULL,
    target_field_key     VARCHAR(100) NOT NULL,
    traversal_edge_type  VARCHAR(50) NOT NULL DEFAULT 'USES_INPUT',
    co_occurrence_count  INTEGER NOT NULL DEFAULT 1,
    avg_sentiment_score  NUMERIC(4, 3) NOT NULL DEFAULT 0.000,
    success_rate         NUMERIC(4, 3) NOT NULL DEFAULT 1.000,
    differential_noise   NUMERIC(6, 4) NOT NULL DEFAULT 0.0000,
    last_updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_cohort_traversal UNIQUE (cohort_hash, source_bo_key, target_field_key, traversal_edge_type)
);

CREATE INDEX IF NOT EXISTS idx_heuristics_lookup 
ON ai_intelligence.anonymized_graph_heuristics (cohort_hash, source_bo_key, co_occurrence_count DESC);

-- 3. Granular Interaction Sentiment & Frustration Tracing (Tenant-Scoped)
CREATE TABLE IF NOT EXISTS ai_intelligence.interaction_sentiment_trace (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    session_id           UUID NOT NULL,
    user_hash            VARCHAR(64) NOT NULL,
    raw_prompt_sanitized TEXT NOT NULL,
    sentiment_polarity   NUMERIC(4, 3) NOT NULL,
    frustration_signals  JSONB NOT NULL DEFAULT '[]'::jsonb,
    query_abandoned      BOOLEAN NOT NULL DEFAULT FALSE,
    explicit_feedback    SMALLINT CHECK (explicit_feedback IN (-1, 1) OR explicit_feedback IS NULL),
    feedback_reason      VARCHAR(100),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sentiment_tenant_session 
ON ai_intelligence.interaction_sentiment_trace (tenant_id, session_id, created_at DESC);
