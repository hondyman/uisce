-- ============================================================================
-- MIGRATION: Consolidate Business Objects (Client Investor, Customer, Portfolio, Trade)
-- ============================================================================
-- This migration consolidates all business objects into the business_objects table
-- Date: 2025-11-10
-- Purpose: Unify all business object definitions in PostgreSQL

BEGIN;

-- Get the default tenant ID (assuming you have one)
WITH default_tenant AS (
  SELECT id FROM public.tenants LIMIT 1
)
INSERT INTO public.business_objects (
    tenant_id,
    key,
    name,
    display_name,
    technical_name,
    description,
    icon,
    is_core,
    category,
    created_by,
    last_modified_by
) VALUES
-- Client Investor Business Object
(
    (SELECT id FROM default_tenant),
    'client_investor',
    'Client Investor',
    'Client Investor',
    'client_investor',
    'Core BO: Investor profile with relationship tracking',
    'user-circle',
    true,
    'Customer & Relationships',
    NULL,
    NULL
),
-- Customer Business Object
(
    (SELECT id FROM default_tenant),
    'customer',
    'Customer',
    'Customer',
    'customer',
    'Core BO: Customer profile and segmentation',
    'users',
    true,
    'Customer & Relationships',
    NULL,
    NULL
),
-- Portfolio Business Object
(
    (SELECT id FROM default_tenant),
    'portfolio',
    'Portfolio',
    'Portfolio',
    'portfolio',
    'Core BO: Asset portfolio management',
    'briefcase',
    true,
    'Financial Assets',
    NULL,
    NULL
),
-- Trade Business Object
(
    (SELECT id FROM default_tenant),
    'trade',
    'Trade',
    'Trade',
    'trade',
    'Core BO: Security transaction',
    'trending-up',
    true,
    'Financial Transactions',
    NULL,
    NULL
)
ON CONFLICT (tenant_id, key) DO UPDATE SET
    name = EXCLUDED.name,
    display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    last_modified_at = now();

-- ============================================================================
-- Insert Subtypes for Client Investor
-- ============================================================================
WITH default_tenant AS (
  SELECT id FROM public.tenants LIMIT 1
),
investor_bo AS (
  SELECT id FROM public.business_objects 
  WHERE key = 'client_investor' 
  AND tenant_id = (SELECT id FROM default_tenant)
)
INSERT INTO public.bo_subtypes (
    business_object_id,
    tenant_id,
    key,
    name,
    display_name,
    technical_name,
    description,
    is_core,
    sequence,
    created_by,
    last_modified_by
) VALUES
(
    (SELECT id FROM investor_bo),
    (SELECT id FROM default_tenant),
    'individual',
    'Individual Investor',
    'Individual Investor',
    'individual_investor',
    'Individual investor profile',
    true,
    1,
    NULL,
    NULL
),
(
    (SELECT id FROM investor_bo),
    (SELECT id FROM default_tenant),
    'institutional',
    'Institutional Investor',
    'Institutional Investor',
    'institutional_investor',
    'Institutional investor profile',
    true,
    2,
    NULL,
    NULL
)
ON CONFLICT (tenant_id, business_object_id, key) DO NOTHING;

-- ============================================================================
-- Insert Subtypes for Customer
-- ============================================================================
WITH default_tenant AS (
  SELECT id FROM public.tenants LIMIT 1
),
customer_bo AS (
  SELECT id FROM public.business_objects 
  WHERE key = 'customer' 
  AND tenant_id = (SELECT id FROM default_tenant)
)
INSERT INTO public.bo_subtypes (
    business_object_id,
    tenant_id,
    key,
    name,
    display_name,
    technical_name,
    description,
    is_core,
    sequence,
    created_by,
    last_modified_by
) VALUES
(
    (SELECT id FROM customer_bo),
    (SELECT id FROM default_tenant),
    'retail_customer',
    'Retail Customer',
    'Retail Customer',
    'retail_customer',
    'Retail customer segment',
    true,
    1,
    NULL,
    NULL
),
(
    (SELECT id FROM customer_bo),
    (SELECT id FROM default_tenant),
    'industry_customer',
    'Industry Customer',
    'Industry Customer',
    'industry_customer',
    'Industry/B2B customer segment',
    true,
    2,
    NULL,
    NULL
),
(
    (SELECT id FROM customer_bo),
    (SELECT id FROM default_tenant),
    'government_customer',
    'Government Customer',
    'Government Customer',
    'government_customer',
    'Government entity customer segment',
    true,
    3,
    NULL,
    NULL
)
ON CONFLICT (tenant_id, business_object_id, key) DO NOTHING;

