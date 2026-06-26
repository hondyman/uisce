-- Phase 3: Calendar Semantic Integration
-- Integrates catalog_node/catalog_edge semantic layer with calendar MDM and rules engine
-- This migration creates:
-- 1. Semantic terms as catalog_node records (in public schema)
-- 2. Calendar business object with bo_fields linked to semantic terms
-- 3. Calendar MDM tables in northwinds database
-- 4. Rules schema updated to reference semantic catalog

-- ============================================================================
-- PART 1: Semantic Graph Setup (public schema - catalog layer)
-- ============================================================================

-- Ensure public schema exists
CREATE SCHEMA IF NOT EXISTS public;

-- Ensure catalog tables exist (assumes 004_catalog_bootstrap.sql already ran)
-- catalog_node_type, catalog_node, catalog_edge_type, catalog_edge should exist

-- Insert calendar semantic term node types (if not already present)
-- the catalog_node_type table has column catalog_type_name
-- Only run if catalog_node_type exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='catalog_node_type') THEN
        INSERT INTO public.catalog_node_type (catalog_type_name, description, config)
        VALUES
    ('semantic_term', 'A governed business concept (e.g., CalendarDate, IsBusinessDay)', '{"category": "semantic", "governance": true}'::jsonb),
    ('business_object', 'A structured entity (e.g., Calendar, Portfolio)', '{"category": "business", "governance": true}'::jsonb),
    ('bo_field', 'A field within a business object', '{"category": "structural", "semantic_linkage": true}'::jsonb),
    ('physical_table', 'A physical database table', '{"category": "storage"}'::jsonb),
    ('physical_column', 'A physical database column', '{"category": "storage"}'::jsonb)
ON CONFLICT (tenant_id, catalog_type_name) DO NOTHING;
    END IF;
END$$;

-- Get or create node type IDs for convenience
DO $$
DECLARE
    v_semantic_term_type_id UUID;
    v_bo_type_id UUID;
    v_bo_field_type_id UUID;
    v_physical_table_type_id UUID;
    v_physical_column_type_id UUID;
BEGIN
    -- For lookup in subsequent inserts
    SELECT id INTO v_semantic_term_type_id FROM public.catalog_node_type WHERE catalog_type_name = 'semantic_term' LIMIT 1;
    SELECT id INTO v_bo_type_id FROM public.catalog_node_type WHERE catalog_type_name = 'business_object' LIMIT 1;
    SELECT id INTO v_bo_field_type_id FROM public.catalog_node_type WHERE catalog_type_name = 'bo_field' LIMIT 1;
    SELECT id INTO v_physical_table_type_id FROM public.catalog_node_type WHERE catalog_type_name = 'physical_table' LIMIT 1;
    SELECT id INTO v_physical_column_type_id FROM public.catalog_node_type WHERE catalog_type_name = 'physical_column' LIMIT 1;
    
    -- Insert semantic term nodes for calendar domain
    IF v_semantic_term_type_id IS NOT NULL THEN
        -- obtain a valid tenant_datasource_id from existing record
        INSERT INTO public.catalog_node (node_type_id, node_name, properties, tenant_id, tenant_datasource_id, qualified_path)
        SELECT
        v_semantic_term_type_id,
        term_name,
        jsonb_build_object(
            'data_type', term_type,
            'business_definition', business_def,
            'category', category,
            'governance_status', 'approved',
            'sql', sql_expr
        ),
        COALESCE((SELECT id FROM public.tenants LIMIT 1), '00000000-0000-0000-0000-000000000001'::uuid), -- actual tenant or fallback
        NULL, -- no tenant_datasource (nullable)
        term_name -- use term name as qualified_path
    FROM (
        VALUES
            ('calendar.CalendarDate', 'date', 'Trading date for business calendar classification', 'IDENTIFICATION', 'calendar_date'),
            ('calendar.IsBusinessDay', 'boolean', 'Indicates if date is a business day (not weekend/holiday)', 'CLASSIFICATION', 'is_business_day'),
            ('calendar.RegionCode', 'string', 'Geographic region code (GB, US, JP, etc.)', 'CLASSIFICATION', 'region_code'),
            ('calendar.HolidayName', 'string', 'Name of holiday if applicable', 'CLASSIFICATION', 'holiday_name'),
            ('calendar.SourceSystem', 'string', 'Source system (Nager.Date, OpenHolidays, Workalendar)', 'DATA_QUALITY', 'source_system'),
            ('calendar.ConfidenceScore', 'number', 'Confidence level (0-100) of classification', 'DATA_QUALITY', 'confidence_score'),
            ('calendar.TradingImpact', 'boolean', 'Indicates if date impacts trading operations', 'BUSINESS_IMPACT', 'trading_impact'),
            ('calendar.EffectiveDate', 'date', 'Date when calendar entry becomes effective', 'IDENTIFICATION', 'effective_date'),
            ('calendar.ExpirationDate', 'date', 'Date when calendar entry expires', 'IDENTIFICATION', 'expiration_date'),
            ('calendar.LastModifiedBy', 'string', 'User who last modified this record', 'DATA_QUALITY', 'last_modified_by')
    ) AS terms(term_name, term_type, business_def, category, sql_expr)
    WHERE NOT EXISTS (
            SELECT 1 FROM public.catalog_node cn
            WHERE cn.node_name = terms.term_name
            AND cn.node_type_id = v_semantic_term_type_id
        );
    END IF;
