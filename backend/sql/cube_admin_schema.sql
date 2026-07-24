-- Cube Admin Console Schema
-- Multi-tenant organization hierarchy and premium features

-- Organizations (MSP / Platform hierarchy)
CREATE TABLE IF NOT EXISTS cube_organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL DEFAULT 'client', -- platform, msp, client
    parent_org_id UUID REFERENCES cube_organizations(id),
    settings JSONB DEFAULT '{}',
    billing_plan VARCHAR(50) DEFAULT 'starter',
    max_tenants INT DEFAULT 10,
    max_queries_per_day BIGINT DEFAULT 100000,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_cube_orgs_parent ON cube_organizations(parent_org_id);
CREATE INDEX idx_cube_orgs_type ON cube_organizations(type);
CREATE INDEX idx_cube_orgs_slug ON cube_organizations(slug);

-- Tenant Cube Configurations
CREATE TABLE IF NOT EXISTS tenant_cube_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    organization_id UUID REFERENCES cube_organizations(id),
    resource_group VARCHAR(100) DEFAULT 'tenant_standard',
    cache_tier VARCHAR(50) DEFAULT 'standard',
    refresh_mode VARCHAR(50) DEFAULT 'interval',
    refresh_cron VARCHAR(100),
    refresh_timezone VARCHAR(50) DEFAULT 'UTC',
    max_concurrent_queries INT DEFAULT 10,
    query_timeout_seconds INT DEFAULT 120,
    preagg_enabled BOOLEAN DEFAULT TRUE,
    sql_api_enabled BOOLEAN DEFAULT FALSE,
    graphql_enabled BOOLEAN DEFAULT TRUE,
    custom_schema_path VARCHAR(500),
    feature_flags JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, datasource_id)
);

CREATE INDEX idx_tenant_cube_org ON tenant_cube_configs(organization_id);
CREATE INDEX idx_tenant_cube_active ON tenant_cube_configs(is_active);

-- Cube Definitions (Semantic Catalog)
CREATE TABLE IF NOT EXISTS cube_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    data_source VARCHAR(50) DEFAULT 'starrocks',
    sql_definition TEXT NOT NULL,
    dimensions JSONB DEFAULT '[]',
    measures JSONB DEFAULT '[]',
    joins JSONB DEFAULT '[]',
    pre_aggregations JSONB DEFAULT '[]',
    refresh_key JSONB,
    is_public BOOLEAN DEFAULT FALSE,
    is_shared BOOLEAN DEFAULT FALSE,
    version INT DEFAULT 1,
    status VARCHAR(50) DEFAULT 'draft',
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, datasource_id, name, version)
);

CREATE INDEX idx_cube_def_tenant ON cube_definitions(tenant_id, datasource_id);
CREATE INDEX idx_cube_def_category ON cube_definitions(category);
CREATE INDEX idx_cube_def_shared ON cube_definitions(is_shared);
CREATE INDEX idx_cube_def_status ON cube_definitions(status);

-- Query Analytics
CREATE TABLE IF NOT EXISTS cube_query_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    query_hash VARCHAR(64) NOT NULL,
    query_text TEXT,
    cubes_used TEXT[] DEFAULT '{}',
    measures_used TEXT[] DEFAULT '{}',
    dimensions_used TEXT[] DEFAULT '{}',
    filters_applied JSONB,
    preagg_used BOOLEAN DEFAULT FALSE,
    preagg_name VARCHAR(255),
    cache_hit BOOLEAN DEFAULT FALSE,
    duration_ms BIGINT NOT NULL,
    rows_returned BIGINT DEFAULT 0,
    bytes_scanned BIGINT DEFAULT 0,
    user_id VARCHAR(255),
    client_ip INET,
    user_agent TEXT,
    error_message TEXT,
    executed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_cube_analytics_tenant ON cube_query_analytics(tenant_id, executed_at DESC);
CREATE INDEX idx_cube_analytics_hash ON cube_query_analytics(query_hash, executed_at DESC);
CREATE INDEX idx_cube_analytics_time ON cube_query_analytics(executed_at DESC);
CREATE INDEX idx_cube_analytics_slow ON cube_query_analytics(duration_ms DESC) WHERE duration_ms > 1000;

-- Pre-aggregation Suggestions
CREATE TABLE IF NOT EXISTS cube_preagg_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    cube_name VARCHAR(255) NOT NULL,
    suggestion_type VARCHAR(50) DEFAULT 'auto',
    measures TEXT[] DEFAULT '{}',
    dimensions TEXT[] DEFAULT '{}',
    time_dimension VARCHAR(255),
    granularity VARCHAR(50),
    query_count BIGINT DEFAULT 0,
    avg_duration_ms BIGINT DEFAULT 0,
    estimated_savings_ms BIGINT DEFAULT 0,
    score DECIMAL(10,4) DEFAULT 0,
    yaml_definition TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    reviewed_by UUID,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_preagg_sugg_tenant ON cube_preagg_suggestions(tenant_id, status);
