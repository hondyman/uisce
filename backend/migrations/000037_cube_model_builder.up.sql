-- Migration: Cube Model Builder with RBAC/ABAC Security
-- Description: Creates tables for catalog-integrated Cube model management with security policies

-- Core Cube Models (generated from metadata catalog)
CREATE TABLE IF NOT EXISTS cube_core_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    datasource_id UUID NOT NULL REFERENCES tenant_product_datasource(id) ON DELETE CASCADE,
    catalog_node_id UUID REFERENCES catalog_node(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    sql_table VARCHAR(500),
    data_source VARCHAR(255) DEFAULT 'default',
    description TEXT,
    generated_yaml TEXT,
    measures JSONB DEFAULT '[]'::jsonb,
    dimensions JSONB DEFAULT '[]'::jsonb,
    joins JSONB DEFAULT '[]'::jsonb,
    sync_status VARCHAR(50) DEFAULT 'pending',
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, datasource_id, name)
);

-- Custom Cube Model Extensions (user-defined customizations)
CREATE TABLE IF NOT EXISTS cube_custom_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    datasource_id UUID NOT NULL REFERENCES tenant_product_datasource(id) ON DELETE CASCADE,
    core_model_id UUID REFERENCES cube_core_models(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    extension_type VARCHAR(50) NOT NULL CHECK (extension_type IN ('extend', 'override', 'standalone')),
    custom_config JSONB DEFAULT '{}'::jsonb,
    version INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, datasource_id, name)
);

-- Custom model version history for audit trail
CREATE TABLE IF NOT EXISTS cube_custom_model_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    custom_model_id UUID NOT NULL REFERENCES cube_custom_models(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    custom_config JSONB NOT NULL,
    change_description TEXT,
    changed_by UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(custom_model_id, version)
);

-- ABAC Security Policies for Cube queries
CREATE TABLE IF NOT EXISTS cube_security_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    policy_type VARCHAR(50) NOT NULL CHECK (policy_type IN ('row', 'column', 'access', 'query')),
    priority INTEGER DEFAULT 100,
    enabled BOOLEAN DEFAULT true,
    target_cubes JSONB DEFAULT '[]'::jsonb,  -- Empty = all cubes
    target_members JSONB DEFAULT '[]'::jsonb, -- Empty = all members
    conditions JSONB NOT NULL DEFAULT '{}'::jsonb,
    effects JSONB NOT NULL DEFAULT '{}'::jsonb,
    version INTEGER DEFAULT 1,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

-- Policy version history
CREATE TABLE IF NOT EXISTS cube_security_policy_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID NOT NULL REFERENCES cube_security_policies(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    conditions JSONB NOT NULL,
    effects JSONB NOT NULL,
    change_description TEXT,
    changed_by UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(policy_id, version)
);

-- Security Decision Cache (persistent cache for complex policy evaluations)
CREATE TABLE IF NOT EXISTS cube_security_cache (
    cache_key VARCHAR(64) PRIMARY KEY,
    tenant_id UUID NOT NULL,
    decision JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    hit_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_hit_at TIMESTAMPTZ
);

-- Model Builder Wizard Sessions
CREATE TABLE IF NOT EXISTS cube_model_builder_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    datasource_id UUID NOT NULL REFERENCES tenant_product_datasource(id) ON DELETE CASCADE,
    session_type VARCHAR(50) NOT NULL DEFAULT 'custom',
    current_step INTEGER DEFAULT 1,
    total_steps INTEGER DEFAULT 6,
    session_data JSONB DEFAULT '{}'::jsonb,
    status VARCHAR(50) DEFAULT 'in_progress',
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    result_model_id UUID REFERENCES cube_custom_models(id) ON DELETE SET NULL
);

-- Wizard Step Data (persisted for session recovery)
CREATE TABLE IF NOT EXISTS cube_model_builder_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES cube_model_builder_sessions(id) ON DELETE CASCADE,
    step_number INTEGER NOT NULL,
    step_type VARCHAR(100) NOT NULL,
    step_data JSONB DEFAULT '{}'::jsonb,
    completed BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(session_id, step_number)
);

-- Pre-aggregation Configurations
CREATE TABLE IF NOT EXISTS cube_preaggregation_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    datasource_id UUID NOT NULL REFERENCES tenant_product_datasource(id) ON DELETE CASCADE,
    cube_model_id UUID REFERENCES cube_custom_models(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    preagg_type VARCHAR(50) NOT NULL DEFAULT 'rollup',
    measures JSONB DEFAULT '[]'::jsonb,
    dimensions JSONB DEFAULT '[]'::jsonb,
    time_dimension VARCHAR(255),
    granularity VARCHAR(50),
    partition_granularity VARCHAR(50),
    refresh_key TEXT,
    build_range_start TEXT,
    build_range_end TEXT,
    indexes JSONB DEFAULT '[]'::jsonb,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, datasource_id, cube_model_id, name)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_cube_core_models_tenant ON cube_core_models(tenant_id, datasource_id);
