-- =============================================================================
-- Migration: Marketplace for Rules and Calculations
-- =============================================================================
-- This migration creates tables for a marketplace where organizations can:
-- 1. Browse and discover pre-built rules and calculations
-- 2. Add them to their platform/tenant
-- 3. Manage their usage and versions

-- Main marketplace items (rules and calculations)
CREATE TABLE IF NOT EXISTS marketplace_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Basic metadata
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    item_type VARCHAR(50) NOT NULL, -- 'rule' or 'calculation'
    version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    
    -- Categorization
    category VARCHAR(100) NOT NULL, -- 'ESG', 'Risk', 'Compliance', 'Performance', etc.
    subcategories TEXT[] DEFAULT '{}', -- Array of tags: 'Private Capital', 'Mutual Funds', etc.
    severity VARCHAR(20), -- For rules: BLOCK, WARNING, INFO
    
    -- Detailed metadata
    icon_emoji VARCHAR(10), -- e.g., "🌱", "⚖️"
    color_hex VARCHAR(7), -- e.g., "#10B981"
    summary TEXT, -- Brief one-liner
    long_description TEXT, -- Detailed explanation
    
    -- Implementation details
    implementation_json JSONB NOT NULL, -- Schema, logic, parameters
    scope VARCHAR(50), -- 'PORTFOLIO', 'SECURITY', 'ACCOUNT', etc.
    rule_type VARCHAR(100), -- 'CONDITION', 'ACTION', 'CALCULATION', etc.
    
    -- Usage information
    frequency VARCHAR(50), -- 'ON_TRADE', 'DAILY', 'MONTHLY', etc.
    evaluation_order INTEGER, -- Execution priority
    
    -- Ownership and status
    creator_id UUID, -- Organization/vendor that created this
    is_public BOOLEAN DEFAULT TRUE, -- Can other tenants see/use it?
    is_official BOOLEAN DEFAULT FALSE, -- Is this an official/recommended item?
    is_core BOOLEAN DEFAULT FALSE, -- Is this a core/essential item?
    status VARCHAR(50) DEFAULT 'active', -- 'active', 'deprecated', 'beta', 'archived'
    
    -- External API integrations
    external_api_providers TEXT[], -- Array: 'MSCI', 'Bloomberg', 'AWS', etc.
    requires_credentials BOOLEAN DEFAULT FALSE,
    
    -- Ratings and usage
    usage_count INTEGER DEFAULT 0,
    rating DECIMAL(3,2), -- 1.0 to 5.0
    downloads_count INTEGER DEFAULT 0,
    
    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT valid_item_type CHECK (item_type IN ('rule', 'calculation')),
    CONSTRAINT valid_status CHECK (status IN ('active', 'deprecated', 'beta', 'archived'))
);

-- Tenant marketplace subscriptions (which items they've added)
CREATE TABLE IF NOT EXISTS tenant_marketplace_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    marketplace_item_id UUID NOT NULL REFERENCES marketplace_items(id) ON DELETE CASCADE,
    
    -- Customization for this tenant
    custom_name VARCHAR(255), -- Tenant can rename locally
    custom_parameters JSONB, -- Tenant-specific config overrides
    enabled_for_tenant BOOLEAN DEFAULT TRUE,
    
    -- Usage tracking
    added_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP WITH TIME ZONE,
    usage_count INTEGER DEFAULT 0,
    
    -- Versioning
    marketplace_version_at_time_of_add VARCHAR(20), -- What version they added
    local_version VARCHAR(20), -- Their current version (may differ from marketplace)
    has_local_modifications BOOLEAN DEFAULT FALSE,
    
    -- Feedback
    tenant_rating INTEGER, -- 1-5 stars
    tenant_feedback TEXT,
    
    CONSTRAINT unique_tenant_item UNIQUE (tenant_id, marketplace_item_id)
);

-- Rule/calculation parameters registry
CREATE TABLE IF NOT EXISTS marketplace_item_parameters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    marketplace_item_id UUID NOT NULL REFERENCES marketplace_items(id) ON DELETE CASCADE,
    
    -- Parameter definition
    param_name VARCHAR(255) NOT NULL,
    param_type VARCHAR(50) NOT NULL, -- 'string', 'number', 'boolean', 'date', 'array', etc.
    description TEXT,
    
    -- Configuration
    is_required BOOLEAN DEFAULT FALSE,
    default_value JSONB,
    validation_rules JSONB, -- Min, max, pattern, etc.
    
    -- Display
    display_name VARCHAR(255),
    display_order INTEGER DEFAULT 0,
    help_text TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_param_name UNIQUE (marketplace_item_id, param_name)
);

-- Usage/analytics for marketplace items
CREATE TABLE IF NOT EXISTS marketplace_item_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    marketplace_item_id UUID NOT NULL REFERENCES marketplace_items(id) ON DELETE CASCADE,
    
    -- Execution tracking
    execution_date DATE NOT NULL,
    execution_count INTEGER DEFAULT 1,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    average_execution_time_ms INTEGER,
    
    -- Performance
    last_result_status VARCHAR(50), -- 'success', 'failure', 'warning'
    last_error_message TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_tenant_item_date UNIQUE (tenant_id, marketplace_item_id, execution_date)
);

-- Versions/revisions of marketplace items
CREATE TABLE IF NOT EXISTS marketplace_item_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    marketplace_item_id UUID NOT NULL REFERENCES marketplace_items(id) ON DELETE CASCADE,
    
    version VARCHAR(20) NOT NULL,
    implementation_json JSONB NOT NULL, -- The actual content at this version
    changelog TEXT, -- What changed
    
    created_by UUID, -- Who published this version
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    is_deprecated BOOLEAN DEFAULT FALSE,
    deprecation_reason TEXT,
    replacement_version VARCHAR(20), -- Point to newer version if deprecated
    
    CONSTRAINT unique_version UNIQUE (marketplace_item_id, version)
);

