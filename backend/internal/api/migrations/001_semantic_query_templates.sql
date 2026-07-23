-- ============================================================================
-- Semantic Query Templates Schema
-- ============================================================================
-- This migration creates the complete template system with versioning, RBAC,
-- and execution tracking. All tables support multi-tenancy and include proper
-- indexing for performance.
-- ============================================================================

-- Table: semantic_query_templates
-- Main template entity with metadata, query definition, and parameters
CREATE TABLE IF NOT EXISTS semantic_query_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    
    -- Template metadata
    name VARCHAR(255) NOT NULL,
    description TEXT,
    datasource VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL DEFAULT 'v1',
    
    -- Query definition (stored as JSONB for nested structure)
    semantic_query JSONB NOT NULL,
    
    -- Parameter definitions (array of parameter objects)
    parameters JSONB NOT NULL DEFAULT '[]',
    
    -- Access control & visibility
    visibility VARCHAR(50) NOT NULL DEFAULT 'private', -- private, team, public
    tags TEXT[] DEFAULT ARRAY[]::TEXT[],
    
    -- Deprecation tracking
    deprecated BOOLEAN DEFAULT FALSE,
    deprecation_reason TEXT,
    
    -- Audit fields
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(255),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT unique_tenant_datasource_name UNIQUE (tenant_id, datasource, name),
    CONSTRAINT valid_visibility CHECK (visibility IN ('private', 'team', 'public'))
);

-- Indexes for template lookup
CREATE INDEX idx_semantic_templates_tenant_id ON semantic_query_templates(tenant_id);
CREATE INDEX idx_semantic_templates_datasource ON semantic_query_templates(datasource);
CREATE INDEX idx_semantic_templates_visibility ON semantic_query_templates(visibility);
CREATE INDEX idx_semantic_templates_created_by ON semantic_query_templates(created_by);
CREATE INDEX idx_semantic_templates_deprecated ON semantic_query_templates(deprecated);

-- Full-text search on name and description
CREATE INDEX idx_semantic_templates_search 
ON semantic_query_templates USING GIN (
    to_tsvector('english', name || ' ' || COALESCE(description, ''))
);

-- ============================================================================
-- Table: semantic_query_template_versions
-- Complete version history with change tracking
-- ============================================================================

CREATE TABLE IF NOT EXISTS semantic_query_template_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL REFERENCES semantic_query_templates(id) ON DELETE CASCADE,
    
    -- Version tracking
    version_number INTEGER NOT NULL,
    
    -- Snapshot of template at this version
    name VARCHAR(255) NOT NULL,
    description TEXT,
    semantic_query JSONB NOT NULL,
    parameters JSONB NOT NULL,
    
    -- Change tracking
    change_message TEXT,
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Promotion tracking
    is_promoted BOOLEAN DEFAULT FALSE,
    promoted_at TIMESTAMP,
    promoted_by VARCHAR(255),
    
    -- Constraints  
    CONSTRAINT unique_template_version UNIQUE (template_id, version_number)
);

-- Indexes for version history
CREATE INDEX idx_template_versions_template_id ON semantic_query_template_versions(template_id);
CREATE INDEX idx_template_versions_created_by ON semantic_query_template_versions(created_by);
CREATE INDEX idx_template_versions_promoted ON semantic_query_template_versions(is_promoted);

-- ============================================================================
-- Table: semantic_query_template_permissions
-- Role-based access control per template
-- ============================================================================

CREATE TABLE IF NOT EXISTS semantic_query_template_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL REFERENCES semantic_query_templates(id) ON DELETE CASCADE,
    
    -- Role definition (global roles)
    role VARCHAR(50) NOT NULL, -- viewer, editor, admin
    
    -- Fine-grained permissions
    can_run BOOLEAN DEFAULT FALSE,
    can_edit BOOLEAN DEFAULT FALSE,
    can_delete BOOLEAN DEFAULT FALSE,
    can_promote BOOLEAN DEFAULT FALSE,
    
    -- Constraints
    CONSTRAINT unique_template_role UNIQUE (template_id, role),
    CONSTRAINT valid_role CHECK (role IN ('viewer', 'editor', 'admin'))
);

-- Index for permission lookup
CREATE INDEX idx_template_permissions_template_id ON semantic_query_template_permissions(template_id);

-- ============================================================================
-- Table: semantic_query_template_parameter_constraints
-- Parameter-level RBAC and validation rules
-- ============================================================================

CREATE TABLE IF NOT EXISTS semantic_query_template_parameter_constraints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL REFERENCES semantic_query_templates(id) ON DELETE CASCADE,
    parameter_name VARCHAR(255) NOT NULL,
    
    -- Parameter constraints
    allowed_roles TEXT[] DEFAULT ARRAY[]::TEXT[], -- roles that can modify this param
    min_value FLOAT,
    max_value FLOAT,
    whitelisted_values JSONB, -- array of allowed values
    
    -- Masking/redaction
    is_sensitive BOOLEAN DEFAULT FALSE,
    
    -- Constraints
    CONSTRAINT unique_parameter_constraint UNIQUE (template_id, parameter_name)
);

-- Index for parameter constraint lookup
CREATE INDEX idx_parameter_constraints_template_id ON semantic_query_template_parameter_constraints(template_id);

-- ============================================================================
-- Table: semantic_query_template_executions
-- Execution history and metrics for templates
-- ============================================================================

