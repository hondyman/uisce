-- ============================================================================
-- MIGRATION: Restructure business objects to have subtypes as separate rows
-- ============================================================================
-- Adds parent_id column to business_objects table
-- Migrates subtypes from JSONB config nested structure to separate rows
-- Reorganizes hierarchy: Parent business objects contain entity_fields in config,
-- Child rows (subtypes) contain subtype_fields in config with parent_id reference
-- Date: 2025-11-10

-- Step 1: Add parent_id column for subtype-parent relationships
ALTER TABLE public.business_objects 
ADD COLUMN parent_id uuid REFERENCES public.business_objects(id) ON DELETE CASCADE;

-- Step 2: Create index on parent_id for query performance
CREATE INDEX idx_business_objects_parent_id ON public.business_objects(parent_id);

-- Step 3: Migrate subtypes for Client Investor
DO $$ 
DECLARE
    v_parent_id uuid;
    v_tenant_id uuid;
BEGIN
    -- Get the parent Client Investor and its tenant
    SELECT id, tenant_id INTO v_parent_id, v_tenant_id
    FROM public.business_objects 
    WHERE name = 'Client Investor' AND parent_id IS NULL
    LIMIT 1;
    
    IF v_parent_id IS NOT NULL THEN
        -- Extract individual subtype and create as separate row
        INSERT INTO public.business_objects (
            tenant_id, name, display_name, description, icon, 
            parent_id, config, is_system
        ) 
        SELECT 
            v_tenant_id,
            'Individual Investor',
            'Individual Investor',
            'Investor profile for individual investors',
            NULL,
            v_parent_id,
            (
                SELECT config->'subtypes'->'individual' || 
                jsonb_build_object('entity_fields', COALESCE(config->'entity_fields', '[]'::jsonb))
                FROM public.business_objects WHERE id = v_parent_id
            ),
            true
        WHERE NOT EXISTS (
            SELECT 1 FROM public.business_objects 
            WHERE name = 'Individual Investor' AND parent_id = v_parent_id
        );
        
        -- Extract institutional subtype and create as separate row
        INSERT INTO public.business_objects (
            tenant_id, name, display_name, description, icon,
            parent_id, config, is_system
        )
        SELECT 
            v_tenant_id,
            'Institutional Investor',
            'Institutional Investor',
            'Investor profile for institutional investors',
            NULL,
            v_parent_id,
            (
                SELECT config->'subtypes'->'institutional' || 
                jsonb_build_object('entity_fields', COALESCE(config->'entity_fields', '[]'::jsonb))
                FROM public.business_objects WHERE id = v_parent_id
            ),
            true
        WHERE NOT EXISTS (
            SELECT 1 FROM public.business_objects 
            WHERE name = 'Institutional Investor' AND parent_id = v_parent_id
        );
        
        -- Update parent config to remove subtypes (keep only entity_fields)
        UPDATE public.business_objects 
        SET config = jsonb_build_object(
            'technical_name', config->'technical_name',
            'category', config->'category',
            'isCore', config->'isCore',
            'entity_fields', config->'entity_fields'
        )
        WHERE id = v_parent_id;
    END IF;
END $$;

-- Step 4: Migrate subtypes for Customer
DO $$
DECLARE
    v_parent_id uuid;
    v_tenant_id uuid;
BEGIN
    SELECT id, tenant_id INTO v_parent_id, v_tenant_id
    FROM public.business_objects 
    WHERE name = 'Customer' AND parent_id IS NULL
    LIMIT 1;
    
    IF v_parent_id IS NOT NULL THEN
        -- Extract retail_customer subtype
        INSERT INTO public.business_objects (
            tenant_id, name, display_name, description, icon,
            parent_id, config, is_system
        )
        SELECT 
            v_tenant_id,
            'Retail Customer',
            'Retail Customer',
            'Customer profile for retail customers',
            NULL,
            v_parent_id,
            (
                SELECT config->'subtypes'->'retail_customer' || 
                jsonb_build_object('entity_fields', COALESCE(config->'entity_fields', '[]'::jsonb))
                FROM public.business_objects WHERE id = v_parent_id
            ),
            true
        WHERE NOT EXISTS (
            SELECT 1 FROM public.business_objects 
            WHERE name = 'Retail Customer' AND parent_id = v_parent_id
        );
        
        -- Extract industry_customer subtype
        INSERT INTO public.business_objects (
            tenant_id, name, display_name, description, icon,
            parent_id, config, is_system
        )
        SELECT 
            v_tenant_id,
            'Industry Customer',
            'Industry Customer',
            'Customer profile for industry customers',
            NULL,
            v_parent_id,
            (
                SELECT config->'subtypes'->'industry_customer' || 
                jsonb_build_object('entity_fields', COALESCE(config->'entity_fields', '[]'::jsonb))
                FROM public.business_objects WHERE id = v_parent_id
            ),
            true
        WHERE NOT EXISTS (
            SELECT 1 FROM public.business_objects 
            WHERE name = 'Industry Customer' AND parent_id = v_parent_id
        );
        
        -- Extract government_customer subtype
        INSERT INTO public.business_objects (
            tenant_id, name, display_name, description, icon,
            parent_id, config, is_system
        )
        SELECT 
            v_tenant_id,
            'Government Customer',
            'Government Customer',
            'Customer profile for government customers',
            NULL,
            v_parent_id,
            (
                SELECT config->'subtypes'->'government_customer' || 
                jsonb_build_object('entity_fields', COALESCE(config->'entity_fields', '[]'::jsonb))
                FROM public.business_objects WHERE id = v_parent_id
            ),
            true
        WHERE NOT EXISTS (
            SELECT 1 FROM public.business_objects 
            WHERE name = 'Government Customer' AND parent_id = v_parent_id
        );
        
        -- Update parent config to remove subtypes
        UPDATE public.business_objects 
        SET config = jsonb_build_object(
            'technical_name', config->'technical_name',
            'category', config->'category',
            'isCore', config->'isCore',
            'entity_fields', config->'entity_fields'
        )
        WHERE id = v_parent_id;
    END IF;
