-- ============================================================================
-- MIGRATION: Consolidate Business Objects in PostgreSQL
-- ============================================================================
-- This migration consolidates all business objects into the business_objects table
-- Updated to match the actual schema: uses 'name' instead of 'key'
-- Date: 2025-11-10

BEGIN;

-- ============================================================================
-- Get default tenant (or create one if needed)
-- ============================================================================
INSERT INTO public.tenants (id, name, display_name, created_at)
VALUES (
  '00000000-0000-0000-0000-000000000001'::uuid,
  'Default Tenant',
  'Default Tenant',
  now()
) ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- Insert Business Objects
-- ============================================================================

-- Client Investor Business Object
INSERT INTO public.business_objects (
    tenant_id,
    name,
    display_name,
    description,
    icon,
    config,
    is_system
) VALUES (
    '00000000-0000-0000-0000-000000000001'::uuid,
    'Client Investor',
    'Client Investor',
    'Core BO: Investor profile with relationship tracking',
    'user-circle',
    jsonb_build_object(
        'technical_name', 'client_investor',
        'category', 'Customer & Relationships',
        'isCore', true,
        'entity_fields', jsonb_build_array(
            jsonb_build_object('key', 'investor_id', 'name', 'Investor ID', 'businessName', 'Investor ID', 'technicalName', 'investor_id', 'type', 'text', 'isCore', true),
            jsonb_build_object('key', 'legal_name', 'name', 'Legal Name', 'businessName', 'Legal Name', 'technicalName', 'legal_name', 'type', 'text', 'isCore', true),
            jsonb_build_object('key', 'email', 'name', 'Email', 'businessName', 'Email', 'technicalName', 'email', 'type', 'text', 'isCore', true),
            jsonb_build_object('key', 'phone', 'name', 'Phone', 'businessName', 'Phone', 'technicalName', 'phone', 'type', 'text', 'isCore', true),
            jsonb_build_object('key', 'aum', 'name', 'AUM', 'businessName', 'AUM', 'technicalName', 'aum', 'type', 'number', 'isCore', true)
        ),
        'subtypes', jsonb_build_object(
            'individual', jsonb_build_object(
                'name', 'Individual Investor',
                'businessName', 'Individual Investor',
                'technicalName', 'individual_investor',
                'isCore', true,
                'subtype_fields', jsonb_build_array(
                    jsonb_build_object('key', 'ssn', 'name', 'SSN', 'businessName', 'SSN', 'technicalName', 'ssn', 'type', 'text', 'isCore', true),
                    jsonb_build_object('key', 'date_of_birth', 'name', 'Date of Birth', 'businessName', 'Date of Birth', 'technicalName', 'date_of_birth', 'type', 'date', 'isCore', true)
                )
            ),
            'institutional', jsonb_build_object(
                'name', 'Institutional Investor',
                'businessName', 'Institutional Investor',
                'technicalName', 'institutional_investor',
                'isCore', true,
                'subtype_fields', jsonb_build_array(
                    jsonb_build_object('key', 'ein', 'name', 'EIN', 'businessName', 'EIN', 'technicalName', 'ein', 'type', 'text', 'isCore', true),
                    jsonb_build_object('key', 'registration_status', 'name', 'Registration Status', 'businessName', 'Registration Status', 'technicalName', 'registration_status', 'type', 'text', 'isCore', true)
                )
            )
        )
    ),
    true
) ON CONFLICT (name, tenant_id) DO UPDATE SET
    description = EXCLUDED.description,
    config = EXCLUDED.config,
    updated_at = now();