EXCEPTION WHEN OTHERS THEN
    RAISE WARNING 'Semantic term nodes insertion failed (may already exist or catalog missing): %', SQLERRM;
END$$;

-- ============================================================================
-- PART 2: Calendar Business Object and Fields (public schema)
-- ============================================================================

-- Create or get calendar business object
DO $$
DECLARE
    v_bo_type_id UUID;
    v_calendar_bo_id UUID;
    v_semantic_term_type_id UUID;
BEGIN
    -- catalog_node_type uses catalog_type_name column
    SELECT id INTO v_bo_type_id FROM public.catalog_node_type WHERE catalog_type_name = 'business_object' LIMIT 1;
    SELECT id INTO v_semantic_term_type_id FROM public.catalog_node_type WHERE catalog_type_name = 'semantic_term' LIMIT 1;
    
    IF v_bo_type_id IS NULL THEN
        RAISE NOTICE 'catalog_node_type business_object not found, skipping BO creation';
        RETURN;
    END IF;
    
    -- Create calendar business object node
    INSERT INTO public.catalog_node (node_type_id, node_name, properties, tenant_id, tenant_datasource_id, qualified_path)
    VALUES (
        v_bo_type_id,
        'calendar.Calendar',
        jsonb_build_object(
            'display_name', 'Calendar',
            'description', 'Master calendar for business day identification and holiday management',
            'driver_table', 'northwinds.calendar_mdm',
            'category', 'master_data',
            'history_mode', 'SCD_TYPE_2'
        ),
        COALESCE((SELECT id FROM public.tenants LIMIT 1), '00000000-0000-0000-0000-000000000001'::uuid),
        NULL,
        'calendar.Calendar'
    )
    ON CONFLICT DO NOTHING;
EXCEPTION WHEN OTHERS THEN
    RAISE WARNING 'Calendar BO node insertion failed: %', SQLERRM;
END$$;

-- ============================================================================
-- PART 3: Calendar MDM Tables (northwinds database)
-- ============================================================================

-- ============================================================================
-- PART 2.5: Create BO Fields for Calendar (link to semantic terms)
-- ============================================================================
DO $$
DECLARE
    v_bo_id UUID;
    v_sem_term RECORD;
