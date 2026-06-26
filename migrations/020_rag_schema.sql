-- Up Migration

-- 1. Tenant Registry
CREATE TABLE IF NOT EXISTS tenants (
    tenant_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_code VARCHAR(50) UNIQUE NOT NULL,
    tenant_name VARCHAR(255) NOT NULL,
    schema_name VARCHAR(63) UNIQUE NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb
);

-- 2. Tenant Configuration (Metadata-First)
CREATE TABLE IF NOT EXISTS tenant_configs (
    config_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(tenant_id),
    config_type VARCHAR(50) NOT NULL, -- 'rag', 'document_types', 'search', 'security'
    config_data JSONB NOT NULL,
    version INTEGER DEFAULT 1,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, config_type, version)
);

-- 3. Document Processing Jobs (Global)
CREATE TABLE IF NOT EXISTS document_jobs (
    job_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(tenant_id),
    document_id VARCHAR(255) NOT NULL,
    job_type VARCHAR(50) NOT NULL, -- 'ingest', 'reindex', 'delete'
    status VARCHAR(20) DEFAULT 'pending',
    temporal_workflow_id VARCHAR(255),
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 4. Configuration Deployments Log
CREATE TABLE IF NOT EXISTS config_deployments (
    deployment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version_id UUID, -- Link to specific config version if needed
    tenant_id UUID REFERENCES tenants(tenant_id),
    config_type VARCHAR(50) NOT NULL,
    version INTEGER NOT NULL,
    deployed_by VARCHAR(255),
    deployed_at TIMESTAMPTZ DEFAULT NOW(),
    changelog TEXT,
    status VARCHAR(20) DEFAULT 'success'
);

-- Down Migration
-- DROP TABLE IF EXISTS config_deployments;
-- DROP TABLE IF EXISTS document_jobs;
-- DROP TABLE IF EXISTS tenant_configs;
-- DROP TABLE IF EXISTS tenants;