-- Tenant feedback on marketplace items
CREATE TABLE IF NOT EXISTS marketplace_item_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    marketplace_item_id UUID NOT NULL REFERENCES marketplace_items(id) ON DELETE CASCADE,
    
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    feedback_text TEXT,
    helpful_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_feedback UNIQUE (tenant_id, marketplace_item_id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_marketplace_items_type ON marketplace_items(item_type);
CREATE INDEX IF NOT EXISTS idx_marketplace_items_category ON marketplace_items(category);
CREATE INDEX IF NOT EXISTS idx_marketplace_items_status ON marketplace_items(status);
CREATE INDEX IF NOT EXISTS idx_marketplace_items_public ON marketplace_items(is_public, status);
CREATE INDEX IF NOT EXISTS idx_marketplace_items_official ON marketplace_items(is_official, status);

CREATE INDEX IF NOT EXISTS idx_tenant_marketplace_items_tenant ON tenant_marketplace_items(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_marketplace_items_item ON tenant_marketplace_items(marketplace_item_id);
CREATE INDEX IF NOT EXISTS idx_tenant_marketplace_items_added ON tenant_marketplace_items(tenant_id, added_at DESC);
CREATE INDEX IF NOT EXISTS idx_tenant_marketplace_items_enabled ON tenant_marketplace_items(tenant_id, enabled_for_tenant);

CREATE INDEX IF NOT EXISTS idx_marketplace_item_usage_tenant ON marketplace_item_usage(tenant_id);
CREATE INDEX IF NOT EXISTS idx_marketplace_item_usage_date ON marketplace_item_usage(execution_date DESC);

-- Update triggers
CREATE OR REPLACE FUNCTION update_marketplace_items_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER marketplace_items_timestamp_trigger
BEFORE UPDATE ON marketplace_items
FOR EACH ROW
EXECUTE FUNCTION update_marketplace_items_timestamp();

CREATE TRIGGER tenant_marketplace_items_timestamp_trigger
BEFORE UPDATE ON tenant_marketplace_items
FOR EACH ROW
EXECUTE FUNCTION update_marketplace_items_timestamp();

CREATE TRIGGER marketplace_item_usage_timestamp_trigger
BEFORE UPDATE ON marketplace_item_usage
FOR EACH ROW
EXECUTE FUNCTION update_marketplace_items_timestamp();

CREATE TRIGGER marketplace_item_feedback_timestamp_trigger
BEFORE UPDATE ON marketplace_item_feedback
FOR EACH ROW
EXECUTE FUNCTION update_marketplace_items_timestamp();

-- ============================================================================
-- Sample data: Initial marketplace items (rules)
-- ============================================================================

INSERT INTO marketplace_items (
    name, description, item_type, category, subcategories, severity,
    icon_emoji, color_hex, summary, long_description,
    implementation_json, scope, rule_type, frequency, evaluation_order,
    is_public, is_official, is_core, status
) VALUES
(
    'ESG Compliance',
    'Validate environmental, social and governance compliance requirements',
    'rule',
    'ESG & Sustainability',
    ARRAY['ESG', 'Compliance'],
    'BLOCK',
    '🌱',
    '#10B981',
    'Ensure ESG compliance before executing trades',
    'Checks environmental, social, and governance metrics against policy thresholds. Blocks trades that fail ESG criteria.',
    '{"type": "ESG_COMPLIANCE", "provider": "MSCI", "metrics": ["carbon_score", "esg_rating"]}'::jsonb,
    'PORTFOLIO',
    'CONDITION',
    'ON_TRADE',
    1,
    TRUE,
    TRUE,
    TRUE,
    'active'
),
(
    'AML Compliance Check',
    'Screen accounts against AML watchlists and sanctions databases',
    'rule',
    'Compliance & Regulatory',
    ARRAY['AML', 'Compliance', 'Risk'],
    'BLOCK',
    '⚖️',
    '#059669',
    'Check AML compliance before allowing transactions',
    'Screens counterparties and accounts against World-Check and other AML databases. Blocks suspicious transactions.',
    '{"type": "AML_SCREENING", "provider": "World-Check", "refresh_frequency": "daily"}'::jsonb,
    'ACCOUNT',
    'CONDITION',
    'ON_TRADE',
    2,
    TRUE,
    TRUE,
    TRUE,
    'active'
),
(
    'Margin Compliance',
    'Ensure account margin requirements are met',
    'rule',
    'Risk Management',
    ARRAY['Margin', 'Risk', 'Compliance'],
    'BLOCK',
    '⚠️',
    '#EF4444',
    'Block trades that violate margin requirements',
    'Validates that proposed trades maintain adequate margin levels. Prevents under-margined positions.',
    '{"type": "MARGIN_CHECK", "fields": ["buying_power", "required_margin"]}'::jsonb,
    'ACCOUNT',
    'CONDITION',
    'ON_TRADE',
    3,
    TRUE,
    TRUE,
    TRUE,
    'active'
),
(
    'Concentration Limit',
    'Prevent excessive concentration in single securities',
    'rule',
    'Risk Management',
    ARRAY['Concentration', 'Risk', 'Portfolio'],
    'WARNING',
    '📊',
    '#3B82F6',
    'Warn when position concentration exceeds limits',
    'Validates portfolio concentration limits. Issues warning when single position exceeds threshold percentage.',
    '{"type": "CONCENTRATION_LIMIT", "threshold_pct": 10}'::jsonb,
    'PORTFOLIO',
    'CONDITION',
    'DAILY',
    4,
    TRUE,
    TRUE,
    FALSE,
    'active'
);

-- Add more sample calculations if needed...
