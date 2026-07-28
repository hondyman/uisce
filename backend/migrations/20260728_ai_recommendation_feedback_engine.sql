-- Migration: AI Recommendation & Feedback Engine
-- Enables vector search, interaction telemetry, sentiment tracking, and closed-loop personalization

CREATE EXTENSION IF NOT EXISTS vector;

-- 1. AI Telemetry & Interaction Log Table
CREATE TABLE IF NOT EXISTS ai_interaction_log (
    interaction_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    session_hash TEXT NOT NULL,
    user_role TEXT NOT NULL DEFAULT 'analyst',
    prompt_raw_scrubbed TEXT NOT NULL,
    response_summary TEXT,
    sentiment_score DOUBLE PRECISION DEFAULT 0.0,
    intent_category TEXT,
    referenced_bo_keys TEXT[],
    token_usage_prompt INT DEFAULT 0,
    token_usage_completion INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_log_tenant ON ai_interaction_log(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ai_log_created_at ON ai_interaction_log(created_at DESC);

-- 2. AI Interaction Vector Embeddings Table (pgvector)
CREATE TABLE IF NOT EXISTS ai_interaction_embeddings (
    interaction_id UUID PRIMARY KEY REFERENCES ai_interaction_log(interaction_id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    session_hash TEXT NOT NULL,
    prompt_text TEXT NOT NULL,
    embedding VECTOR(1536), -- Vector representation of prompt/context
    referenced_bo_keys TEXT[],
    sentiment_score DOUBLE PRECISION DEFAULT 0.0,
    intent_category TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_embeddings_tenant ON ai_interaction_embeddings(tenant_id);
-- HNSW vector similarity search index
CREATE INDEX IF NOT EXISTS idx_ai_embeddings_vector ON ai_interaction_embeddings USING hnsw (embedding vector_cosine_ops);

-- 3. AI Config Table (Rule 1 Alignment: Metadata-driven weights & thresholds)
CREATE TABLE IF NOT EXISTS ai_recommendation_config (
    config_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    sentiment_alert_threshold DOUBLE PRECISION DEFAULT -0.3,
    vector_similarity_threshold DOUBLE PRECISION DEFAULT 0.75,
    graph_traversal_depth INT DEFAULT 2,
    decay_factor DOUBLE PRECISION DEFAULT 0.95,
    max_recommendations INT DEFAULT 3,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_tenant_ai_config UNIQUE(tenant_id)
);

-- Seed core tenant config
INSERT INTO ai_recommendation_config (tenant_id, sentiment_alert_threshold, vector_similarity_threshold, graph_traversal_depth, decay_factor, max_recommendations)
VALUES ('00000000-0000-0000-0000-000000000001', -0.3, 0.75, 2, 0.95, 3)
ON CONFLICT (tenant_id) DO NOTHING;

-- 4. User Engagement & Feedback Table (Closed-Loop Personalization)
CREATE TABLE IF NOT EXISTS user_behavior_stats (
    stat_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    session_hash TEXT NOT NULL,
    bo_key TEXT NOT NULL,
    recommendation_label TEXT NOT NULL,
    interaction_count INT DEFAULT 1,
    positive_clicks INT DEFAULT 0,
    negative_dismissals INT DEFAULT 0,
    weight_score DOUBLE PRECISION DEFAULT 1.0,
    last_interaction_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_user_bo_rec UNIQUE(tenant_id, session_hash, bo_key, recommendation_label)
);

CREATE INDEX IF NOT EXISTS idx_user_behavior_tenant ON user_behavior_stats(tenant_id, session_hash);
