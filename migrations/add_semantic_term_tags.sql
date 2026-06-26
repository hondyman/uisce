-- Migration: Add Tags Support for Semantic Terms in Catalog
-- Purpose: Enable tagging system for semantic terms to support classification, organization, and wizard suggestions
-- Date: 2026-01-04

-- Step 1: Add tags JSONB column to catalog_node table
-- This stores tags as a JSON array for flexible schema evolution
ALTER TABLE IF EXISTS public.catalog_node 
ADD COLUMN IF NOT EXISTS tags JSONB DEFAULT '[]'::jsonb;

-- Create index for efficient tag queries
CREATE INDEX IF NOT EXISTS idx_catalog_node_tags ON public.catalog_node USING GIN(tags);

-- Step 2: Add tags to semantic_term properties schema
-- Update the semantic_term node type with tag property definitions
UPDATE public.catalog_node_type 
SET properties = COALESCE(properties, '[]'::jsonb) || 
  jsonb_build_array(
    jsonb_build_object(
      'name', 'tags',
      'title', 'Tags',
      'data_type', 'array',
      'order', 10,
      'required', false,
      'description', 'Semantic term classification tags',
      'tag_categories', jsonb_build_array(
        jsonb_build_object('category', 'business_area', 'display_name', 'Business Area', 'icon', 'briefcase'),
        jsonb_build_object('category', 'data_type', 'display_name', 'Data Type', 'icon', 'database'),
        jsonb_build_object('category', 'domain', 'display_name', 'Domain', 'icon', 'tag'),
        jsonb_build_object('category', 'usage_pattern', 'display_name', 'Usage Pattern', 'icon', 'flow'),
        jsonb_build_object('category', 'sensitivity', 'display_name', 'Sensitivity', 'icon', 'shield'),
        jsonb_build_object('category', 'governance', 'display_name', 'Governance', 'icon', 'check-circle')
      )
    )
  )
WHERE catalog_type_name = 'semantic_term' AND tenant_id = 'default';

-- Step 3: Create semantic_term_tags reference table for normalization (optional for complex tag management)
-- This allows for tag reusability and hierarchical tag structures
CREATE TABLE IF NOT EXISTS public.semantic_term_tags (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    tag_key VARCHAR(255) NOT NULL UNIQUE,
    tag_label VARCHAR(255) NOT NULL,
    tag_category VARCHAR(100) NOT NULL, -- business_area, data_type, domain, usage_pattern, sensitivity, governance
    description TEXT,
    color_code VARCHAR(7),
    icon_name VARCHAR(255),
    auto_suggest BOOLEAN DEFAULT false,
    sort_order INT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, tag_key)
);

CREATE INDEX IF NOT EXISTS idx_semantic_term_tags_tenant ON public.semantic_term_tags(tenant_id);
CREATE INDEX IF NOT EXISTS idx_semantic_term_tags_category ON public.semantic_term_tags(tag_category);
CREATE INDEX IF NOT EXISTS idx_semantic_term_tags_auto_suggest ON public.semantic_term_tags(auto_suggest) WHERE auto_suggest = true;

-- Step 4: Create semantic_term_tag_usage table to track tag suggestions and frequency
-- This enables the wizard to learn which tags are most commonly used
CREATE TABLE IF NOT EXISTS public.semantic_term_tag_suggestions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    semantic_term_id uuid NOT NULL,
    tag_key VARCHAR(255) NOT NULL,
    suggestion_reason VARCHAR(100), -- inferred_from_datatype, inferred_from_domain, inferred_from_name, inferred_from_expression, user_created, auto_suggested
    confidence_score FLOAT DEFAULT 0.0, -- 0.0-1.0 confidence level
    is_accepted BOOLEAN,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_term_tag_suggestions_term ON public.semantic_term_tag_suggestions(semantic_term_id);
CREATE INDEX IF NOT EXISTS idx_term_tag_suggestions_tag ON public.semantic_term_tag_suggestions(tag_key);
CREATE INDEX IF NOT EXISTS idx_term_tag_suggestions_reason ON public.semantic_term_tag_suggestions(suggestion_reason);

