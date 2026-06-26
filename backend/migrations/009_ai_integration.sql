-- backend/migrations/009_ai_integration.sql

-- AI Training Data: Historical patterns and semantic context
CREATE TABLE IF NOT EXISTS edm.ai_training_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL, -- No hard FK here to allow flexibility across systems, but RLS enforced
    source_type VARCHAR(50) NOT NULL, -- rule_patterns, drift_data, user_feedback
    input_data JSONB NOT NULL, -- Semantic context / query features
    output_data JSONB NOT NULL, -- Resulting rule/prediction
    explainability_score INT NOT NULL CHECK (explainability_score BETWEEN 0 AND 100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_ai_training_data_tenant ON edm.ai_training_data(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ai_training_data_source ON edm.ai_training_data(source_type);

-- AI Feedback: User corrections and confidence scores
CREATE TABLE IF NOT EXISTS edm.ai_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    suggestion_id UUID NOT NULL,
    confidence INT NOT NULL CHECK (confidence BETWEEN 1 AND 10),
    correction JSONB, -- User's correction of output_data
    comments TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ai_feedback_tenant ON edm.ai_feedback(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ai_feedback_suggestion ON edm.ai_feedback(suggestion_id);

-- AI Model Versions: Tracking performance and drift
CREATE TABLE IF NOT EXISTS edm.ai_model_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    model_type VARCHAR(50) NOT NULL, -- rule_suggestion, drift_prediction, semantic_chat
    version INT NOT NULL,
    performance_metrics JSONB NOT NULL,
    drift_metrics JSONB NOT NULL,
    training_data_size INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_ai_model_versions_tenant ON edm.ai_model_versions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ai_model_versions_active ON edm.ai_model_versions(model_type, is_active);

-- Row Level Security (RLS) Policies
ALTER TABLE edm.ai_training_data ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS ai_training_data_tenant_isolation ON edm.ai_training_data;
CREATE POLICY ai_training_data_tenant_isolation ON edm.ai_training_data
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

ALTER TABLE edm.ai_feedback ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS ai_feedback_tenant_isolation ON edm.ai_feedback;
CREATE POLICY ai_feedback_tenant_isolation ON edm.ai_feedback
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

ALTER TABLE edm.ai_model_versions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS ai_model_versions_tenant_isolation ON edm.ai_model_versions;
CREATE POLICY ai_model_versions_tenant_isolation ON edm.ai_model_versions
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));
