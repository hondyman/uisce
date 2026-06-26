-- ============================================================================
-- SEMANTIC MODEL INHERITANCE & BUSINESS OBJECT SYNC
-- Core models are templates; tenants always use custom models that extend core
-- ============================================================================

-- Add business object link to semantic cubes
ALTER TABLE semantic_cubes_v2 
ADD COLUMN IF NOT EXISTS business_object_id UUID REFERENCES business_objects(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS model_type TEXT DEFAULT 'custom' CHECK (model_type IN ('core', 'custom', 'override'));

CREATE INDEX IF NOT EXISTS idx_semantic_cubes_bo ON semantic_cubes_v2(business_object_id);
CREATE INDEX IF NOT EXISTS idx_semantic_cubes_type ON semantic_cubes_v2(model_type);

COMMENT ON COLUMN semantic_cubes_v2.business_object_id IS 'Links semantic cube to business object for sync';
COMMENT ON COLUMN semantic_cubes_v2.model_type IS 'core=template (never exposed), custom=tenant copy, override=tenant modified';

-- ============================================================================
-- CORE SEMANTIC MODELS (Templates - Never Directly Used)
-- ============================================================================

-- Create core semantic cube for Client Investor
INSERT INTO semantic_cubes_v2 (
    id, tenant_id, name, display_name, description, sql, status, is_system
)
SELECT 
    gen_random_uuid(),
    (SELECT id FROM tenants WHERE name = 'Default Tenant' LIMIT 1),
    'core_client_investor',
    'Client Investor (Core)',
    'Core semantic model for Client Investor - DO NOT USE DIRECTLY',
    'SELECT * FROM bo_instances',
    'draft',
    true
WHERE NOT EXISTS (SELECT 1 FROM semantic_cubes_v2 WHERE name = 'core_client_investor');

-- Core Individual Investor
INSERT INTO semantic_cubes_v2 (
    id, tenant_id, name, display_name, description, sql, status, is_system, source_cube_id
)
SELECT 
    gen_random_uuid(),
    (SELECT id FROM tenants WHERE name = 'Default Tenant' LIMIT 1),
    'core_individual_investor',
    'Individual Investor (Core)',
    'Core semantic model for Individual Investor - Extends Client Investor',
    'SELECT * FROM bo_instances',
    'draft',
    true,
    (SELECT id FROM semantic_cubes_v2 WHERE name = 'core_client_investor')
WHERE NOT EXISTS (SELECT 1 FROM semantic_cubes_v2 WHERE name = 'core_individual_investor');

-- Core Institutional Investor
INSERT INTO semantic_cubes_v2 (
    id, tenant_id, name, display_name, description, sql, status, is_system, source_cube_id
)
SELECT 
    gen_random_uuid(),
    (SELECT id FROM tenants WHERE name = 'Default Tenant' LIMIT 1),
    'core_institutional_investor',
    'Institutional Investor (Core)',
    'Core semantic model for Institutional Investor - Extends Client Investor',
    'SELECT * FROM bo_instances',
    'draft',
    true,
    (SELECT id FROM semantic_cubes_v2 WHERE name = 'core_client_investor')
WHERE NOT EXISTS (SELECT 1 FROM semantic_cubes_v2 WHERE name = 'core_institutional_investor');

-- ============================================================================
-- FUNCTION: Provision Tenant Semantic Model from Core
-- Creates a custom model for tenant that extends a core model
-- ============================================================================

CREATE OR REPLACE FUNCTION provision_tenant_semantic_model(
    p_tenant_id UUID,
    p_core_cube_id UUID,
    p_datasource_id UUID DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    v_new_cube_id UUID;
    v_core_cube RECORD;
    v_bo_id UUID;
BEGIN
    -- Get core cube details
    SELECT * INTO v_core_cube 
    FROM semantic_cubes_v2 
    WHERE id = p_core_cube_id AND model_type = 'core';
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Core cube not found: %', p_core_cube_id;
    END IF;
    
    -- Check if tenant already has this model
    SELECT id INTO v_new_cube_id
    FROM semantic_cubes_v2
    WHERE tenant_id = p_tenant_id 
      AND source_cube_id = p_core_cube_id
      AND model_type IN ('custom', 'override');
    
    IF FOUND THEN
        RETURN v_new_cube_id;
    END IF;
    
    -- Create new tenant cube extending core
    v_new_cube_id := gen_random_uuid();
    
    INSERT INTO semantic_cubes_v2 (
        id, tenant_id, name, label, description,
        sql_table, data_source, status,
        is_system, model_type, source_cube_id, business_object_id,
        datasource_id
    ) VALUES (
        v_new_cube_id,
        p_tenant_id,
        REPLACE(v_core_cube.name, 'core_', ''),
        REPLACE(v_core_cube.display_name, ' (Core)', ''),
        'Custom model extending: ' || v_core_cube.display_name,
        v_core_cube.sql_table,
        v_core_cube.data_source,
        'active',
        false,
        'custom',
        p_core_cube_id,
        v_core_cube.business_object_id,
        p_datasource_id
    );
    
    -- Copy dimensions from core (if dimensions table exists)
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'cube_dimensions_v2') THEN
      INSERT INTO cube_dimensions_v2 (cube_id, name, sql, label, type, is_primary_key, is_inherited)
      SELECT v_new_cube_id, name, sql, label, type, is_primary_key, true
      FROM cube_dimensions_v2
      WHERE cube_id = p_core_cube_id;
    END IF;

    -- Copy measures from core (if measures table exists)
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'cube_measures_v2') THEN
      INSERT INTO cube_measures_v2 (cube_id, name, sql, label, type, is_inherited)
      SELECT v_new_cube_id, name, sql, label, type, true
      FROM cube_measures_v2
      WHERE cube_id = p_core_cube_id;
    END IF;
    
    RETURN v_new_cube_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- FUNCTION: Sync Semantic Model with Business Object Fields
