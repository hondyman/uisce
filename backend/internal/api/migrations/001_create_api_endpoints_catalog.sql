-- Create API Endpoints Catalog table
CREATE TABLE IF NOT EXISTS api_endpoints_catalog (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID,
  
  -- Core endpoint info
  endpoint_name VARCHAR(255) NOT NULL,
  description TEXT,
  http_method VARCHAR(10) NOT NULL,
  url_path VARCHAR(500) NOT NULL,
  
  -- Classification
  category VARCHAR(100) NOT NULL,
  subcategory VARCHAR(100),
  purpose VARCHAR(50), -- create, read, update, delete, execute, search, etc.
  
  -- Schema and documentation
  request_schema JSONB,
  response_schema JSONB,
  parameters JSONB, -- Array of {name, in, required, description, dataType}
  examples JSONB, -- Array of example requests/responses
  tags TEXT[], -- Array of tags for grouping
  
  -- Configuration
  requires_auth BOOLEAN DEFAULT true,
  is_active BOOLEAN DEFAULT true,
  version VARCHAR(50) DEFAULT '1.0.0',
  
  -- Audit
  created_by UUID,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  
  CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Create indexes for common queries
CREATE INDEX idx_api_endpoints_tenant_id ON api_endpoints_catalog(tenant_id);
CREATE INDEX idx_api_endpoints_datasource_id ON api_endpoints_catalog(datasource_id);
CREATE INDEX idx_api_endpoints_category ON api_endpoints_catalog(category);
CREATE INDEX idx_api_endpoints_http_method ON api_endpoints_catalog(http_method);
CREATE INDEX idx_api_endpoints_is_active ON api_endpoints_catalog(is_active);
CREATE INDEX idx_api_endpoints_created_at ON api_endpoints_catalog(created_at DESC);

-- Create trigger for updated_at
CREATE OR REPLACE FUNCTION update_api_endpoints_catalog_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = CURRENT_TIMESTAMP;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER api_endpoints_catalog_update_trigger
BEFORE UPDATE ON api_endpoints_catalog
FOR EACH ROW
EXECUTE FUNCTION update_api_endpoints_catalog_updated_at();

-- Create table for API endpoint relationships to entities
CREATE TABLE IF NOT EXISTS api_endpoint_entity_mappings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  api_endpoint_id UUID NOT NULL REFERENCES api_endpoints_catalog(id) ON DELETE CASCADE,
  entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
  tenant_id UUID NOT NULL,
  
  -- Relationship type
  relationship_type VARCHAR(50), -- can_read, can_create, can_update, can_delete, can_execute, etc.
  
  -- Audit
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  
  UNIQUE(api_endpoint_id, entity_id, tenant_id, relationship_type),
  CONSTRAINT fk_tenant_mapping FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_endpoint_entity_mappings_endpoint ON api_endpoint_entity_mappings(api_endpoint_id);
CREATE INDEX idx_endpoint_entity_mappings_entity ON api_endpoint_entity_mappings(entity_id);
CREATE INDEX idx_endpoint_entity_mappings_tenant ON api_endpoint_entity_mappings(tenant_id);

-- Create trigger for endpoint entity mappings updated_at
CREATE TRIGGER api_endpoint_entity_mappings_update_trigger
BEFORE UPDATE ON api_endpoint_entity_mappings
FOR EACH ROW
EXECUTE FUNCTION update_api_endpoints_catalog_updated_at();

-- Create table for API endpoint relationships to datasources
CREATE TABLE IF NOT EXISTS api_endpoint_datasource_mappings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  api_endpoint_id UUID NOT NULL REFERENCES api_endpoints_catalog(id) ON DELETE CASCADE,
  datasource_id UUID NOT NULL REFERENCES datasources(id) ON DELETE CASCADE,
  tenant_id UUID NOT NULL,
  
  -- Relationship type
  relationship_type VARCHAR(50), -- can_read, can_write, can_validate, can_sync, etc.
  
  -- Audit
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  
  UNIQUE(api_endpoint_id, datasource_id, tenant_id, relationship_type),
  CONSTRAINT fk_tenant_datasource_mapping FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_endpoint_datasource_mappings_endpoint ON api_endpoint_datasource_mappings(api_endpoint_id);
CREATE INDEX idx_endpoint_datasource_mappings_datasource ON api_endpoint_datasource_mappings(datasource_id);
CREATE INDEX idx_endpoint_datasource_mappings_tenant ON api_endpoint_datasource_mappings(tenant_id);

-- Create trigger for endpoint datasource mappings updated_at
CREATE TRIGGER api_endpoint_datasource_mappings_update_trigger
BEFORE UPDATE ON api_endpoint_datasource_mappings
FOR EACH ROW
EXECUTE FUNCTION update_api_endpoints_catalog_updated_at();
