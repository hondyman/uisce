-- Migration: Granular Categorical Feedback & Multi-Dimensional Personalization
-- Extends explicit user feedback with categorical diagnostic tags & multi-dimensional personalization profiles

-- 1. Explicit Feedback Diagnostic Log Table
CREATE TABLE IF NOT EXISTS ai_explicit_feedback (
    feedback_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    session_hash TEXT NOT NULL,
    interaction_id UUID REFERENCES ai_interaction_log(interaction_id) ON DELETE SET NULL,
    target_bo_key TEXT,
    rating_type VARCHAR(20) NOT NULL, -- 'THUMBS_UP', 'THUMBS_DOWN', 'RATING_STARS'
    star_score INT CHECK (star_score BETWEEN 1 AND 5),
    error_category VARCHAR(100), -- 'WRONG_TABLE', 'INCORRECT_FORMULA', 'MISSING_DATA', 'HALLUCINATED_SCHEMA', 'OTHER'
    user_comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_explicit_feedback_tenant ON ai_explicit_feedback(tenant_id, target_bo_key);
CREATE INDEX IF NOT EXISTS idx_explicit_feedback_rating ON ai_explicit_feedback(rating_type, error_category);

-- 2. Multi-Dimensional User Personalization Profile Table
CREATE TABLE IF NOT EXISTS user_personalization_profiles (
    profile_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_hash TEXT NOT NULL,
    preferred_domain VARCHAR(50) DEFAULT 'PORTFOLIO',
    preferred_currency VARCHAR(10) DEFAULT 'USD',
    preferred_dialects TEXT[] DEFAULT ARRAY['POSTGRES'], -- e.g. ['POSTGRES', 'STARROCKS', 'ICEBERG']
    frequent_bo_keys TEXT[] DEFAULT ARRAY[]::TEXT[],
    saved_filter_presets JSONB DEFAULT '{}'::jsonb,
    feedback_score_bias DOUBLE PRECISION DEFAULT 1.0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_tenant_user_profile UNIQUE(tenant_id, user_hash)
);

CREATE INDEX IF NOT EXISTS idx_user_profile_tenant ON user_personalization_profiles(tenant_id, user_hash);

-- Enable RLS for Personalization Profiles (Rule 7 Alignment)
ALTER TABLE user_personalization_profiles ENABLE ROW LEVEL SECURITY;

CREATE POLICY user_profile_tenant_isolation ON user_personalization_profiles
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
    );
