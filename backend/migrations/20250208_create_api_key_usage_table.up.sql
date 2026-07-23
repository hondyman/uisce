-- Create API key usage logging table
CREATE TABLE IF NOT EXISTS api_key_usage (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id  UUID NOT NULL,
    user_id     UUID,
    tenant_id   UUID,
    path        TEXT NOT NULL,
    method      TEXT NOT NULL,
    region      TEXT,
    ip_address  INET,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_api_key_usage_api_key_id ON api_key_usage(api_key_id);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_tenant_id ON api_key_usage(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_created_at ON api_key_usage(created_at);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_path ON api_key_usage(path);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_method ON api_key_usage(method);