-- Customer Business Object
INSERT INTO public.business_objects (
    tenant_id,
    name,
    display_name,
    description,
    icon,
    config,
    is_system
) VALUES (
    '00000000-0000-0000-0000-000000000001'::uuid,
    'Customer',
    'Customer',
    'Core BO: Customer profile and segmentation',
    'users',
    jsonb_build_object(
        'technical_name', 'customer',
        'category', 'Customer & Relationships',
        'isCore', true,
        'entity_fields', jsonb_build_array(
            jsonb_build_object('key', 'customer_id', 'name', 'Customer ID', 'businessName', 'Customer ID', 'technicalName', 'customer_id', 'type', 'text', 'isCore', true),
            jsonb_build_object('key', 'customer_name', 'name', 'Customer Name', 'businessName', 'Customer Name', 'technicalName', 'customer_name', 'type', 'text', 'isCore', true),
            jsonb_build_object('key', 'email', 'name', 'Email', 'businessName', 'Email', 'technicalName', 'email', 'type', 'text', 'isCore', true),
            jsonb_build_object('key', 'phone', 'name', 'Phone', 'businessName', 'Phone', 'technicalName', 'phone', 'type', 'text', 'isCore', true),
            jsonb_build_object('key', 'status', 'name', 'Status', 'businessName', 'Status', 'technicalName', 'status', 'type', 'text', 'isCore', true)
        ),
        'subtypes', jsonb_build_object(
            'retail_customer', jsonb_build_object(
                'name', 'Retail Customer',
                'businessName', 'Retail Customer',
                'technicalName', 'retail_customer',
                'isCore', true,
                'subtype_fields', jsonb_build_array(
                    jsonb_build_object('key', 'loyalty_tier', 'name', 'Loyalty Tier', 'businessName', 'Loyalty Tier', 'technicalName', 'loyalty_tier', 'type', 'text', 'isCore', true),
                    jsonb_build_object('key', 'annual_spend', 'name', 'Annual Spend', 'businessName', 'Annual Spend', 'technicalName', 'annual_spend', 'type', 'number', 'isCore', true)
                )
            ),
            'industry_customer', jsonb_build_object(
                'name', 'Industry Customer',
                'businessName', 'Industry Customer',
                'technicalName', 'industry_customer',
                'isCore', true,
                'subtype_fields', jsonb_build_array(
                    jsonb_build_object('key', 'industry_sector', 'name', 'Industry Sector', 'businessName', 'Industry Sector', 'technicalName', 'industry_sector', 'type', 'text', 'isCore', true),
                    jsonb_build_object('key', 'company_size', 'name', 'Company Size', 'businessName', 'Company Size', 'technicalName', 'company_size', 'type', 'text', 'isCore', true)
                )
            ),
            'government_customer', jsonb_build_object(
                'name', 'Government Customer',
                'businessName', 'Government Customer',
                'technicalName', 'government_customer',
                'isCore', true,
                'subtype_fields', jsonb_build_array(
                    jsonb_build_object('key', 'agency_code', 'name', 'Agency Code', 'businessName', 'Agency Code', 'technicalName', 'agency_code', 'type', 'text', 'isCore', true),
                    jsonb_build_object('key', 'contract_type', 'name', 'Contract Type', 'businessName', 'Contract Type', 'technicalName', 'contract_type', 'type', 'text', 'isCore', true)
                )
            )
        )
    ),
    true
) ON CONFLICT (name, tenant_id) DO UPDATE SET
    description = EXCLUDED.description,
    config = EXCLUDED.config,
    updated_at = now();

-- Portfolio Business Object
INSERT INTO public.business_objects (
    tenant_id,
    name,
    display_name,
    description,
    icon,
    config,
    is_system
) VALUES (
    '00000000-0000-0000-0000-000000000001'::uuid,
    'Portfolio',
    'Portfolio',
    'Core BO: Asset portfolio management',
    'briefcase',
    jsonb_build_object(
        'technical_name', 'portfolio',
        'category', 'Financial Assets',
        'isCore', true,
        'entity_fields', jsonb_build_array(
            jsonb_build_object('key', 'portfolio_id', 'name', 'Portfolio ID', 'businessName', 'Portfolio ID', 'technicalName', 'portfolio_id', 'type', 'text', 'isCore', true),
            jsonb_build_object('key', 'portfolio_name', 'name', 'Portfolio Name', 'businessName', 'Portfolio Name', 'technicalName', 'portfolio_name', 'type', 'text', 'isCore', true),
            jsonb_build_object('key', 'inception_date', 'name', 'Inception Date', 'businessName', 'Inception Date', 'technicalName', 'inception_date', 'type', 'date', 'isCore', true),
            jsonb_build_object('key', 'total_value', 'name', 'Total Value', 'businessName', 'Total Value', 'technicalName', 'total_value', 'type', 'number', 'isCore', true)
        ),
        'subtypes', jsonb_build_object(
            'discretionary', jsonb_build_object(
                'name', 'Discretionary Portfolio',
                'businessName', 'Discretionary Portfolio',
                'technicalName', 'discretionary_portfolio',
                'isCore', true,
                'subtype_fields', jsonb_build_array(
                    jsonb_build_object('key', 'advisor_controlled', 'name', 'Advisor Controlled', 'businessName', 'Advisor Controlled', 'technicalName', 'advisor_controlled', 'type', 'boolean', 'isCore', true)
                )
            )
        )
    ),
    true
) ON CONFLICT (name, tenant_id) DO UPDATE SET
    description = EXCLUDED.description,
    config = EXCLUDED.config,
    updated_at = now();