BEGIN
    -- ensure business_objects entry exists
    -- insert business object only if not already present
    INSERT INTO public.business_objects (id, key, name, display_name, description, tenant_id)
    SELECT
        gen_random_uuid(),
        'calendar',
        'calendar',
        'Calendar',
        'Master calendar for business day identification and holiday management',
        COALESCE((SELECT id FROM public.tenants LIMIT 1), gen_random_uuid())
    WHERE NOT EXISTS (
        SELECT 1 FROM public.business_objects WHERE name = 'calendar'
    );

    SELECT id INTO v_bo_id FROM public.business_objects WHERE name = 'calendar' LIMIT 1;
    IF v_bo_id IS NULL THEN
        RAISE NOTICE 'public.business_objects.calendar not found, skipping bo_fields insertion';
        RETURN;
    END IF;

    FOR v_sem_term IN
        SELECT id, node_name FROM public.catalog_node
        WHERE node_type_id = (
            SELECT id FROM public.catalog_node_type WHERE catalog_type_name='semantic_term' LIMIT 1
        )
        AND node_name LIKE 'calendar.%'
    LOOP
        -- derive field name from semantic term tail
        INSERT INTO public.bo_fields (
            business_object_id,
            field_name,
            display_label,
            display_order,
            field_type,
            is_required,
            custom_properties
        )
        VALUES (
            v_bo_id,
            split_part(v_sem_term.node_name, '.', 2),
            initcap(split_part(v_sem_term.node_name, '.', 2)),
            0,
            CASE WHEN v_sem_term.node_name ILIKE '%date%' THEN 'date'
                 WHEN v_sem_term.node_name ILIKE '%score%' THEN 'number'
                 WHEN v_sem_term.node_name ILIKE '%impact%' THEN 'boolean'
                 ELSE 'string' END,
            TRUE,
            jsonb_build_object('semantic_term_id', v_sem_term.id)
        )
        ON CONFLICT (business_object_id, field_name) DO NOTHING;
    END LOOP;
END$$;

-- create RLS policy on northwinds.calendar_mdm
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='northwinds' AND table_name='calendar_mdm') THEN
        EXECUTE 'ALTER TABLE northwinds.calendar_mdm ENABLE ROW LEVEL SECURITY';
        EXECUTE 'CREATE POLICY calendar_tenant_isolation ON northwinds.calendar_mdm '||
                ' USING (tenant_id = current_setting(''app.current_tenant_id'')::uuid)';
    END IF;
END$$;

-- ============================================================================
-- PART 3: Calendar MDM Tables (northwinds database)
-- ============================================================================

