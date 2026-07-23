-- Household Reports Schema
-- Creates tables for household ledger, report templates, and execution tracking

-- Household ledger stores aggregated financial data per household per period
CREATE TABLE IF NOT EXISTS household_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    household_id UUID NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    
    -- Aggregated data from semantic views
    data JSONB NOT NULL DEFAULT '{}'::jsonb,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT,
    
    -- Ensure one ledger entry per household per period
    CONSTRAINT ux_household_ledger_period UNIQUE (tenant_id, household_id, period_start, period_end)
);

CREATE INDEX IF NOT EXISTS idx_household_ledger_tenant ON household_ledger(tenant_id);
CREATE INDEX IF NOT EXISTS idx_household_ledger_household ON household_ledger(household_id);
CREATE INDEX IF NOT EXISTS idx_household_ledger_period ON household_ledger(period_start, period_end);

-- Report templates define reusable report configurations
CREATE TABLE IF NOT EXISTS report_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- Template identification
    template_name TEXT NOT NULL,
    description TEXT,
    category TEXT, -- e.g., 'wealth_summary', 'tax_report', 'performance'
    
    -- Semantic views to query
    semantic_view_ids UUID[] NOT NULL DEFAULT '{}',
    
    -- Layout configuration (defines how data is arranged in PDF)
    layout_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    
    -- Report parameters schema (what inputs are needed)
    parameter_schema JSONB DEFAULT '{}'::jsonb,
    
    -- Status and visibility
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_public BOOLEAN NOT NULL DEFAULT false, -- Public templates vs tenant-specific
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT,
    version INTEGER NOT NULL DEFAULT 1,
    
    CONSTRAINT ux_report_template_name UNIQUE (tenant_id, template_name)
);

CREATE INDEX IF NOT EXISTS idx_report_templates_tenant ON report_templates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_report_templates_category ON report_templates(category);
CREATE INDEX IF NOT EXISTS idx_report_templates_active ON report_templates(is_active);

-- Report executions track generated reports
CREATE TABLE IF NOT EXISTS report_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    template_id UUID NOT NULL REFERENCES report_templates(id) ON DELETE CASCADE,
    
    -- Execution context
    household_id UUID,
    parameters JSONB DEFAULT '{}'::jsonb,
    
    -- Status tracking
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    error_message TEXT,
    
    -- Output
    output_url TEXT, -- S3/GCS URL to generated PDF
    output_size_bytes INTEGER,
    
    -- Performance metrics
    execution_time_ms INTEGER,
    rows_processed INTEGER,
    
    -- Temporal workflow (if async)
    workflow_id TEXT,
    run_id TEXT,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_by TEXT
);

CREATE INDEX IF NOT EXISTS idx_report_executions_tenant ON report_executions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_report_executions_template ON report_executions(template_id);
CREATE INDEX IF NOT EXISTS idx_report_executions_household ON report_executions(household_id);
CREATE INDEX IF NOT EXISTS idx_report_executions_status ON report_executions(status);
CREATE INDEX IF NOT EXISTS idx_report_executions_created ON report_executions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_report_executions_workflow ON report_executions(workflow_id) WHERE workflow_id IS NOT NULL;

-- Insert default wealth summary template
INSERT INTO report_templates (
    tenant_id,
    template_name,
    description,
    category,
    semantic_view_ids,
    layout_config,
    parameter_schema,
    is_public
) VALUES (
    '00000000-0000-0000-0000-000000000000'::uuid, -- System tenant
    'Wealth Summary Report',
    'Comprehensive household wealth overview including assets, liabilities, and net worth',
    'wealth_summary',
    '{}', -- Will be populated with actual semantic view IDs
    '{
        "sections": [
            {"title": "Executive Summary", "type": "summary", "order": 1},
            {"title": "Asset Allocation", "type": "chart", "order": 2, "chart_type": "pie"},
            {"title": "Performance Summary", "type": "table", "order": 3},
            {"title": "Tax Implications", "type": "table", "order": 4}
        ],
        "page_size": "letter",
        "orientation": "portrait",
        "include_toc": true
    }'::jsonb,
    '{
        "fields": [
            {"name": "household_id", "type": "uuid", "required": true},
            {"name": "period_start", "type": "date", "required": true},
            {"name": "period_end", "type": "date", "required": true},
            {"name": "include_projections", "type": "boolean", "required": false, "default": false}
        ]
    }'::jsonb,
    true -- Public template available to all tenants
) ON CONFLICT (tenant_id, template_name) DO NOTHING;

-- Add comments for documentation
COMMENT ON TABLE household_ledger IS 'Stores aggregated household financial data for report generation';
COMMENT ON TABLE report_templates IS 'Reusable report configurations that define data sources and layout';
COMMENT ON TABLE report_executions IS 'Tracks report generation jobs and stores output URLs';

COMMENT ON COLUMN household_ledger.data IS 'JSONB containing aggregated metrics from semantic views';
COMMENT ON COLUMN report_templates.semantic_view_ids IS 'Array of semantic view UUIDs to query for report data';
COMMENT ON COLUMN report_templates.layout_config IS 'JSONB defining PDF layout, sections, charts, and styling';
COMMENT ON COLUMN report_executions.workflow_id IS 'Temporal workflow ID if executed asynchronously';
