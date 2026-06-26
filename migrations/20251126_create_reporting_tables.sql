-- Migration: 20251126_create_reporting_tables
-- Description: Create semantic reporting platform tables
-- Author: semlayer
-- Date: 2025-11-26

BEGIN;

-- ============================================================================
-- REPORT DEFINITIONS (Core templates)
-- ============================================================================
CREATE TABLE IF NOT EXISTS report_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    tenant_datasource_id UUID NOT NULL REFERENCES tenant_datasources(id) ON DELETE CASCADE,
    
    -- Identity
    report_key VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    tags JSONB DEFAULT '[]',
    
    -- Type/Classification
    report_type VARCHAR(50) NOT NULL DEFAULT 'paginated',
    output_formats JSONB DEFAULT '["pdf", "html", "excel"]',
    
    -- Definition (metadata-first)
    definition JSONB NOT NULL,
    parameters_schema JSONB DEFAULT '[]',
    
    -- Semantic Layer Binding
    semantic_cube_id UUID,
    semantic_query JSONB,
    
    -- Versioning
    version INTEGER NOT NULL DEFAULT 1,
    is_current BOOLEAN NOT NULL DEFAULT TRUE,
    previous_version_id UUID REFERENCES report_definitions(id),
    
    -- Ownership
    is_core BOOLEAN NOT NULL DEFAULT FALSE,
    base_report_id UUID REFERENCES report_definitions(id),
    
    -- Lifecycle
    status VARCHAR(50) DEFAULT 'draft',
    published_at TIMESTAMPTZ,
    published_by UUID,
    
    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by UUID,
    
    CONSTRAINT uq_report_definition_version UNIQUE(tenant_id, tenant_datasource_id, report_key, version)
);

-- Indexes for report_definitions
CREATE INDEX idx_report_definitions_tenant ON report_definitions(tenant_id, tenant_datasource_id);
CREATE INDEX idx_report_definitions_key ON report_definitions(report_key);
CREATE INDEX idx_report_definitions_category ON report_definitions(category);
CREATE INDEX idx_report_definitions_status ON report_definitions(status);
CREATE INDEX idx_report_definitions_is_current ON report_definitions(is_current) WHERE is_current = TRUE;
CREATE INDEX idx_report_definitions_is_core ON report_definitions(is_core) WHERE is_core = TRUE;

-- ============================================================================
-- REPORT EXTENSIONS (Tenant customizations)
-- ============================================================================
CREATE TABLE IF NOT EXISTS report_extensions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    tenant_datasource_id UUID NOT NULL REFERENCES tenant_datasources(id) ON DELETE CASCADE,
    
    -- Link to core
    base_report_id UUID NOT NULL REFERENCES report_definitions(id) ON DELETE CASCADE,
    
    -- Extension definition
    extension_key VARCHAR(255) NOT NULL,
    extension_name VARCHAR(255),
    description TEXT,
    
    -- What's customized
    extension_definition JSONB NOT NULL DEFAULT '{}',
    overrides JSONB DEFAULT '{}',
    additions JSONB DEFAULT '{}',
    removals JSONB DEFAULT '{}',
    
    -- Parameter overrides
    parameter_defaults JSONB DEFAULT '{}',
    
    -- Versioning
    version INTEGER NOT NULL DEFAULT 1,
    is_current BOOLEAN NOT NULL DEFAULT TRUE,
    core_version_target INTEGER,
    
    -- Lifecycle
    status VARCHAR(50) DEFAULT 'draft',
    published_at TIMESTAMPTZ,
    published_by UUID,
    
    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by UUID,
    
    CONSTRAINT uq_report_extension_version UNIQUE(tenant_id, tenant_datasource_id, extension_key, version)
);

-- Indexes for report_extensions
CREATE INDEX idx_report_extensions_tenant ON report_extensions(tenant_id, tenant_datasource_id);
CREATE INDEX idx_report_extensions_base ON report_extensions(base_report_id);
CREATE INDEX idx_report_extensions_is_current ON report_extensions(is_current) WHERE is_current = TRUE;

