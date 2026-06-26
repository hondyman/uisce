-- Secrets Management Schema
-- Enterprise-grade secrets metadata for Vault/AWS/Azure integration

-- Core secrets metadata (low/no-code UI)
CREATE TABLE secret_metadata (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL,                  -- e.g., "secret/invest/prod-db"
    secret_type TEXT DEFAULT 'kv-v2',    -- kv-v2 | aws | azure | database
    description TEXT,
    ttl INTERVAL,                        -- Temporal rotation interval
    max_versions INT DEFAULT 10,
    tags TEXT[],
    attributes JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, path)
);

-- Link secrets to ABAC policies
CREATE TABLE secret_policy (
    secret_id UUID REFERENCES secret_metadata(id) ON DELETE CASCADE,
    policy_id UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (secret_id, policy_id)
);

-- Version history for rotation tracking
CREATE TABLE secret_version (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    secret_id UUID REFERENCES secret_metadata(id) ON DELETE CASCADE,
    version INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    created_by UUID,
    destroy_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}'
);

-- Audit trail for AI anomaly analysis
CREATE TABLE secret_access_log (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    secret_id UUID REFERENCES secret_metadata(id),
    user_id UUID,
    action TEXT NOT NULL,                -- "read" | "rotate" | "list" | "create" | "delete"
    ip_address INET,
    user_agent TEXT,
    geolocation JSONB,
    requested_at TIMESTAMPTZ DEFAULT now(),
    success BOOLEAN DEFAULT true,
    error_message TEXT,
    abac_result JSONB
);

-- Indexes for performance
CREATE INDEX idx_secret_metadata_tenant ON secret_metadata(tenant_id);
CREATE INDEX idx_secret_metadata_path ON secret_metadata(path);
CREATE INDEX idx_secret_metadata_tags ON secret_metadata USING GIN(tags);
CREATE INDEX idx_secret_access_log_time ON secret_access_log(requested_at DESC);
CREATE INDEX idx_secret_access_log_user ON secret_access_log(user_id);
CREATE INDEX idx_secret_access_log_secret ON secret_access_log(secret_id);
CREATE INDEX idx_secret_version_secret ON secret_version(secret_id, version DESC);

-- View for secrets needing rotation (used by Temporal workflow)
CREATE OR REPLACE VIEW secrets_needing_rotation AS
SELECT *
FROM secret_metadata
WHERE ttl IS NOT NULL
  AND (updated_at + ttl) < now()
  AND deleted_at IS NULL;

COMMENT ON TABLE secret_metadata IS 'Core secrets metadata for low/no-code UI - stores path, type, and config for Vault/AWS/Azure';
COMMENT ON TABLE secret_access_log IS 'Audit trail for AI-powered anomaly detection in secrets access patterns';