CREATE INDEX idx_preagg_sugg_score ON cube_preagg_suggestions(score DESC) WHERE status = 'pending';

-- Scheduled Reports
CREATE TABLE IF NOT EXISTS cube_scheduled_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    query JSONB NOT NULL,
    format VARCHAR(50) DEFAULT 'csv',
    schedule VARCHAR(100) NOT NULL,
    timezone VARCHAR(50) DEFAULT 'UTC',
    recipients TEXT[] DEFAULT '{}',
    delivery_method VARCHAR(50) DEFAULT 'email',
    s3_destination VARCHAR(500),
    webhook_url VARCHAR(500),
    is_active BOOLEAN DEFAULT TRUE,
    last_run_at TIMESTAMPTZ,
    last_run_status VARCHAR(50),
    next_run_at TIMESTAMPTZ,
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_cube_reports_tenant ON cube_scheduled_reports(tenant_id);
CREATE INDEX idx_cube_reports_next ON cube_scheduled_reports(next_run_at) WHERE is_active = TRUE;

-- Admin Users
CREATE TABLE IF NOT EXISTS cube_admin_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    organization_id UUID REFERENCES cube_organizations(id),
    role VARCHAR(50) NOT NULL DEFAULT 'tenant_viewer',
    allowed_tenants UUID[] DEFAULT '{}',
    permissions TEXT[] DEFAULT '{}',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, organization_id)
);

CREATE INDEX idx_cube_admin_org ON cube_admin_users(organization_id);
CREATE INDEX idx_cube_admin_role ON cube_admin_users(role);

-- Audit Log
CREATE TABLE IF NOT EXISTS cube_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    organization_id UUID,
    user_id UUID,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    resource_id UUID,
    changes JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_cube_audit_tenant ON cube_audit_log(tenant_id, created_at DESC);
CREATE INDEX idx_cube_audit_org ON cube_audit_log(organization_id, created_at DESC);
CREATE INDEX idx_cube_audit_action ON cube_audit_log(action, created_at DESC);

-- Insert default platform organization
INSERT INTO cube_organizations (id, name, slug, type, billing_plan, max_tenants, max_queries_per_day)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Platform Admin',
    'platform',
    'platform',
    'unlimited',
    -1,
    -1
) ON CONFLICT (slug) DO NOTHING;

-- Create materialized view for daily query stats
CREATE MATERIALIZED VIEW IF NOT EXISTS cube_daily_query_stats AS
SELECT
    tenant_id,
    DATE(executed_at) as query_date,
    COUNT(*) as total_queries,
    COUNT(*) FILTER (WHERE cache_hit) as cache_hits,
    COUNT(*) FILTER (WHERE preagg_used) as preagg_hits,
    COUNT(*) FILTER (WHERE error_message IS NOT NULL AND error_message != '') as errors,
    AVG(duration_ms)::BIGINT as avg_duration_ms,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY duration_ms)::BIGINT as p50_duration_ms,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms)::BIGINT as p95_duration_ms,
    PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY duration_ms)::BIGINT as p99_duration_ms,
    SUM(bytes_scanned) as total_bytes_scanned,
    COUNT(DISTINCT user_id) as unique_users
FROM cube_query_analytics
GROUP BY tenant_id, DATE(executed_at);

CREATE UNIQUE INDEX idx_daily_stats_pk ON cube_daily_query_stats(tenant_id, query_date);

-- Function to refresh daily stats
CREATE OR REPLACE FUNCTION refresh_cube_daily_stats()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY cube_daily_query_stats;
END;
$$ LANGUAGE plpgsql;

-- Comments
COMMENT ON TABLE cube_organizations IS 'Organization hierarchy for MSP/multi-tenant management';
COMMENT ON TABLE tenant_cube_configs IS 'Per-tenant Cube.js configuration and feature flags';
COMMENT ON TABLE cube_definitions IS 'Semantic layer cube catalog with versioning';
COMMENT ON TABLE cube_query_analytics IS 'Query execution telemetry for optimization';
COMMENT ON TABLE cube_preagg_suggestions IS 'Auto-generated pre-aggregation recommendations';
COMMENT ON TABLE cube_scheduled_reports IS 'Scheduled semantic layer reports';
COMMENT ON TABLE cube_admin_users IS 'Admin console user roles and permissions';
COMMENT ON TABLE cube_audit_log IS 'Audit trail for admin actions';
