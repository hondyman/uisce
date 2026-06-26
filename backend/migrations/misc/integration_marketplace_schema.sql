-- Integration Marketplace Schema
-- Enables discovery, installation, and management of process integrations

-- Marketplace catalog of available integrations
CREATE TABLE IF NOT EXISTS marketplace_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    integration_key VARCHAR(100) UNIQUE NOT NULL,  -- e.g., 'slack', 'teams', 'email'
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL CHECK (category IN ('communication', 'automation', 'storage', 'analytics', 'ai', 'crm', 'other')),
    provider VARCHAR(255),                          -- e.g., 'Slack Technologies', 'Microsoft'
    icon_url TEXT,
    version VARCHAR(50) DEFAULT '1.0.0',
    is_official BOOLEAN DEFAULT TRUE,              -- Official vs community integrations
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Configuration schema
    config_schema JSONB,                           -- JSON Schema for configuration form
    auth_type VARCHAR(50) CHECK (auth_type IN ('none', 'api_key', 'oauth2', 'basic_auth', 'custom')),
    oauth_config JSONB,                            -- OAuth2 configuration if applicable
    
    -- Capabilities
    supports_webhooks BOOLEAN DEFAULT FALSE,
    supports_polling BOOLEAN DEFAULT FALSE,
    supports_actions BOOLEAN DEFAULT TRUE,
    
    -- Documentation
    documentation_url TEXT,
    setup_guide TEXT,
    example_payload JSONB,
    
    -- Metadata
    install_count INT DEFAULT 0,
    rating DECIMAL(2,1) CHECK (rating >= 0 AND rating <= 5),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tenant-specific installed integrations
CREATE TABLE IF NOT EXISTS installed_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    integration_id UUID NOT NULL REFERENCES marketplace_integrations(id) ON DELETE CASCADE,
    
    -- Installation metadata
    installed_by VARCHAR(255),
    installed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_enabled BOOLEAN DEFAULT TRUE,
    
    -- Configuration (encrypted in production)
    config JSONB DEFAULT '{}',                     -- Tenant-specific settings
    credentials JSONB DEFAULT '{}',                -- API keys, OAuth tokens, etc.
    
    -- OAuth state (if applicable)
    oauth_state VARCHAR(255),
    oauth_access_token TEXT,
    oauth_refresh_token TEXT,
    oauth_expires_at TIMESTAMP WITH TIME ZONE,
    
    -- Usage stats
    last_used_at TIMESTAMP WITH TIME ZONE,
    execution_count INT DEFAULT 0,
    success_count INT DEFAULT 0,
    failure_count INT DEFAULT 0,
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(tenant_id, datasource_id, integration_id)
);

-- Integration execution logs
CREATE TABLE IF NOT EXISTS integration_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    installed_integration_id UUID NOT NULL REFERENCES installed_integrations(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    
    -- Execution context
    workflow_id UUID,
    workflow_type VARCHAR(255),
    step_name VARCHAR(255),
    action VARCHAR(100) NOT NULL,                  -- e.g., 'send_message', 'create_ticket', 'upload_file'
    
    -- Request/Response
    request_payload JSONB,
    response_payload JSONB,
    
    -- Status
    status VARCHAR(50) NOT NULL CHECK (status IN ('pending', 'success', 'failed', 'timeout', 'cancelled')),
    error_message TEXT,
    error_details JSONB,
    
    -- Timing
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INT,
    
    -- Retry info
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    next_retry_at TIMESTAMP WITH TIME ZONE
);

-- Integration-specific configurations (for complex integrations)
CREATE TABLE IF NOT EXISTS marketplace_integration_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    installed_integration_id UUID NOT NULL REFERENCES installed_integrations(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    
    setting_key VARCHAR(255) NOT NULL,
    setting_value JSONB,
    is_secret BOOLEAN DEFAULT FALSE,               -- Flag for sensitive values
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(installed_integration_id, setting_key)
);

-- Indexes for performance
CREATE INDEX idx_marketplace_integrations_category ON marketplace_integrations(category);
CREATE INDEX idx_marketplace_integrations_active ON marketplace_integrations(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_marketplace_integrations_rating ON marketplace_integrations(rating DESC);

CREATE INDEX idx_installed_integrations_tenant ON installed_integrations(tenant_id, datasource_id);
CREATE INDEX idx_installed_integrations_enabled ON installed_integrations(is_enabled) WHERE is_enabled = TRUE;
CREATE INDEX idx_installed_integrations_integration ON installed_integrations(integration_id);

CREATE INDEX idx_integration_executions_installed ON integration_executions(installed_integration_id);
CREATE INDEX idx_integration_executions_tenant ON integration_executions(tenant_id, datasource_id);
CREATE INDEX idx_integration_executions_workflow ON integration_executions(workflow_id);
CREATE INDEX idx_integration_executions_status ON integration_executions(status, started_at DESC);
CREATE INDEX idx_integration_executions_date ON integration_executions(started_at DESC);

CREATE INDEX idx_marketplace_settings_installed ON marketplace_integration_settings(installed_integration_id);

-- Triggers for auto-updating timestamps
CREATE OR REPLACE FUNCTION update_marketplace_integration_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER marketplace_integrations_updated
    BEFORE UPDATE ON marketplace_integrations
    FOR EACH ROW
    EXECUTE FUNCTION update_marketplace_integration_timestamp();

CREATE TRIGGER installed_integrations_updated
    BEFORE UPDATE ON installed_integrations
    FOR EACH ROW
    EXECUTE FUNCTION update_marketplace_integration_timestamp();

CREATE TRIGGER marketplace_settings_updated
    BEFORE UPDATE ON marketplace_integration_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_marketplace_integration_timestamp();

-- Comments for documentation
COMMENT ON TABLE marketplace_integrations IS 'Catalog of available integrations in the marketplace';
COMMENT ON TABLE installed_integrations IS 'Tenant-specific integration installations with configuration';
COMMENT ON TABLE integration_executions IS 'Execution logs for integration actions';
COMMENT ON TABLE marketplace_integration_settings IS 'Additional configuration storage for complex integrations';

COMMENT ON COLUMN marketplace_integrations.config_schema IS 'JSON Schema defining required configuration fields for the integration';
COMMENT ON COLUMN marketplace_integrations.oauth_config IS 'OAuth2 configuration including client_id, authorization_url, token_url, scopes';
COMMENT ON COLUMN installed_integrations.credentials IS 'Encrypted credentials (API keys, OAuth tokens) - should be encrypted at rest in production';
COMMENT ON COLUMN integration_executions.request_payload IS 'The data sent to the integration (sanitized for sensitive data)';
COMMENT ON COLUMN marketplace_integration_settings.is_secret IS 'If true, this value should be encrypted and never exposed in API responses';