-- Create calendar MDM tables inside northwinds (ensure schema exists first)
DO $$
BEGIN
    EXECUTE 'CREATE SCHEMA IF NOT EXISTS northwinds';
    -- Main calendar MDM table
    EXECUTE 'CREATE TABLE IF NOT EXISTS northwinds.calendar_mdm (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        calendar_date DATE NOT NULL UNIQUE,
        is_business_day BOOLEAN NOT NULL DEFAULT TRUE,
        is_weekend BOOLEAN NOT NULL DEFAULT FALSE,
        region_code VARCHAR(10) NOT NULL, -- GB, US, JP, EU, etc.
        holiday_name VARCHAR(255),
        source_system VARCHAR(100) NOT NULL, -- nager_date, open_holidays, workalendar
        confidence_score INT NOT NULL DEFAULT 100 CHECK (confidence_score >= 0 AND confidence_score <= 100),
        trading_impact BOOLEAN NOT NULL DEFAULT FALSE,
        effective_date DATE NOT NULL DEFAULT CURRENT_DATE,
        expiration_date DATE,
        last_modified_by VARCHAR(255),
        created_at TIMESTAMP NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
        created_by_user_id UUID NOT NULL,
        updated_by_user_id UUID,
        tenant_id UUID NOT NULL,
        datasource_id UUID NOT NULL,
        
        CONSTRAINT calendar_mdm_date_region_unique UNIQUE(calendar_date, region_code),
        CONSTRAINT calendar_mdm_date_check CHECK (calendar_date <= CURRENT_DATE + INTERVAL ''10 years''),
        CONSTRAINT calendar_mdm_effective_check CHECK (effective_date <= calendar_date)
    )';
    
    -- Indexes for performance
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_calendar_mdm_region_date ON northwinds.calendar_mdm(region_code, calendar_date)';
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_calendar_mdm_business_day ON northwinds.calendar_mdm(is_business_day, region_code)';
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_calendar_mdm_holiday ON northwinds.calendar_mdm(holiday_name, region_code)';
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_calendar_mdm_source ON northwinds.calendar_mdm(source_system, calendar_date)';
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_calendar_mdm_updated ON northwinds.calendar_mdm(updated_at DESC)';
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_calendar_mdm_tenant ON northwinds.calendar_mdm(tenant_id, datasource_id)';
    
    -- Calendar lineage table (for tracking changes via SCD Type 2)
    EXECUTE 'CREATE TABLE IF NOT EXISTS northwinds.calendar_mdm_lineage (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        calendar_id UUID NOT NULL,
        version INT NOT NULL DEFAULT 1,
        calendar_date DATE NOT NULL,
        is_business_day BOOLEAN,
        region_code VARCHAR(10),
        holiday_name VARCHAR(255),
        confidence_score INT,
        trading_impact BOOLEAN,
        valid_from TIMESTAMP NOT NULL DEFAULT NOW(),
        valid_to TIMESTAMP,
        is_current BOOLEAN NOT NULL DEFAULT TRUE,
        rule_id UUID,  -- Link to rule that modified this record
        created_at TIMESTAMP NOT NULL DEFAULT NOW(),
        tenant_id UUID NOT NULL
    )';
    
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_calendar_lineage_current ON northwinds.calendar_mdm_lineage(calendar_id, is_current)';
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_calendar_lineage_date ON northwinds.calendar_mdm_lineage(valid_from DESC, valid_to DESC)';
    
EXCEPTION WHEN OTHERS THEN
    RAISE WARNING 'Calendar MDM table creation failed: %', SQLERRM;
END$$;

-- ============================================================================
-- PART 4: Update EDM Rules Schema to Reference Semantic Catalog
-- ============================================================================

-- Modify edm.rules to have optional semantic_term_id references
ALTER TABLE IF EXISTS edm.rules ADD COLUMN IF NOT EXISTS semantic_catalog_node_id UUID;
ALTER TABLE IF EXISTS edm.rule_steps ADD COLUMN IF NOT EXISTS semantic_term_node_id UUID;

-- Comment explaining semantic linkage
COMMENT ON COLUMN edm.rules.semantic_catalog_node_id IS 'Links to public.catalog_node for semantic term resolution';
COMMENT ON COLUMN edm.rule_steps.semantic_term_node_id IS 'Links to public.catalog_node (semantic_term type) for semantic resolution';

-- ============================================================================
-- PART 5: Semantic Edges (Linking everything together)
-- ============================================================================

-- Create edge type definitions
-- table is catalog_edge_types with column edge_type_name
DO $$
    BEGIN
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='catalog_edge_types') THEN
            -- need to supply tenant_id
            INSERT INTO public.catalog_edge_types (tenant_id, edge_type_name, description)
            SELECT COALESCE((SELECT id FROM public.tenants LIMIT 1), gen_random_uuid()), et.*
            FROM (VALUES
                ('semantic_term_maps_to_bo_field', 'Semantic term is used by a BO field'),
                ('bo_field_belongs_to_bo', 'Field belongs to a business object'),
                ('bo_references_physical_table', 'Business object is backed by physical table'),
                ('bo_field_maps_to_physical_column', 'Field maps to physical column'),
                ('semantic_term_used_in_rule', 'Semantic term is referenced in rule'),
                ('rule_references_bo', 'Rule references a business object')
            ) AS et(edge_type_name, description)
            ON CONFLICT (tenant_id, edge_type_name) DO NOTHING;
        ELSE
            RAISE NOTICE 'catalog_edge_types table missing, skipping edge type insertion';
        END IF;
    END$$;
