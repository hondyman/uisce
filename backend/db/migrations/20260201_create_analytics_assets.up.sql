CREATE TABLE IF NOT EXISTS analytics_assets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(255) NOT NULL,
    core_asset_id VARCHAR(255) NOT NULL, -- Logical ID of the template (e.g. 'core-trade')
    actual_asset_id VARCHAR(255) NOT NULL, -- Superset Dashboard ID
    version VARCHAR(50) NOT NULL, -- e.g. 'v1'
    is_customized BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(tenant_id, core_asset_id)
);

CREATE INDEX idx_analytics_assets_tenant ON analytics_assets(tenant_id);