END $$;

-- Step 5: Migrate subtypes for Portfolio
DO $$
DECLARE
    v_parent_id uuid;
    v_tenant_id uuid;
BEGIN
    SELECT id, tenant_id INTO v_parent_id, v_tenant_id
    FROM public.business_objects 
    WHERE name = 'Portfolio' AND parent_id IS NULL
    LIMIT 1;
    
    IF v_parent_id IS NOT NULL THEN
        -- Extract discretionary subtype
        INSERT INTO public.business_objects (
            tenant_id, name, display_name, description, icon,
            parent_id, config, is_system
        )
        SELECT 
            v_tenant_id,
            'Discretionary Portfolio',
            'Discretionary Portfolio',
            'Portfolio managed at discretion of advisor',
            NULL,
            v_parent_id,
            (
                SELECT config->'subtypes'->'discretionary' || 
                jsonb_build_object('entity_fields', COALESCE(config->'entity_fields', '[]'::jsonb))
                FROM public.business_objects WHERE id = v_parent_id
            ),
            true
        WHERE NOT EXISTS (
            SELECT 1 FROM public.business_objects 
            WHERE name = 'Discretionary Portfolio' AND parent_id = v_parent_id
        );
        
        -- Update parent config to remove subtypes
        UPDATE public.business_objects 
        SET config = jsonb_build_object(
            'technical_name', config->'technical_name',
            'category', config->'category',
            'isCore', config->'isCore',
            'entity_fields', config->'entity_fields'
        )
        WHERE id = v_parent_id;
    END IF;
END $$;

-- Step 6: Migrate subtypes for Trade
DO $$
DECLARE
    v_parent_id uuid;
    v_tenant_id uuid;
BEGIN
    SELECT id, tenant_id INTO v_parent_id, v_tenant_id
    FROM public.business_objects 
    WHERE name = 'Trade' AND parent_id IS NULL
    LIMIT 1;
    
    IF v_parent_id IS NOT NULL THEN
        -- Extract regular subtype
        INSERT INTO public.business_objects (
            tenant_id, name, display_name, description, icon,
            parent_id, config, is_system
        )
        SELECT 
            v_tenant_id,
            'Regular Trade',
            'Regular Trade',
            'Standard security transaction',
            NULL,
            v_parent_id,
            (
                SELECT config->'subtypes'->'regular' || 
                jsonb_build_object('entity_fields', COALESCE(config->'entity_fields', '[]'::jsonb))
                FROM public.business_objects WHERE id = v_parent_id
            ),
            true
        WHERE NOT EXISTS (
            SELECT 1 FROM public.business_objects 
            WHERE name = 'Regular Trade' AND parent_id = v_parent_id
        );
        
        -- Extract block_trade subtype
        INSERT INTO public.business_objects (
            tenant_id, name, display_name, description, icon,
            parent_id, config, is_system
        )
        SELECT 
            v_tenant_id,
            'Block Trade',
            'Block Trade',
            'Large block security transaction',
            NULL,
            v_parent_id,
            (
                SELECT config->'subtypes'->'block_trade' || 
                jsonb_build_object('entity_fields', COALESCE(config->'entity_fields', '[]'::jsonb))
                FROM public.business_objects WHERE id = v_parent_id
            ),
            true
        WHERE NOT EXISTS (
            SELECT 1 FROM public.business_objects 
            WHERE name = 'Block Trade' AND parent_id = v_parent_id
        );
        
        -- Update parent config to remove subtypes
        UPDATE public.business_objects 
        SET config = jsonb_build_object(
            'technical_name', config->'technical_name',
            'category', config->'category',
            'isCore', config->'isCore',
            'entity_fields', config->'entity_fields'
        )
        WHERE id = v_parent_id;
    END IF;
END $$;