-- ============================================================================
-- REPORT INSTANCES (Generated reports)
-- ============================================================================
CREATE TABLE IF NOT EXISTS report_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    tenant_datasource_id UUID NOT NULL REFERENCES tenant_datasources(id) ON DELETE CASCADE,
    
    -- Source definition
    report_definition_id UUID NOT NULL REFERENCES report_definitions(id) ON DELETE CASCADE,
    report_extension_id UUID REFERENCES report_extensions(id) ON DELETE SET NULL,
    
    -- Merged definition (snapshot at render time)
    merged_definition JSONB,
    
    -- Context (what entity the report is for)
    context_type VARCHAR(100),
    context_id UUID,
    context_name VARCHAR(255),
    
    -- Parameters used
    parameters JSONB DEFAULT '{}',
    
    -- Generated content
    output_format VARCHAR(50) NOT NULL,
    output_data BYTEA,
    output_url VARCHAR(500),
    output_metadata JSONB DEFAULT '{}',
    
    -- Lifecycle
    status VARCHAR(50) DEFAULT 'pending',
    error_message TEXT,
    
    -- Timing
    requested_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    generation_time_ms INTEGER,
    
    -- Requester
    requested_by UUID,
    
    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for report_instances
CREATE INDEX idx_report_instances_tenant ON report_instances(tenant_id, tenant_datasource_id);
CREATE INDEX idx_report_instances_definition ON report_instances(report_definition_id);
CREATE INDEX idx_report_instances_context ON report_instances(context_type, context_id);
CREATE INDEX idx_report_instances_status ON report_instances(status);
CREATE INDEX idx_report_instances_requested ON report_instances(requested_at DESC);
CREATE INDEX idx_report_instances_expires ON report_instances(expires_at) WHERE expires_at IS NOT NULL;

-- ============================================================================
-- REPORT SCHEDULES (Recurring reports)
-- ============================================================================
CREATE TABLE IF NOT EXISTS report_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    tenant_datasource_id UUID NOT NULL REFERENCES tenant_datasources(id) ON DELETE CASCADE,
    
    -- Report definition
    report_definition_id UUID NOT NULL REFERENCES report_definitions(id) ON DELETE CASCADE,
    report_extension_id UUID REFERENCES report_extensions(id) ON DELETE SET NULL,
    
    -- Schedule
    schedule_name VARCHAR(255) NOT NULL,
    description TEXT,
    cron_expression VARCHAR(100) NOT NULL,
    timezone VARCHAR(50) DEFAULT 'UTC',
    
    -- Parameters template
    parameters_template JSONB DEFAULT '{}',
    
    -- Context (dynamic or fixed)
    context_type VARCHAR(100),
    context_query JSONB,
    fixed_context_id UUID,
    
    -- Output
    output_formats JSONB DEFAULT '["pdf"]',
    
    -- Delivery
    delivery_config JSONB DEFAULT '{}',
    
    -- State
    is_active BOOLEAN DEFAULT TRUE,
    last_run_at TIMESTAMPTZ,
    last_run_status VARCHAR(50),
    last_run_error TEXT,
    next_run_at TIMESTAMPTZ,
    run_count INTEGER DEFAULT 0,
    
    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for report_schedules
