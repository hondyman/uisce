-- Migration: Add tenant QoS configuration table
-- Created: 2025-09-08
-- Description: Adds tenant_configs table for per-tenant Quality of Service settings

CREATE TABLE IF NOT EXISTS tenant_configs (
    tenant_id VARCHAR(255) PRIMARY KEY,
    tier INTEGER NOT NULL DEFAULT 0, -- 0=Bronze, 1=Silver, 2=Gold
    concurrency_limit INTEGER NOT NULL DEFAULT 10,
    token_rate INTEGER NOT NULL DEFAULT 100,
    burst_tokens INTEGER NOT NULL DEFAULT 200,
    cpu_limit DECIMAL(5,2) NOT NULL DEFAULT 10.0,
    memory_limit BIGINT NOT NULL DEFAULT 104857600, -- 100MB default
    cache_ttl INTERVAL NOT NULL DEFAULT '5 minutes',
    priority INTEGER NOT NULL DEFAULT 1,
    features JSONB NOT NULL DEFAULT '{
        "automation_auto_apply": false,
        "conversational_features": true,
        "advanced_analytics": false,
        "custom_integrations": false
    }',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- CREATE index IF NOT EXISTS for efficient tenant lookups
CREATE INDEX IF NOT EXISTS idx_tenant_configs_tenant_id ON tenant_configs(tenant_id);

-- CREATE index IF NOT EXISTS for tier-based queries
CREATE INDEX IF NOT EXISTS idx_tenant_configs_tier ON tenant_configs(tier);

-- Add trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_tenant_config_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_tenant_config_updated_at ON tenant_configs;
CREATE TRIGGER trigger_tenant_config_updated_at
    BEFORE UPDATE ON tenant_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_tenant_config_updated_at();

-- Insert default configurations for common tiers
INSERT INTO tenant_configs (tenant_id, tier, concurrency_limit, token_rate, burst_tokens, cpu_limit, memory_limit, cache_ttl, priority, features)
VALUES
    ('default_bronze', 0, 10, 100, 200, 10.0, 104857600, '5 minutes', 1,
     '{"automation_auto_apply": false, "conversational_features": true, "advanced_analytics": false, "custom_integrations": false}'),
    ('default_silver', 1, 50, 500, 1000, 25.0, 524288000, '10 minutes', 5,
     '{"automation_auto_apply": true, "conversational_features": true, "advanced_analytics": false, "custom_integrations": false}'),
    ('default_gold', 2, 100, 1000, 2000, 50.0, 1073741824, '15 minutes', 10,
     '{"automation_auto_apply": true, "conversational_features": true, "advanced_analytics": true, "custom_integrations": true}')
ON CONFLICT (tenant_id) DO NOTHING;