-- Step 5: Insert predefined tags for common use cases
INSERT INTO public.semantic_term_tags (
    tenant_id, tag_key, tag_label, tag_category, description, color_code, auto_suggest, sort_order
) VALUES
-- Business Area Tags
('default', 'sales', 'Sales', 'business_area', 'Related to sales and revenue', '#2E7D32', true, 1),
('default', 'finance', 'Finance', 'business_area', 'Related to financial operations', '#1565C0', true, 2),
('default', 'marketing', 'Marketing', 'business_area', 'Related to marketing and campaigns', '#C2185B', true, 3),
('default', 'operations', 'Operations', 'business_area', 'Related to operations', '#F57C00', true, 4),
('default', 'hr', 'Human Resources', 'business_area', 'Related to HR and people', '#6A1B9A', true, 5),
('default', 'product', 'Product', 'business_area', 'Related to product management', '#0097A7', true, 6),
('default', 'supply_chain', 'Supply Chain', 'business_area', 'Related to supply chain', '#00695C', true, 7),
('default', 'customer', 'Customer', 'business_area', 'Related to customer data', '#C62828', true, 8),

-- Data Type Tags
('default', 'numeric', 'Numeric', 'data_type', 'Numeric/quantitative data', '#FF6F00', true, 1),
('default', 'text', 'Text', 'data_type', 'Text/string data', '#00838F', true, 2),
('default', 'date', 'Date', 'data_type', 'Date/temporal data', '#7B1FA2', true, 3),
('default', 'boolean', 'Boolean', 'data_type', 'Boolean/flag data', '#00BCD4', true, 4),
('default', 'categorical', 'Categorical', 'data_type', 'Categorical/enumerated data', '#5E35B1', true, 5),

-- Domain Tags
('default', 'financial_metric', 'Financial Metric', 'domain', 'Financial calculations and metrics', '#D32F2F', true, 1),
('default', 'kpi', 'KPI', 'domain', 'Key Performance Indicator', '#F57F17', true, 2),
('default', 'dimension', 'Dimension', 'domain', 'Dimensional attribute', '#1976D2', true, 3),
('default', 'measure', 'Measure', 'domain', 'Quantitative measure', '#388E3C', true, 4),
('default', 'derived_metric', 'Derived Metric', 'domain', 'Calculated/derived metric', '#7B1FA2', true, 5),

-- Usage Pattern Tags
('default', 'frequently_used', 'Frequently Used', 'usage_pattern', 'Heavily used in reports/dashboards', '#F57C00', true, 1),
('default', 'deprecated', 'Deprecated', 'usage_pattern', 'Legacy term to be phased out', '#9C27B0', true, 2),
('default', 'high_priority', 'High Priority', 'usage_pattern', 'High priority for governance', '#E91E63', true, 3),
('default', 'experimental', 'Experimental', 'usage_pattern', 'New/experimental term', '#00BCD4', true, 4),
('default', 'dashboard_ready', 'Dashboard Ready', 'usage_pattern', 'Approved for dashboard usage', '#4CAF50', true, 5),

-- Sensitivity Tags
('default', 'public', 'Public', 'sensitivity', 'Public/non-sensitive data', '#4CAF50', true, 1),
('default', 'internal', 'Internal', 'sensitivity', 'Internal use only', '#2196F3', true, 2),
('default', 'confidential', 'Confidential', 'sensitivity', 'Confidential/restricted data', '#F57F17', true, 3),
('default', 'pii', 'PII', 'sensitivity', 'Personally Identifiable Information', '#D32F2F', true, 4),
('default', 'regulated', 'Regulated', 'sensitivity', 'Regulated/compliance data', '#C62828', true, 5),

-- Governance Tags
('default', 'certified', 'Certified', 'governance', 'Certified/approved by governance', '#4CAF50', true, 1),
('default', 'needs_review', 'Needs Review', 'governance', 'Requires governance review', '#FF9800', true, 2),
('default', 'under_construction', 'Under Construction', 'governance', 'Still being defined', '#9C27B0', true, 3),
('default', 'published', 'Published', 'governance', 'Published to business users', '#1976D2', true, 4),
('default', 'legacy', 'Legacy', 'governance', 'Legacy term with ongoing usage', '#795548', true, 5)
ON CONFLICT (tenant_id, tag_key) DO NOTHING;

-- Step 6: Add tags property to semantic_term view (if exists)
-- This allows the view to surface tags from the catalog_node table
-- Views will be updated in the application layer to include tags from catalog_node

-- Verification queries
SELECT '=== Semantic Term Tags Migration Complete ===' as status;

-- Verify tags column exists
SELECT column_name, data_type 
FROM information_schema.columns 
WHERE table_name = 'catalog_node' AND column_name = 'tags';

-- Verify tag index exists
SELECT indexname FROM pg_indexes WHERE tablename = 'catalog_node' AND indexname = 'idx_catalog_node_tags';

-- Count predefined tags by category
SELECT 
    tag_category, 
    COUNT(*) as tag_count
FROM public.semantic_term_tags 
WHERE tenant_id = 'default'
GROUP BY tag_category
ORDER BY tag_category;
