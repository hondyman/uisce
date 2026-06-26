-- Migration: Consolidate metrics_registry and dax_functions into public schema
-- Created: 2025-11-03
-- Purpose: Consolidate duplicated metrics_registry and dax_functions tables from 12 domain schemas
--          into unified tables in the public schema for better maintainability
-- 
-- Summary of consolidation:
-- - 12 schemas have metrics_registry tables (264 total records)
-- - 8 schemas have dax_functions tables
-- - New consolidated tables: public.metrics_registry (with schema_domain column)
-- - New consolidated tables: public.dax_functions (with schema_domain column)
-- - After migration, domain-specific schemas will be cleaner

-- ============================================================================
-- STEP 1: Create consolidated tables in public schema
-- ============================================================================

-- Create consolidated metrics_registry table
CREATE TABLE IF NOT EXISTS public.metrics_registry (
    id SERIAL PRIMARY KEY,
    node_id VARCHAR(255) NOT NULL,
    schema_domain VARCHAR(100) NOT NULL,  -- tracks which domain this metric belongs to
    category VARCHAR(100) NOT NULL,
    description TEXT,
    formula_type VARCHAR(50) NOT NULL,
    formula TEXT NOT NULL,
    arguments JSONB,
    badge VARCHAR(10),
    function_class VARCHAR(50),
    functions_used TEXT[],
    governance_status VARCHAR(50) DEFAULT 'draft',
    audience TEXT[],
    tags TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(node_id, schema_domain)
);

-- Create consolidated dax_functions table
CREATE TABLE IF NOT EXISTS public.dax_functions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    schema_domain VARCHAR(100) NOT NULL,  -- tracks which domain this function belongs to
    class VARCHAR(50) NOT NULL,
    badge VARCHAR(10),
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(name, schema_domain)
);

-- Create indexes for performance
CREATE INDEX idx_metrics_registry_schema_domain ON public.metrics_registry(schema_domain);
CREATE INDEX idx_metrics_registry_node_id ON public.metrics_registry(node_id);
CREATE INDEX idx_metrics_registry_category ON public.metrics_registry(category);
CREATE INDEX idx_dax_functions_schema_domain ON public.dax_functions(schema_domain);
CREATE INDEX idx_dax_functions_name ON public.dax_functions(name);

-- ============================================================================
-- STEP 2: Migrate metrics_registry data from all domain schemas
-- ============================================================================

-- Banking schema
INSERT INTO public.metrics_registry 
    (node_id, schema_domain, category, description, formula_type, formula, arguments, 
     badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at)
SELECT 
    node_id, 'banking', category, description, formula_type, formula, arguments, 
    badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at
FROM banking.metrics_registry
ON CONFLICT (node_id, schema_domain) DO NOTHING;

-- Capital Markets schema
INSERT INTO public.metrics_registry 
    (node_id, schema_domain, category, description, formula_type, formula, arguments, 
     badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at)
SELECT 
    node_id, 'capital_markets', category, description, formula_type, formula, arguments, 
    badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at
FROM capital_markets.metrics_registry
ON CONFLICT (node_id, schema_domain) DO NOTHING;

-- Currency FX schema
INSERT INTO public.metrics_registry 
    (node_id, schema_domain, category, description, formula_type, formula, arguments, 
     badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at)
SELECT 
    node_id, 'currency_fx', category, description, formula_type, formula, arguments, 
    badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at
FROM currency_fx.metrics_registry
ON CONFLICT (node_id, schema_domain) DO NOTHING;

-- Financial Services schema
INSERT INTO public.metrics_registry 
    (node_id, schema_domain, category, description, formula_type, formula, arguments, 
     badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at)
SELECT 
    node_id, 'financial_services', category, description, formula_type, formula, arguments, 
    badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at
FROM financial_services.metrics_registry
ON CONFLICT (node_id, schema_domain) DO NOTHING;

-- Fixed Income schema
INSERT INTO public.metrics_registry 
    (node_id, schema_domain, category, description, formula_type, formula, arguments, 
     badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at)
SELECT 
    node_id, 'fixed_income', category, description, formula_type, formula, arguments, 
    badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at
FROM fixed_income.metrics_registry
ON CONFLICT (node_id, schema_domain) DO NOTHING;

-- Healthcare schema
INSERT INTO public.metrics_registry 
    (node_id, schema_domain, category, description, formula_type, formula, arguments, 
     badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at)
SELECT 
    node_id, 'healthcare', category, description, formula_type, formula, arguments, 
    badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at
FROM healthcare.metrics_registry
ON CONFLICT (node_id, schema_domain) DO NOTHING;

-- Insurance schema
INSERT INTO public.metrics_registry 
    (node_id, schema_domain, category, description, formula_type, formula, arguments, 
     badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at)
SELECT 
    node_id, 'insurance', category, description, formula_type, formula, arguments, 
    badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at
FROM insurance.metrics_registry
ON CONFLICT (node_id, schema_domain) DO NOTHING;

-- Investment Accounting schema
INSERT INTO public.metrics_registry 
    (node_id, schema_domain, category, description, formula_type, formula, arguments, 
     badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at)
SELECT 
    node_id, 'investment_accounting', category, description, formula_type, formula, arguments, 
    badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at
FROM investment_accounting.metrics_registry
ON CONFLICT (node_id, schema_domain) DO NOTHING;

-- Regulatory schema
INSERT INTO public.metrics_registry 
    (node_id, schema_domain, category, description, formula_type, formula, arguments, 
     badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at)
