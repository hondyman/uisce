-- Migration: 001_initial_schema.sql
-- Core tenant and product schema with RLS (Row Level Security)
-- This replaces the Hasura-managed schema

-- ============================================================================
-- Enable UUID generation
-- ============================================================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- Tenants table (if not exists)
-- ============================================================================
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);

-- ============================================================================
-- Update trigger function for timestamps
-- ============================================================================
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Tenant-scoped tables with RLS
-- ============================================================================

-- ============================================================================
-- Product (alpha_product)
-- ============================================================================
CREATE TABLE IF NOT EXISTS alpha_product (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_name VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    product_code VARCHAR(255),
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TRIGGER update_alpha_product BEFORE UPDATE ON alpha_product
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE INDEX IF NOT EXISTS idx_alpha_product_active ON alpha_product(is_active);
CREATE INDEX IF NOT EXISTS idx_alpha_product_status ON alpha_product(status);

-- Product alias
CREATE TABLE IF NOT EXISTS product (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_name VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    product_code VARCHAR(255),
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TRIGGER update_product BEFORE UPDATE ON product
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- ============================================================================
-- Datasource
-- ============================================================================
CREATE TABLE IF NOT EXISTS alpha_datasource (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    datasource_name VARCHAR NOT NULL,
    datasource_code VARCHAR NOT NULL UNIQUE,
    is_active BOOLEAN DEFAULT true,
    config JSONB,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    datasource_type VARCHAR
);

CREATE TRIGGER update_alpha_datasource BEFORE UPDATE ON alpha_datasource
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE INDEX IF NOT EXISTS idx_alpha_datasource_active ON alpha_datasource(is_active);

-- ============================================================================
-- Tenant Instance
-- ============================================================================
CREATE TABLE IF NOT EXISTS tenant_instance (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(255) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    instance_name VARCHAR(255) NOT NULL,
    config JSONB NOT NULL,
    is_active BOOLEAN DEFAULT true,
    url VARCHAR(255),
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, instance_name),
    UNIQUE(url)
);

CREATE TRIGGER update_tenant_instance BEFORE UPDATE ON tenant_instance
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE INDEX IF NOT EXISTS idx_tenant_instance_active ON tenant_instance(is_active);
CREATE INDEX IF NOT EXISTS idx_tenant_instance_tenant_id ON tenant_instance(tenant_id);

-- ============================================================================
-- Tenant Product (join table)
-- ============================================================================
CREATE TABLE IF NOT EXISTS tenant_product (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_instance_id UUID NOT NULL REFERENCES tenant_instance(id) ON DELETE CASCADE,
    alpha_product_id UUID NOT NULL REFERENCES alpha_product(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    version FLOAT4 NOT NULL,
    is_active BOOLEAN DEFAULT false,
    UNIQUE(tenant_instance_id, alpha_product_id)
);

CREATE TRIGGER update_tenant_product BEFORE UPDATE ON tenant_product
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE INDEX IF NOT EXISTS idx_tenant_product_active ON tenant_product(is_active);

-- ============================================================================
-- Tenant Product Datasource
-- ============================================================================
CREATE TABLE IF NOT EXISTS tenant_product_datasource (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_product_id UUID NOT NULL REFERENCES tenant_product(id) ON DELETE CASCADE,
    alpha_datasource_id UUID NOT NULL REFERENCES alpha_datasource(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true,
    config JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    source_name VARCHAR,
    chart BYTEA,
    last_scan_at TIMESTAMPTZ,
    last_scan_status VARCHAR(50),
    connection_id UUID,
    UNIQUE(tenant_product_id, source_name)
);

CREATE TRIGGER update_tenant_product_datasource BEFORE UPDATE ON tenant_product_datasource
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE INDEX IF NOT EXISTS idx_tenant_product_datasource_active ON tenant_product_datasource(is_active);

-- ============================================================================
-- Connections (unified)
-- ============================================================================
CREATE TABLE IF NOT EXISTS connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    host VARCHAR(255),
    port INTEGER,
    database VARCHAR(255),
    schema VARCHAR(255),
    username VARCHAR(255),
    password VARCHAR(255),
    base_url VARCHAR(255),
    api_key VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    tenant_product_id UUID REFERENCES tenant_product(id) ON DELETE SET NULL
);

CREATE TRIGGER update_connections BEFORE UPDATE ON connections
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE INDEX IF NOT EXISTS idx_connections_tenant ON connections(tenant_id);
CREATE INDEX IF NOT EXISTS idx_connections_active ON connections(is_active);

-- Add connection FK to tenant_product_datasource if not exists
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'tenant_product_datasource_connection_fk') THEN
        ALTER TABLE tenant_product_datasource
        ADD CONSTRAINT tenant_product_datasource_connection_fk
        FOREIGN KEY (connection_id) REFERENCES connections(id) ON DELETE SET NULL ON UPDATE CASCADE;
    END IF;
END $$;

-- ============================================================================
-- Fabric Definition
-- ============================================================================
CREATE TABLE IF NOT EXISTS fabric_defn (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    tenant_datasource_id UUID REFERENCES tenant_product_datasource(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    definition JSONB NOT NULL,
    version INT DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TRIGGER update_fabric_defn BEFORE UPDATE ON fabric_defn
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE INDEX IF NOT EXISTS idx_fabric_defn_tenant ON fabric_defn(tenant_id);

-- Add FK constraints if not exists
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_fabric_defn_tenant') THEN
        ALTER TABLE fabric_defn
        ADD CONSTRAINT fk_fabric_defn_tenant
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_fabric_defn_tenant_datasource') THEN
        ALTER TABLE fabric_defn
        ADD CONSTRAINT fk_fabric_defn_tenant_datasource
        FOREIGN KEY (tenant_datasource_id) REFERENCES tenant_product_datasource(id) ON DELETE CASCADE;
    END IF;
END $$;

-- ============================================================================
-- Enable Row Level Security (RLS)
-- ============================================================================

-- Enable RLS on tenant-scoped tables
ALTER TABLE tenant_instance ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_product ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_product_datasource ENABLE ROW LEVEL SECURITY;
ALTER TABLE connections ENABLE ROW LEVEL SECURITY;
ALTER TABLE fabric_defn ENABLE ROW LEVEL SECURITY;

-- RLS Policies for tenant_instance
CREATE POLICY tenant_isolation_tenant_instance ON tenant_instance
    FOR ALL USING (tenant_id = current_setting('app.current_tenant', true));

-- RLS Policies for tenant_product (via tenant_instance)
CREATE POLICY tenant_isolation_tenant_product ON tenant_product
    FOR ALL USING (
        tenant_instance_id IN (
            SELECT id FROM tenant_instance WHERE tenant_id = current_setting('app.current_tenant', true)::UUID
        )
    );

-- RLS Policies for tenant_product_datasource (via tenant_product -> tenant_instance)
CREATE POLICY tenant_isolation_tenant_product_datasource ON tenant_product_datasource
    FOR ALL USING (
        tenant_product_id IN (
            SELECT tp.id FROM tenant_product tp
            JOIN tenant_instance ti ON tp.tenant_instance_id = ti.id
            WHERE ti.tenant_id = current_setting('app.current_tenant', true)::UUID
        )
    );

-- RLS Policies for connections
CREATE POLICY tenant_isolation_connections ON connections
    FOR ALL USING (tenant_id = current_setting('app.current_tenant', true)::UUID);

-- RLS Policies for fabric_defn
CREATE POLICY tenant_isolation_fabric_defn ON fabric_defn
    FOR ALL USING (tenant_id = current_setting('app.current_tenant', true)::UUID);