CREATE INDEX IF NOT EXISTS idx_cube_core_models_catalog ON cube_core_models(catalog_node_id);
CREATE INDEX IF NOT EXISTS idx_cube_custom_models_tenant ON cube_custom_models(tenant_id, datasource_id);
CREATE INDEX IF NOT EXISTS idx_cube_custom_models_core ON cube_custom_models(core_model_id);
CREATE INDEX IF NOT EXISTS idx_cube_security_policies_tenant ON cube_security_policies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_cube_security_policies_enabled ON cube_security_policies(tenant_id, enabled) WHERE enabled = true;
CREATE INDEX IF NOT EXISTS idx_cube_security_cache_tenant ON cube_security_cache(tenant_id);
CREATE INDEX IF NOT EXISTS idx_cube_security_cache_expiry ON cube_security_cache(expires_at);
CREATE INDEX IF NOT EXISTS idx_cube_builder_sessions_tenant ON cube_model_builder_sessions(tenant_id, datasource_id);
CREATE INDEX IF NOT EXISTS idx_cube_preagg_configs_tenant ON cube_preaggregation_configs(tenant_id, datasource_id);

-- Triggers for updated_at timestamps
CREATE OR REPLACE FUNCTION update_cube_models_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS cube_core_models_updated ON cube_core_models;
CREATE TRIGGER cube_core_models_updated
    BEFORE UPDATE ON cube_core_models
    FOR EACH ROW EXECUTE FUNCTION update_cube_models_timestamp();

DROP TRIGGER IF EXISTS cube_custom_models_updated ON cube_custom_models;
CREATE TRIGGER cube_custom_models_updated
    BEFORE UPDATE ON cube_custom_models
    FOR EACH ROW EXECUTE FUNCTION update_cube_models_timestamp();

DROP TRIGGER IF EXISTS cube_security_policies_updated ON cube_security_policies;
CREATE TRIGGER cube_security_policies_updated
    BEFORE UPDATE ON cube_security_policies
    FOR EACH ROW EXECUTE FUNCTION update_cube_models_timestamp();

DROP TRIGGER IF EXISTS cube_builder_sessions_updated ON cube_model_builder_sessions;
CREATE TRIGGER cube_builder_sessions_updated
    BEFORE UPDATE ON cube_model_builder_sessions
    FOR EACH ROW EXECUTE FUNCTION update_cube_models_timestamp();

DROP TRIGGER IF EXISTS cube_builder_steps_updated ON cube_model_builder_steps;
CREATE TRIGGER cube_builder_steps_updated
    BEFORE UPDATE ON cube_model_builder_steps
    FOR EACH ROW EXECUTE FUNCTION update_cube_models_timestamp();

-- Function to clean expired cache entries
CREATE OR REPLACE FUNCTION cube_security_cache_cleanup()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM cube_security_cache WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to version custom model changes
CREATE OR REPLACE FUNCTION cube_custom_model_version_trigger()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.custom_config IS DISTINCT FROM NEW.custom_config THEN
        INSERT INTO cube_custom_model_versions (custom_model_id, version, custom_config, changed_by)
        VALUES (NEW.id, NEW.version, NEW.custom_config, NEW.created_by);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS cube_custom_model_version ON cube_custom_models;
CREATE TRIGGER cube_custom_model_version
    AFTER UPDATE ON cube_custom_models
    FOR EACH ROW EXECUTE FUNCTION cube_custom_model_version_trigger();

-- Function to version security policy changes
CREATE OR REPLACE FUNCTION cube_security_policy_version_trigger()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.conditions IS DISTINCT FROM NEW.conditions OR OLD.effects IS DISTINCT FROM NEW.effects THEN
        INSERT INTO cube_security_policy_versions (policy_id, version, conditions, effects, changed_by)
        VALUES (NEW.id, NEW.version, NEW.conditions, NEW.effects, NEW.created_by);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS cube_security_policy_version ON cube_security_policies;
CREATE TRIGGER cube_security_policy_version
    AFTER UPDATE ON cube_security_policies
    FOR EACH ROW EXECUTE FUNCTION cube_security_policy_version_trigger();

-- Comments for documentation
COMMENT ON TABLE cube_core_models IS 'Core Cube models auto-generated from metadata catalog';
COMMENT ON TABLE cube_custom_models IS 'User-defined Cube model extensions and customizations';
COMMENT ON TABLE cube_security_policies IS 'ABAC security policies for Cube query access control';
COMMENT ON TABLE cube_security_cache IS 'Persistent cache for security decision results';
COMMENT ON TABLE cube_model_builder_sessions IS 'Wizard sessions for visual model building';
COMMENT ON TABLE cube_preaggregation_configs IS 'Pre-aggregation configurations for performance optimization';

COMMENT ON COLUMN cube_custom_models.extension_type IS 'extend: add to core model, override: replace core definitions, standalone: independent model';
COMMENT ON COLUMN cube_security_policies.policy_type IS 'row: row-level filtering, column: column masking, access: allow/deny, query: query limits';
COMMENT ON COLUMN cube_security_policies.conditions IS 'JSONB with roles, groups, attributes, time_window, ip_ranges, data_classification';
COMMENT ON COLUMN cube_security_policies.effects IS 'JSONB with action, row_filters, column_masks, query_limits, audit_log';