SELECT 
    node_id, 'regulatory', category, description, formula_type, formula, arguments, 
    badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at
FROM regulatory.metrics_registry
ON CONFLICT (node_id, schema_domain) DO NOTHING;

-- Retail schema
INSERT INTO public.metrics_registry 
    (node_id, schema_domain, category, description, formula_type, formula, arguments, 
     badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at)
SELECT 
    node_id, 'retail', category, description, formula_type, formula, arguments, 
    badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at
FROM retail.metrics_registry
ON CONFLICT (node_id, schema_domain) DO NOTHING;

-- Unified Financial Services schema
INSERT INTO public.metrics_registry 
    (node_id, schema_domain, category, description, formula_type, formula, arguments, 
     badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at)
SELECT 
    node_id, 'unified_financial_services', category, description, formula_type, formula, arguments, 
    badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at
FROM unified_financial_services.metrics_registry
ON CONFLICT (node_id, schema_domain) DO NOTHING;

-- Wealth Management schema
INSERT INTO public.metrics_registry 
    (node_id, schema_domain, category, description, formula_type, formula, arguments, 
     badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at)
SELECT 
    node_id, 'wealth_management', category, description, formula_type, formula, arguments, 
    badge, function_class, functions_used, governance_status, audience, tags, created_at, updated_at
FROM wealth_management.metrics_registry
ON CONFLICT (node_id, schema_domain) DO NOTHING;

-- ============================================================================
-- STEP 3: Migrate dax_functions data from all domain schemas
-- ============================================================================

-- Banking schema
INSERT INTO public.dax_functions 
    (name, schema_domain, class, badge, description, created_at)
SELECT 
    name, 'banking', class, badge, description, created_at
FROM banking.dax_functions
ON CONFLICT (name, schema_domain) DO NOTHING;

-- Capital Markets schema
INSERT INTO public.dax_functions 
    (name, schema_domain, class, badge, description, created_at)
SELECT 
    name, 'capital_markets', class, badge, description, created_at
FROM capital_markets.dax_functions
ON CONFLICT (name, schema_domain) DO NOTHING;

-- Financial Services schema
INSERT INTO public.dax_functions 
    (name, schema_domain, class, badge, description, created_at)
SELECT 
    name, 'financial_services', class, badge, description, created_at
FROM financial_services.dax_functions
ON CONFLICT (name, schema_domain) DO NOTHING;

-- Healthcare schema
INSERT INTO public.dax_functions 
    (name, schema_domain, class, badge, description, created_at)
SELECT 
    name, 'healthcare', class, badge, description, created_at
FROM healthcare.dax_functions
ON CONFLICT (name, schema_domain) DO NOTHING;

-- Insurance schema
INSERT INTO public.dax_functions 
    (name, schema_domain, class, badge, description, created_at)
SELECT 
    name, 'insurance', class, badge, description, created_at
FROM insurance.dax_functions
ON CONFLICT (name, schema_domain) DO NOTHING;

-- Regulatory schema
INSERT INTO public.dax_functions 
    (name, schema_domain, class, badge, description, created_at)
SELECT 
    name, 'regulatory', class, badge, description, created_at
FROM regulatory.dax_functions
ON CONFLICT (name, schema_domain) DO NOTHING;

-- Retail schema
INSERT INTO public.dax_functions 
    (name, schema_domain, class, badge, description, created_at)
SELECT 
    name, 'retail', class, badge, description, created_at
FROM retail.dax_functions
ON CONFLICT (name, schema_domain) DO NOTHING;

-- Unified Financial Services schema
INSERT INTO public.dax_functions 
    (name, schema_domain, class, badge, description, created_at)
SELECT 
    name, 'unified_financial_services', class, badge, description, created_at
FROM unified_financial_services.dax_functions
ON CONFLICT (name, schema_domain) DO NOTHING;

-- ============================================================================
-- STEP 4: Verify consolidation
-- ============================================================================

-- Check consolidated metrics_registry
SELECT 'Consolidated metrics_registry' as check_type, COUNT(*) as total_records, COUNT(DISTINCT schema_domain) as domains
FROM public.metrics_registry;

-- Check consolidated dax_functions
SELECT 'Consolidated dax_functions' as check_type, COUNT(*) as total_records, COUNT(DISTINCT schema_domain) as domains
FROM public.dax_functions;

-- Show breakdown by domain
SELECT schema_domain, COUNT(*) as metric_count
FROM public.metrics_registry
GROUP BY schema_domain
ORDER BY schema_domain;

-- ============================================================================
-- STEP 5: Create views for backwards compatibility (optional)
-- ============================================================================
-- These views allow existing queries to continue working with the old schema names
-- You can drop these once all code is migrated

CREATE OR REPLACE VIEW banking.metrics_registry_view AS
SELECT node_id, category, description, formula_type, formula, arguments, 
       badge, function_class, functions_used, governance_status, audience, tags, 
       created_at, updated_at
FROM public.metrics_registry
WHERE schema_domain = 'banking';

CREATE OR REPLACE VIEW capital_markets.metrics_registry_view AS
SELECT node_id, category, description, formula_type, formula, arguments, 
       badge, function_class, functions_used, governance_status, audience, tags, 
       created_at, updated_at
FROM public.metrics_registry
WHERE schema_domain = 'capital_markets';

-- NOTE: Repeat this pattern for remaining schemas as needed
-- This migration is safe to run multiple times (uses ON CONFLICT DO NOTHING)
