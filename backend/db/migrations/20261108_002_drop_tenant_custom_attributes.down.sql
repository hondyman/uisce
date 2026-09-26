CREATE TABLE IF NOT EXISTS public.tenant_custom_attributes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    bo_id VARCHAR(128) NOT NULL,
    attribute_name VARCHAR(100) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    data_type VARCHAR(50) NOT NULL,
    jsonb_path VARCHAR(255) NOT NULL,
    is_filterable BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    created_by VARCHAR(255) NOT NULL DEFAULT 'system',
    CONSTRAINT uk_tenant_custom_attr UNIQUE (tenant_id, bo_id, attribute_name)
);

CREATE INDEX IF NOT EXISTS idx_custom_attrs_tenant_bo
    ON public.tenant_custom_attributes(tenant_id, bo_id);