CREATE TABLE IF NOT EXISTS semantic_query_template_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL REFERENCES semantic_query_templates(id) ON DELETE CASCADE,
    version_number INTEGER,
    
    -- Execution context
    executed_by VARCHAR(255) NOT NULL,
    executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Performance metrics
    duration_ms INTEGER,
    row_count INTEGER,
    
    -- Generated SQL (for audit/debugging)
    generated_sql TEXT,
    
    -- Parameters used (for audit/replay)
    parameters_used JSONB,
    
    -- Result summary
    status VARCHAR(50) DEFAULT 'success', -- success, error, timeout
    error_message TEXT,
    
    -- Cache information
    cache_hit BOOLEAN DEFAULT FALSE,
    cache_layer VARCHAR(50) -- nl, query, sql, none
);

-- Indexes for execution history
CREATE INDEX idx_template_executions_template_id ON semantic_query_template_executions(template_id);
CREATE INDEX idx_template_executions_executed_by ON semantic_query_template_executions(executed_by);
CREATE INDEX idx_template_executions_executed_at ON semantic_query_template_executions(executed_at DESC);
CREATE INDEX idx_template_executions_status ON semantic_query_template_executions(status);

-- Partition by month for large execution tables
-- This allows for better performance on time-range queries
-- Uncomment when table gets large (millions of rows)
-- CREATE TABLE semantic_query_template_executions_202502 PARTITION OF semantic_query_template_executions
--   FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');

-- ============================================================================
-- Default Role Permissions
-- Insert default role configurations for all new templates
-- ============================================================================

CREATE OR REPLACE FUNCTION create_default_template_permissions()
RETURNS TRIGGER AS $$
BEGIN
  -- Viewer: Can only run templates
  INSERT INTO semantic_query_template_permissions 
  (template_id, role, can_run, can_edit, can_delete, can_promote) 
  VALUES (NEW.id, 'viewer', true, false, false, false);
  
  -- Editor: Can run and edit
  INSERT INTO semantic_query_template_permissions 
  (template_id, role, can_run, can_edit, can_delete, can_promote) 
  VALUES (NEW.id, 'editor', true, true, false, false);
  
  -- Admin: Full access
  INSERT INTO semantic_query_template_permissions 
  (template_id, role, can_run, can_edit, can_delete, can_promote) 
  VALUES (NEW.id, 'admin', true, true, true, true);
  
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_create_default_permissions
AFTER INSERT ON semantic_query_templates
FOR EACH ROW
EXECUTE FUNCTION create_default_template_permissions();

-- ============================================================================
-- Automatic Version Creation on Update
-- Snapshots template state when it's updated
-- ============================================================================

CREATE OR REPLACE FUNCTION create_template_version_on_update()
RETURNS TRIGGER AS $$
DECLARE
  v_version_number INTEGER;
BEGIN
  -- Get next version number
  SELECT COALESCE(MAX(version_number), 0) + 1
  INTO v_version_number
  FROM semantic_query_template_versions
  WHERE template_id = NEW.id;
  
  -- Create version snapshot
  INSERT INTO semantic_query_template_versions 
  (template_id, version_number, name, description, semantic_query, parameters, 
   change_message, created_by)
  VALUES (
    NEW.id,
    v_version_number,
    NEW.name,
    NEW.description,
    NEW.semantic_query,
    NEW.parameters,
    '', -- Change message would be provided by app layer
    NEW.updated_by
  );
  
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_create_version_on_update
AFTER UPDATE ON semantic_query_templates
FOR EACH ROW
WHEN (OLD.semantic_query IS DISTINCT FROM NEW.semantic_query
   OR OLD.parameters IS DISTINCT FROM NEW.parameters
   OR OLD.name IS DISTINCT FROM NEW.name)
EXECUTE FUNCTION create_template_version_on_update();

-- ============================================================================
-- Automatic Timestamp Update
-- Updates updated_at on every modification
-- ============================================================================

CREATE OR REPLACE FUNCTION update_semantic_templates_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = CURRENT_TIMESTAMP;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_templates_timestamp
BEFORE UPDATE ON semantic_query_templates
FOR EACH ROW
EXECUTE FUNCTION update_semantic_templates_timestamp();

-- ============================================================================
-- Views for Common Queries
-- ============================================================================

-- Latest version of each template
CREATE OR REPLACE VIEW v_template_latest_versions AS
SELECT DISTINCT ON (t.id)
  t.id, t.tenant_id, t.name, t.datasource, t.version as latest_version,
  tv.version_number, tv.created_at, tv.created_by
FROM semantic_query_templates t
JOIN semantic_query_template_versions tv ON t.id = tv.template_id
ORDER BY t.id, tv.version_number DESC;

-- Template usage statistics
CREATE OR REPLACE VIEW v_template_statistics AS
SELECT 
  template_id,
  COUNT(*) as execution_count,
  MAX(executed_at) as last_executed_at,
  AVG(duration_ms) as avg_duration_ms,
  SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END) as cache_hits,
  SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as successful_executions,
  SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as failed_executions
FROM semantic_query_template_executions
GROUP BY template_id;

-- ============================================================================
-- Cleanup/Maintenance Queries
-- ============================================================================

-- Delete old execution records (run periodically via cron)
-- DELETE FROM semantic_query_template_executions 
-- WHERE executed_at < CURRENT_TIMESTAMP - INTERVAL '90 days';

-- Archive deprecated templates (after 30 days deprecated)
-- SELECT id FROM semantic_query_templates 
-- WHERE deprecated = true 
-- AND updated_at < CURRENT_TIMESTAMP - INTERVAL '30 days';