-- ============================================================================
-- Insert Subtypes for Portfolio
-- ============================================================================
WITH default_tenant AS (
  SELECT id FROM public.tenants LIMIT 1
),
portfolio_bo AS (
  SELECT id FROM public.business_objects 
  WHERE key = 'portfolio' 
  AND tenant_id = (SELECT id FROM default_tenant)
)
INSERT INTO public.bo_subtypes (
    business_object_id,
    tenant_id,
    key,
    name,
    display_name,
    technical_name,
    description,
    is_core,
    sequence,
    created_by,
    last_modified_by
) VALUES
(
    (SELECT id FROM portfolio_bo),
    (SELECT id FROM default_tenant),
    'discretionary',
    'Discretionary Portfolio',
    'Discretionary Portfolio',
    'discretionary_portfolio',
    'Advisor-managed discretionary portfolio',
    true,
    1,
    NULL,
    NULL
)
ON CONFLICT (tenant_id, business_object_id, key) DO NOTHING;

-- ============================================================================
-- Insert Subtypes for Trade
-- ============================================================================
WITH default_tenant AS (
  SELECT id FROM public.tenants LIMIT 1
),
trade_bo AS (
  SELECT id FROM public.business_objects 
  WHERE key = 'trade' 
  AND tenant_id = (SELECT id FROM default_tenant)
)
INSERT INTO public.bo_subtypes (
    business_object_id,
    tenant_id,
    key,
    name,
    display_name,
    technical_name,
    description,
    is_core,
    sequence,
    created_by,
    last_modified_by
) VALUES
(
    (SELECT id FROM trade_bo),
    (SELECT id FROM default_tenant),
    'regular',
    'Regular Trade',
    'Regular Trade',
    'regular_trade',
    'Standard security trade',
    true,
    1,
    NULL,
    NULL
),
(
    (SELECT id FROM trade_bo),
    (SELECT id FROM default_tenant),
    'block_trade',
    'Block Trade',
    'Block Trade',
    'block_trade',
    'Large block security trade',
    true,
    2,
    NULL,
    NULL
)
ON CONFLICT (tenant_id, business_object_id, key) DO NOTHING;

