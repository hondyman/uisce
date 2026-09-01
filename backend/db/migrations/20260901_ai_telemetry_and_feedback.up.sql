-- 20260901_ai_telemetry_and_feedback.up.sql
-- AI Telemetry, Sentiment Interaction Ledger, Feedback, and Term Weights

CREATE SCHEMA IF NOT EXISTS catalog_ai;

-- 1. AI Telemetry & Sentiment Interaction Ledger
CREATE TABLE IF NOT EXISTS catalog_ai.interaction_telemetry (
    interaction_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id VARCHAR(100) NOT NULL,
    session_id UUID NOT NULL,
    prompt_text TEXT NOT NULL,
    prompt_sentiment_score NUMERIC(4, 3), -- -1.000 (Frustrated) to +1.000 (Delighted)
    resolved_bo_keys TEXT[] NOT NULL DEFAULT '{}',
    resolved_term_keys TEXT[] NOT NULL DEFAULT '{}',
    execution_time_ms INT NOT NULL,
    was_anonymized BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Explicit Feedback & Weight Adjustment Store
CREATE TABLE IF NOT EXISTS catalog_ai.interaction_feedback (
    feedback_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interaction_id UUID NOT NULL REFERENCES catalog_ai.interaction_telemetry(interaction_id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    rating INT NOT NULL, -- +1 (Thumb Up), -1 (Thumb Down)
    error_category VARCHAR(50), -- HALLUCINATED_TERM, WRONG_AGGREGATION, SLOW_EXECUTION, JOIN_MISMATCH
    corrected_payload JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Dynamic Term Co-occurrence & Recommendation Weights
CREATE TABLE IF NOT EXISTS catalog_ai.term_recommendation_weights (
    source_term_node_id UUID NOT NULL,
    target_term_node_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    affinity_score NUMERIC(8, 4) NOT NULL DEFAULT 1.0000,
    co_occurrence_count BIGINT DEFAULT 1,
    last_reinforced_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (tenant_id, source_term_node_id, target_term_node_id)
);