CREATE INDEX idx_report_schedules_tenant ON report_schedules(tenant_id, tenant_datasource_id);
CREATE INDEX idx_report_schedules_active ON report_schedules(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_report_schedules_next_run ON report_schedules(next_run_at) WHERE is_active = TRUE;

-- ============================================================================
-- REPORT PROVISIONING PACKAGES (Template bundles for tenant setup)
-- ============================================================================
CREATE TABLE IF NOT EXISTS report_packages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Identity
    package_key VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    
    -- Content
    report_definitions JSONB NOT NULL DEFAULT '[]',
    default_schedules JSONB DEFAULT '[]',
    required_cubes JSONB DEFAULT '[]',
    
    -- Versioning
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    
    -- Metadata
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default packages
INSERT INTO report_packages (package_key, display_name, description, category, report_definitions, required_cubes) VALUES
('wealth_management', 'Wealth Management Reports', 'Comprehensive reporting for wealth management firms', 'wealth', 
 '[
    {"key": "household_summary", "displayName": "Household Summary Report"},
    {"key": "household_performance", "displayName": "Household Performance Report"},
    {"key": "asset_allocation", "displayName": "Asset Allocation Report"},
    {"key": "holdings_detail", "displayName": "Holdings Detail Report"},
    {"key": "transaction_history", "displayName": "Transaction History Report"}
  ]',
 '["households", "household_performance", "household_holdings", "transactions"]'
),
('asset_management', 'Asset Management Reports', 'Portfolio and fund reporting for asset managers', 'asset_management',
 '[
    {"key": "portfolio_summary", "displayName": "Portfolio Summary Report"},
    {"key": "fund_performance", "displayName": "Fund Performance Report"},
    {"key": "investor_statement", "displayName": "Investor Statement"}
  ]',
 '["portfolios", "fund_performance", "investor_positions"]'
),
('compliance', 'Compliance Reports', 'Regulatory and compliance reporting', 'compliance',
 '[
    {"key": "aum_report", "displayName": "AUM Report"},
    {"key": "trade_blotter", "displayName": "Trade Blotter"},
    {"key": "exception_report", "displayName": "Exception Report"}
  ]',
 '["aum_summary", "trades", "compliance_exceptions"]'
)
ON CONFLICT (package_key) DO NOTHING;

-- ============================================================================
-- REPORT AUDIT LOG
-- ============================================================================
CREATE TABLE IF NOT EXISTS report_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- What was affected
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    
    -- What happened
    action VARCHAR(50) NOT NULL,
    changes JSONB,
    
    -- Who did it
    user_id UUID,
    user_email VARCHAR(255),
    
    -- When
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Context
    request_id VARCHAR(100),
    ip_address INET
);

-- Index for audit log
CREATE INDEX idx_report_audit_tenant ON report_audit_log(tenant_id, created_at DESC);
CREATE INDEX idx_report_audit_entity ON report_audit_log(entity_type, entity_id);

-- ============================================================================
-- FUNCTIONS
-- ============================================================================

-- Function to get merged report definition (core + extension)
CREATE OR REPLACE FUNCTION get_merged_report_definition(
    p_report_definition_id UUID,
    p_report_extension_id UUID DEFAULT NULL
) RETURNS JSONB AS $$
DECLARE
    v_core_definition JSONB;
    v_extension JSONB;
    v_merged JSONB;
BEGIN
    -- Get core definition
    SELECT definition INTO v_core_definition
    FROM report_definitions
    WHERE id = p_report_definition_id AND is_current = TRUE;
    
    IF v_core_definition IS NULL THEN
        RETURN NULL;
    END IF;
    
    -- If no extension, return core
    IF p_report_extension_id IS NULL THEN
        RETURN v_core_definition;
    END IF;
    
    -- Get extension
    SELECT jsonb_build_object(
        'overrides', overrides,
        'additions', additions,
        'removals', removals,
        'parameter_defaults', parameter_defaults
    ) INTO v_extension
    FROM report_extensions
    WHERE id = p_report_extension_id AND is_current = TRUE;
    
    IF v_extension IS NULL THEN
        RETURN v_core_definition;
    END IF;
    
    -- Merge (simplified - full merge logic in Go)
    v_merged := v_core_definition || jsonb_build_object('_extension', v_extension);
    
    RETURN v_merged;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION update_reporting_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply triggers
DROP TRIGGER IF EXISTS tr_report_definitions_updated ON report_definitions;
CREATE TRIGGER tr_report_definitions_updated
    BEFORE UPDATE ON report_definitions
    FOR EACH ROW EXECUTE FUNCTION update_reporting_updated_at();

DROP TRIGGER IF EXISTS tr_report_extensions_updated ON report_extensions;
CREATE TRIGGER tr_report_extensions_updated
    BEFORE UPDATE ON report_extensions
    FOR EACH ROW EXECUTE FUNCTION update_reporting_updated_at();

DROP TRIGGER IF EXISTS tr_report_schedules_updated ON report_schedules;
CREATE TRIGGER tr_report_schedules_updated
    BEFORE UPDATE ON report_schedules
    FOR EACH ROW EXECUTE FUNCTION update_reporting_updated_at();

COMMIT;