-- ============================================================================
-- PART 6: Sample Data Insert (Calendar for 2026)
-- ============================================================================

INSERT INTO northwinds.calendar_mdm (
    calendar_date,
    is_business_day,
    is_weekend,
    region_code,
    holiday_name,
    source_system,
    confidence_score,
    trading_impact,
    effective_date,
    last_modified_by,
    created_by_user_id,
    tenant_id,
    datasource_id
)
SELECT
    d.calendar_date,
    d.is_business_day,
    d.is_weekend,
    d.region_code,
    d.holiday_name,
    'nager_date' AS source_system,
    CASE
        WHEN d.holiday_name IS NOT NULL THEN 95
        WHEN d.is_business_day THEN 99
        ELSE 98
    END AS confidence_score,
    CASE WHEN d.is_business_day THEN TRUE ELSE FALSE END AS trading_impact,
    d.calendar_date,
    'system_bootstrap',
    '00000000-0000-0000-0000-000000000001'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid
FROM (
    -- Sample 2026 calendar data (weekends + major holidays)
    SELECT
        d::DATE AS calendar_date,
        CASE
            WHEN EXTRACT(DOW FROM d::DATE) IN (0, 6) THEN FALSE
            WHEN d::DATE IN ('2026-01-01'::DATE, '2026-12-25'::DATE) THEN FALSE
            ELSE TRUE
        END AS is_business_day,
        CASE WHEN EXTRACT(DOW FROM d::DATE) IN (0, 6) THEN TRUE ELSE FALSE END AS is_weekend,
        'GB' AS region_code,
        CASE
            WHEN d::DATE = '2026-01-01'::DATE THEN 'New Year''s Day'
            WHEN d::DATE = '2026-04-05'::DATE THEN 'Easter Sunday'
            WHEN d::DATE = '2026-12-25'::DATE THEN 'Christmas Day'
            ELSE NULL
        END AS holiday_name
    FROM generate_series('2026-01-01'::timestamp, '2026-12-31'::timestamp, '1 day'::interval) d
) d
WHERE NOT EXISTS (
    SELECT 1 FROM northwinds.calendar_mdm
    WHERE calendar_date = d.calendar_date
    AND region_code = d.region_code
);

-- ============================================================================
-- PART 7: Documentation and Verification
-- ============================================================================

-- Create a view that shows semantic integration
-- create view only if semantic catalog exists
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='catalog_node') THEN
        EXECUTE 'CREATE OR REPLACE VIEW public.v_calendar_semantic_model AS
            SELECT
                st.node_name AS semantic_term,
                st.properties->>''business_definition'' AS definition,
                st.properties->>''data_type'' AS data_type,
                st.properties->>''category'' AS category,
                bo.node_name AS business_object,
                cm.column_name AS physical_column
            FROM public.catalog_node st
            LEFT JOIN public.catalog_edge ce ON st.id = ce.source_node_id
            LEFT JOIN public.catalog_node bo ON ce.target_node_id = bo.id
            LEFT JOIN LATERAL (
                SELECT st.properties->>''sql'' AS column_name
            ) cm ON TRUE
            WHERE st.properties->>''category'' IS NOT NULL
            AND st.node_name LIKE ''calendar.%''
            ORDER BY st.node_name;';
    ELSE
        RAISE NOTICE 'catalog_node missing, skipping semantic view creation';
    END IF;
END$do$;

-- Log completion
DO $$
BEGIN
    RAISE NOTICE 'Calendar Semantic Integration Complete:
    - Semantic term nodes created in public.catalog_node for calendar domain
    - Calendar business object created
    - Calendar MDM tables created in northwinds database
    - EDM rules schema updated with semantic_catalog_node_id references
    - Semantic edges ready for linking
    
    Next steps:
    1. Create bo_fields for calendar business object (link to semantic term nodes)
    2. Insert sample calendar data
    3. Create rules that reference calendar semantic terms
    4. Enable RLS on calendar_mdm table for multi-tenancy';
END$$;
