-- Migration: Create ai_layouts table
-- Stores AI-generated page layouts with tenant scoping, draft lifecycle, and adoption tracking
-- All layouts are associated with a tenant, primary BO, and model version for auditability

CREATE TABLE IF NOT EXISTS ai_layouts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  primary_bo VARCHAR(255) NOT NULL,
  name VARCHAR(255) NOT NULL,
  layout_type VARCHAR(50) NOT NULL CHECK (layout_type IN ('detail', 'form', 'list')),
  payload JSONB NOT NULL, -- Full PageLayout JSON structure
  model_version VARCHAR(100),
  confidence NUMERIC(3, 2), -- 0.00 to 1.00
  alternatives JSONB DEFAULT '[]', -- Array of alternative PageLayout options
  explanation TEXT, -- Model explanation of why this layout was suggested
  adopted BOOLEAN NOT NULL DEFAULT FALSE, -- TRUE when user applies to actual layout
  adopted_at TIMESTAMP,
  adopted_by TEXT REFERENCES public.app_user(id) ON DELETE SET NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by TEXT, -- May be system or user email
  is_active BOOLEAN NOT NULL DEFAULT TRUE
);

-- Index for efficient tenant + BO queries (primary query pattern)
CREATE INDEX IF NOT EXISTS idx_ai_layouts_tenant_bo 
  ON ai_layouts(tenant_id, primary_bo) 
  WHERE is_active = TRUE AND adopted = FALSE;

-- Index for finding adopted layouts (audit trail)
CREATE INDEX IF NOT EXISTS idx_ai_layouts_adopted 
  ON ai_layouts(tenant_id, adopted, adopted_at);

-- Index for quick ID lookups
CREATE INDEX IF NOT EXISTS idx_ai_layouts_id_active 
  ON ai_layouts(id) 
  WHERE is_active = TRUE;

-- Index for time-based retention policies
CREATE INDEX IF NOT EXISTS idx_ai_layouts_created 
  ON ai_layouts(created_at DESC) 
  WHERE is_active = TRUE;

-- Comments for documentation
COMMENT ON TABLE ai_layouts IS 'Stores AI-generated page layouts in draft form. Layouts are unadopted until user applies them to the active editor. Supports rollback and audit trails via adopted_at and adopted_by.';
COMMENT ON COLUMN ai_layouts.tenant_id IS 'Foreign key to tenants table for multi-tenancy.';
COMMENT ON COLUMN ai_layouts.primary_bo IS 'Primary Business Object name (e.g., "Customer", "Order"). Used for filtering recommendations.';
COMMENT ON COLUMN ai_layouts.layout_type IS 'One of: detail (record view), form (edit/create), list (grid/table).';
COMMENT ON COLUMN ai_layouts.payload IS 'Full PageLayout JSON including sections, fields, relationships. Matches frontend PageLayout interface.';
COMMENT ON COLUMN ai_layouts.model_version IS 'AI model or rule version that generated this layout (e.g., "rulebased-v1", "gpt-4-2024-01").';
COMMENT ON COLUMN ai_layouts.confidence IS 'Confidence score (0.0-1.0) for this suggestion.';
COMMENT ON COLUMN ai_layouts.alternatives IS 'Array of alternative PageLayout options presented to user.';
COMMENT ON COLUMN ai_layouts.explanation IS 'Human-readable explanation of the suggestion (e.g., "Matched prompt keywords and common patterns").';
COMMENT ON COLUMN ai_layouts.adopted IS 'TRUE when user has applied this layout to the editor and published.';
COMMENT ON COLUMN ai_layouts.adopted_at IS 'Timestamp when layout was adopted.';
COMMENT ON COLUMN ai_layouts.adopted_by IS 'User ID who adopted this layout.';
COMMENT ON COLUMN ai_layouts.created_by IS 'Email or system identifier of who triggered generation.';
COMMENT ON COLUMN ai_layouts.is_active IS 'Soft delete flag. Unadopted layouts older than retention period should be archived.';