-- ============================================================================
-- Insert Fields for Client Investor (Entity Level)
-- ============================================================================
WITH default_tenant AS (
  SELECT id FROM public.tenants LIMIT 1
),
investor_bo AS (
  SELECT id FROM public.business_objects 
  WHERE key = 'client_investor' 
  AND tenant_id = (SELECT id FROM default_tenant)
)
INSERT INTO public.bo_fields (
    tenant_id,
    business_object_id,
    key,
    name,
    display_name,
    technical_name,
    type,
    is_core,
    is_required,
    sequence
) VALUES
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM investor_bo),
    'investor_id',
    'Investor ID',
    'Investor ID',
    'investor_id',
    'text',
    true,
    true,
    1
),
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM investor_bo),
    'legal_name',
    'Legal Name',
    'Legal Name',
    'legal_name',
    'text',
    true,
    true,
    2
),
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM investor_bo),
    'email',
    'Email',
    'Email',
    'email',
    'email',
    true,
    false,
    3
),
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM investor_bo),
    'phone',
    'Phone',
    'Phone',
    'phone',
    'text',
    true,
    false,
    4
),
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM investor_bo),
    'aum',
    'AUM',
    'Assets Under Management',
    'aum',
    'currency',
    true,
    false,
    5
)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- Insert Fields for Customer (Entity Level)
-- ============================================================================
WITH default_tenant AS (
  SELECT id FROM public.tenants LIMIT 1
),
customer_bo AS (
  SELECT id FROM public.business_objects 
  WHERE key = 'customer' 
  AND tenant_id = (SELECT id FROM default_tenant)
)
INSERT INTO public.bo_fields (
    tenant_id,
    business_object_id,
    key,
    name,
    display_name,
    technical_name,
    type,
    is_core,
    is_required,
    sequence
) VALUES
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM customer_bo),
    'customer_id',
    'Customer ID',
    'Customer ID',
    'customer_id',
    'text',
    true,
    true,
    1
),
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM customer_bo),
    'customer_name',
    'Customer Name',
    'Customer Name',
    'customer_name',
    'text',
    true,
    true,
    2
),
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM customer_bo),
    'email',
    'Email',
    'Email',
    'email',
    'email',
    true,
    false,
    3
),
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM customer_bo),
    'phone',
    'Phone',
    'Phone',
    'phone',
    'text',
    true,
    false,
    4
),
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM customer_bo),
    'status',
    'Status',
    'Customer Status',
    'status',
    'text',
    true,
    false,
    5
)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- Insert Fields for Portfolio (Entity Level)
-- ============================================================================
WITH default_tenant AS (
  SELECT id FROM public.tenants LIMIT 1
),
portfolio_bo AS (
  SELECT id FROM public.business_objects 
  WHERE key = 'portfolio' 
  AND tenant_id = (SELECT id FROM default_tenant)
)
INSERT INTO public.bo_fields (
    tenant_id,
    business_object_id,
    key,
    name,
    display_name,
    technical_name,
    type,
    is_core,
    is_required,
    sequence
) VALUES
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM portfolio_bo),
    'portfolio_id',
    'Portfolio ID',
    'Portfolio ID',
    'portfolio_id',
    'text',
    true,
    true,
    1
),
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM portfolio_bo),
    'portfolio_name',
    'Portfolio Name',
    'Portfolio Name',
    'portfolio_name',
    'text',
    true,
    true,
    2
),
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM portfolio_bo),
    'inception_date',
    'Inception Date',
    'Inception Date',
    'inception_date',
    'date',
    true,
    false,
    3
),
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM portfolio_bo),
    'total_value',
    'Total Value',
    'Total Portfolio Value',
    'total_value',
    'currency',
    true,
    false,
    4
)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- Insert Fields for Trade (Entity Level)
-- ============================================================================
WITH default_tenant AS (
  SELECT id FROM public.tenants LIMIT 1
),
trade_bo AS (
  SELECT id FROM public.business_objects 
  WHERE key = 'trade' 
  AND tenant_id = (SELECT id FROM default_tenant)
)
INSERT INTO public.bo_fields (
    tenant_id,
    business_object_id,
    key,
    name,
    display_name,
    technical_name,
    type,
    is_core,
    is_required,
    sequence
) VALUES
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM trade_bo),
    'trade_id',
    'Trade ID',
    'Trade ID',
    'trade_id',
    'text',
    true,
    true,
    1
),
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM trade_bo),
    'trade_date',
    'Trade Date',
    'Trade Date',
    'trade_date',
    'date',
    true,
    true,
    2
),
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM trade_bo),
    'ticker',
    'Ticker',
    'Security Ticker',
    'ticker',
    'text',
    true,
    true,
    3
),
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM trade_bo),
    'quantity',
    'Quantity',
    'Trade Quantity',
    'quantity',
    'number',
    true,
    true,
    4
),
(
    (SELECT id FROM default_tenant),
    (SELECT id FROM trade_bo),
    'price',
    'Price',
    'Price per Unit',
    'price',
    'currency',
    true,
    true,
    5
)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- Summary Report
-- ============================================================================
-- Print summary of consolidation
DO $$
DECLARE
    v_tenant_id uuid;
    v_bo_count int;
    v_subtype_count int;
    v_field_count int;
BEGIN
    SELECT id INTO v_tenant_id FROM public.tenants LIMIT 1;
    
    SELECT COUNT(*) INTO v_bo_count FROM public.business_objects 
    WHERE tenant_id = v_tenant_id;
    
    SELECT COUNT(*) INTO v_subtype_count FROM public.bo_subtypes 
    WHERE tenant_id = v_tenant_id;
    
    SELECT COUNT(*) INTO v_field_count FROM public.bo_fields 
    WHERE tenant_id = v_tenant_id;
    
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Business Objects Consolidation Report';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Tenant ID: %', v_tenant_id;
    RAISE NOTICE 'Total Business Objects: %', v_bo_count;
    RAISE NOTICE 'Total Subtypes: %', v_subtype_count;
    RAISE NOTICE 'Total Fields: %', v_field_count;
    RAISE NOTICE '========================================';
END $$;

COMMIT;
