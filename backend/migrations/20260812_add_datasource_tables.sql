-- Connections table - stores database connection details
CREATE TABLE IF NOT EXISTS connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    database TEXT NOT NULL,
    username TEXT NOT NULL,
    password TEXT NOT NULL,
    schema TEXT DEFAULT 'public',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Alpha datasource - master list of datasource types
CREATE TABLE IF NOT EXISTS alpha_datasource (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    datasource_code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tenant instance - links tenant to a product instance
CREATE TABLE IF NOT EXISTS tenant_instance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    datasource_id UUID NOT NULL, -- references the product/software instance
    instance_name TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tenant product - links tenant to product
CREATE TABLE IF NOT EXISTS tenant_product (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    datasource_id UUID NOT NULL,
    product_name TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tenant product datasource - actual datasource config for a tenant's product
CREATE TABLE IF NOT EXISTS tenant_product_datasource (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_product_id UUID NOT NULL REFERENCES tenant_product(id) ON DELETE CASCADE,
    alpha_datasource_id UUID NOT NULL REFERENCES alpha_datasource(id),
    connection_id UUID REFERENCES connections(id),
    source_name TEXT NOT NULL,
    config JSONB, -- fallback config if no connection_id
    is_active BOOLEAN DEFAULT true,
    last_scan_status TEXT,
    last_scan_at TIMESTAMP WITH TIME ZONE,
    last_scan_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_tenant_instance_tenant ON tenant_instance(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_product_tenant ON tenant_product(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_product_datasource_tenant_product ON tenant_product_datasource(tenant_product_id);
CREATE INDEX IF NOT EXISTS idx_tenant_product_datasource_alpha ON tenant_product_datasource(alpha_datasource_id);
