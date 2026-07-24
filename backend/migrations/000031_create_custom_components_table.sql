-- Migration: Create custom_components table
-- This table stores custom component configurations with tenant-scoped data
-- All components are associated with a specific tenant and datasource

CREATE TABLE IF NOT EXISTS custom_components (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  datasource_id UUID NOT NULL REFERENCES tenant_product_datasource(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  type VARCHAR(50) NOT NULL CHECK (type IN ('web_component', 'iframe', 'api_integration', 'custom_widget', 'chart', 'custom_code')),
  config JSONB NOT NULL DEFAULT '{}',
  events JSONB NOT NULL DEFAULT '[]',
  filters JSONB NOT NULL DEFAULT '[]',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by TEXT REFERENCES public.app_user(id) ON DELETE SET NULL,
  updated_by TEXT REFERENCES public.app_user(id) ON DELETE SET NULL,
  is_active BOOLEAN NOT NULL DEFAULT true,
  description TEXT,
  CONSTRAINT unique_component_name UNIQUE (tenant_id, datasource_id, name)
);

-- Index for efficient tenant + datasource queries (primary query pattern)
CREATE INDEX IF NOT EXISTS idx_custom_components_tenant_ds 
  ON custom_components(tenant_id, datasource_id) 
  WHERE is_active = true;

-- Index for active status filtering (soft deletes)
CREATE INDEX IF NOT EXISTS idx_custom_components_active 
  ON custom_components(is_active);

-- Index for quick lookups by component ID
CREATE INDEX IF NOT EXISTS idx_custom_components_id_active 
  ON custom_components(id) 
  WHERE is_active = true;

-- Comments for documentation
COMMENT ON TABLE custom_components IS 'Stores custom component configurations for Workday-style extensibility. All components are tenant and datasource scoped.';
COMMENT ON COLUMN custom_components.type IS 'Component type: web_component, iframe, api_integration, custom_widget, chart, or custom_code';
COMMENT ON COLUMN custom_components.config IS 'Type-specific configuration stored as JSONB. Structure varies by component type.';
COMMENT ON COLUMN custom_components.events IS 'Array of event configurations (JSONB). Defines component-to-component communication.';
COMMENT ON COLUMN custom_components.filters IS 'Array of filter configurations (JSONB). Defines cross-filtering setup.';
COMMENT ON COLUMN custom_components.is_active IS 'Soft delete flag. Set to false instead of deleting records.';
