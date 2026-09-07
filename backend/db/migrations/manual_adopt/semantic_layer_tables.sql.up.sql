-- Migration: Create semantic layer tables
-- Description: Adds semantic_assets, relationship_suggestions, and audit tables

-- Semantic assets linking table
CREATE TABLE IF NOT EXISTS semantic_assets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  business_entity_id UUID NOT NULL,
  core_model_id UUID,
  core_view_id UUID,
  custom_model_id UUID,
  custom_view_id UUID,
  semantic_term_ids UUID[] DEFAULT '{}',
  source_tables TEXT[] DEFAULT '{}',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  UNIQUE(tenant_id, datasource_id, business_entity_id),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  FOREIGN KEY (datasource_id) REFERENCES datasources(id) ON DELETE CASCADE,
  FOREIGN KEY (core_model_id) REFERENCES catalog_node(id) ON DELETE SET NULL,
  FOREIGN KEY (core_view_id) REFERENCES catalog_node(id) ON DELETE SET NULL,
  FOREIGN KEY (custom_model_id) REFERENCES catalog_node(id) ON DELETE SET NULL,
  FOREIGN KEY (custom_view_id) REFERENCES catalog_node(id) ON DELETE SET NULL
);

-- Relationship suggestions table
CREATE TABLE IF NOT EXISTS relationship_suggestions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  source_entity_id UUID NOT NULL,
  target_entity_id UUID NOT NULL,
  confidence FLOAT NOT NULL CHECK (confidence BETWEEN 0 AND 1),
  rationale TEXT,
  scoring_breakdown JSONB,
  accepted BOOLEAN DEFAULT FALSE,
  accepted_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  UNIQUE(tenant_id, datasource_id, source_entity_id, target_entity_id),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  FOREIGN KEY (datasource_id) REFERENCES datasources(id) ON DELETE CASCADE,
  FOREIGN KEY (source_entity_id) REFERENCES catalog_node(id) ON DELETE CASCADE,
  FOREIGN KEY (target_entity_id) REFERENCES catalog_node(id) ON DELETE CASCADE
);

-- Audit trail for suggestions
CREATE TABLE IF NOT EXISTS relationship_suggestion_audit (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  suggestion_id UUID NOT NULL,
  action VARCHAR(50) NOT NULL,
  user_id UUID,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  FOREIGN KEY (suggestion_id) REFERENCES relationship_suggestions(id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_semantic_assets_business_entity 
  ON semantic_assets(business_entity_id);
CREATE INDEX IF NOT EXISTS idx_semantic_assets_tenant_datasource 
  ON semantic_assets(tenant_id, datasource_id);
CREATE INDEX IF NOT EXISTS idx_relationship_suggestions_source 
  ON relationship_suggestions(source_entity_id);
CREATE INDEX IF NOT EXISTS idx_relationship_suggestions_target 
  ON relationship_suggestions(target_entity_id);
CREATE INDEX IF NOT EXISTS idx_relationship_suggestions_confidence 
  ON relationship_suggestions(confidence DESC);
CREATE INDEX IF NOT EXISTS idx_relationship_suggestions_accepted 
  ON relationship_suggestions(accepted);
CREATE INDEX IF NOT EXISTS idx_relationship_suggestions_tenant 
  ON relationship_suggestions(tenant_id, datasource_id);
CREATE INDEX IF NOT EXISTS idx_relationship_suggestion_audit_suggestion 
  ON relationship_suggestion_audit(suggestion_id);