-- Trade Business Object
INSERT INTO public.business_objects (
    tenant_id,
    name,
    display_name,
    description,
    icon,
    config,
    is_system
) VALUES (
    '00000000-0000-0000-0000-000000000001'::uuid,
    'Trade',
    'Trade',
    'Core BO: Security transaction',
    'trending-up',
    jsonb_build_object(
        'technical_name', 'trade',
        'category', 'Financial Transactions',
        'isCore', true,
        'entity_fields', jsonb_build_array(
            jsonb_build_object('key', 'trade_id', 'name', 'Trade ID', 'businessName', 'Trade ID', 'technicalName', 'trade_id', 'type', 'text', 'isCore', true),
            jsonb_build_object('key', 'trade_date', 'name', 'Trade Date', 'businessName', 'Trade Date', 'technicalName', 'trade_date', 'type', 'date', 'isCore', true),
            jsonb_build_object('key', 'ticker', 'name', 'Ticker', 'businessName', 'Ticker', 'technicalName', 'ticker', 'type', 'text', 'isCore', true),
            jsonb_build_object('key', 'quantity', 'name', 'Quantity', 'businessName', 'Quantity', 'technicalName', 'quantity', 'type', 'number', 'isCore', true),
            jsonb_build_object('key', 'price', 'name', 'Price', 'businessName', 'Price', 'technicalName', 'price', 'type', 'number', 'isCore', true)
        ),
        'subtypes', jsonb_build_object(
            'regular', jsonb_build_object(
                'name', 'Regular Trade',
                'businessName', 'Regular Trade',
                'technicalName', 'regular_trade',
                'isCore', true,
                'subtype_fields', jsonb_build_array(
                    jsonb_build_object('key', 'settlement_date', 'name', 'Settlement Date', 'businessName', 'Settlement Date', 'technicalName', 'settlement_date', 'type', 'date', 'isCore', true)
                )
            ),
            'block_trade', jsonb_build_object(
                'name', 'Block Trade',
                'businessName', 'Block Trade',
                'technicalName', 'block_trade',
                'isCore', true,
                'subtype_fields', jsonb_build_array(
                    jsonb_build_object('key', 'block_size', 'name', 'Block Size', 'businessName', 'Block Size', 'technicalName', 'block_size', 'type', 'number', 'isCore', true),
                    jsonb_build_object('key', 'negotiated_price', 'name', 'Negotiated Price', 'businessName', 'Negotiated Price', 'technicalName', 'negotiated_price', 'type', 'boolean', 'isCore', true)
                )
            )
        )
    ),
    true
) ON CONFLICT (name, tenant_id) DO UPDATE SET
    description = EXCLUDED.description,
    config = EXCLUDED.config,
    updated_at = now();

-- ============================================================================
-- Verification Queries
-- ============================================================================

-- Show inserted business objects
RAISE NOTICE '========================================';
RAISE NOTICE 'Consolidated Business Objects in PostgreSQL';
RAISE NOTICE '========================================';

SELECT 
    name,
    display_name,
    config->>'technical_name' as technical_name,
    config->>'category' as category
FROM public.business_objects 
WHERE name IN ('Client Investor', 'Customer', 'Portfolio', 'Trade')
AND tenant_id = '00000000-0000-0000-0000-000000000001'::uuid
ORDER BY name;

COMMIT;

-- ============================================================================
-- Summary
-- ============================================================================
-- All business objects have been consolidated into the business_objects table
-- Each object stores its complete schema (fields, subtypes, metadata) in the config JSONB column
-- This allows unified management of all business objects in a single table
