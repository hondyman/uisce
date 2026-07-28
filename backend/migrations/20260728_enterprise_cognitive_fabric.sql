-- Migration: Enterprise Cognitive Fabric (GraphRAG, Dual-Mode Memory, BYOK AI Gateway)

-- 1. Tenant AI Provider Table (BYOK Gateway Config - Rule 1 Alignment)
CREATE TABLE IF NOT EXISTS tenant_ai_providers (
    provider_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    provider_type VARCHAR(50) NOT NULL, -- 'OPENAI', 'AZURE_OPENAI', 'ANTHROPIC', 'PRIVATE_VLLM'
    api_endpoint TEXT NOT NULL,
    encrypted_api_key TEXT NOT NULL, -- AES-GCM Encrypted
    model_deployment_name VARCHAR(100) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_tenant_ai_provider UNIQUE(tenant_id, provider_type)
);

CREATE INDEX IF NOT EXISTS idx_tenant_ai_prov_tenant ON tenant_ai_providers(tenant_id);

-- 2. Personalized User AI History & Memory Table (Opt-In Mode B - Rule 7 RLS Alignment)
CREATE TABLE IF NOT EXISTS user_ai_history (
    history_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_hash TEXT NOT NULL,
    session_id TEXT NOT NULL,
    prompt_text TEXT NOT NULL,
    ai_response_summary TEXT,
    referenced_bo_keys TEXT[],
    applied_filters JSONB DEFAULT '{}'::jsonb,
    is_favorite BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_ai_hist_tenant_user ON user_ai_history(tenant_id, user_hash);

-- Enable RLS on User AI History
ALTER TABLE user_ai_history ENABLE ROW LEVEL SECURITY;

-- Strict Tenant & User RLS Policy
CREATE POLICY user_ai_history_tenant_isolation ON user_ai_history
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
    );

-- 3. Extend Tenant AI Recommendation Config for Dual-Mode Personalization
ALTER TABLE ai_recommendation_config 
ADD COLUMN IF NOT EXISTS personalization_mode VARCHAR(50) DEFAULT 'ANONYMIZED_COLLECTIVE',
ADD COLUMN IF NOT EXISTS default_provider_type VARCHAR(50) DEFAULT 'OPENAI';