-- Automatically adds dimensions/measures when BO fields change
-- ============================================================================

CREATE OR REPLACE FUNCTION sync_semantic_model_with_bo(
    p_cube_id UUID
) RETURNS INTEGER AS $$
DECLARE
    v_cube RECORD;
    v_field RECORD;
    v_count INTEGER := 0;
BEGIN
    -- Get cube and linked BO
    SELECT c.*, bo.key as bo_key
    INTO v_cube
    FROM semantic_cubes_v2 c
    LEFT JOIN business_objects bo ON c.business_object_id = bo.id
    WHERE c.id = p_cube_id;
    
    IF NOT FOUND OR v_cube.business_object_id IS NULL THEN
        RETURN 0;
    END IF;
    
    -- Add dimensions for BO fields that don't exist yet (use dynamic SQL to avoid referencing missing tables at create-time)
    FOR v_field IN
        EXECUTE format($q$
            SELECT f.key, f.name, f.type, f.is_core
            FROM bo_fields f
            WHERE f.business_object_id = %L
              AND f.subtype_id IS NULL
              AND NOT EXISTS (
                SELECT 1 FROM cube_dimensions_v2 d WHERE d.cube_id = %L AND d.name = f.key
              )
        $q$, p_cube_id::text, p_cube_id::text)
    LOOP
        INSERT INTO cube_dimensions_v2 (cube_id, name, sql, label, type, is_inherited)
        VALUES (
            p_cube_id,
            v_field.key,
            'core_field_values->>' || quote_literal(v_field.key),
            v_field.name,
            CASE v_field.type
                WHEN 'date' THEN 'time'
                WHEN 'datetime' THEN 'time'
                WHEN 'boolean' THEN 'boolean'
                ELSE 'string'
            END,
            v_field.is_core
        );
        v_count := v_count + 1;
    END LOOP;
    
    RETURN v_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Add is_inherited flag to dimensions and measures
-- ============================================================================

ALTER TABLE IF EXISTS cube_dimensions_v2 
ADD COLUMN IF NOT EXISTS is_inherited BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS is_overridden BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS source_dimension_id UUID;

ALTER TABLE IF EXISTS cube_measures_v2 
ADD COLUMN IF NOT EXISTS is_inherited BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS is_overridden BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS source_measure_id UUID;

-- ============================================================================
-- SUCCESS MESSAGE
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '✓ Semantic model linked to business objects';
    RAISE NOTICE '✓ Added model_type (core/custom/override)';
    RAISE NOTICE '✓ Created provision_tenant_semantic_model() function';
    RAISE NOTICE '✓ Created sync_semantic_model_with_bo() function';
    RAISE NOTICE '✓ Added is_inherited/is_overridden to dimensions/measures';
    RAISE NOTICE '✓ Seeded core semantic models for Client/Individual/Institutional Investor';
END $$;
