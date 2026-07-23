-- ============================================================================
-- Cube Model Catalog Integration Schema
-- ============================================================================
-- Purpose: Define core/custom model layering with catalog integration
-- and efficient RBAC/ABAC security policy storage for Cube.js
--
-- Key Features:
-- 1. Core models: Auto-generated from catalog metadata, read-only base
-- 2. Custom models: Extend/override core with tenant-specific customizations
-- 3. Security policies: Pre-computed RBAC/ABAC rules cached for performance
-- 4. Wizard state: Track model building wizard progress
-- ============================================================================

BEGIN;

-- ============================================================================
-- 1. Core Model Definitions (Generated from Catalog)
-- ============================================================================
-- These are auto-generated from catalog_node and should not be manually edited

CREATE TABLE IF NOT EXISTS public.cube_core_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    datasource_id UUID NOT NULL REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
    
    -- Source metadata linkage
    catalog_node_id UUID REFERENCES public.catalog_node(id) ON DELETE SET NULL,
    source_table_id UUID REFERENCES public.metadata_tables(id) ON DELETE SET NULL,
    
    -- Model identity
    model_name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    
    -- SQL configuration
    sql_table VARCHAR(500), -- e.g., "schema.table_name"
    sql_expression TEXT, -- Custom SQL if not table-based
    data_source VARCHAR(100) DEFAULT 'default', -- starrocks, trino, etc.
    
    -- Generated YAML (cached for performance)
    generated_yaml TEXT NOT NULL,
    yaml_hash VARCHAR(64) NOT NULL, -- SHA256 for change detection
    
    -- Metadata
    refresh_key_sql TEXT,
    primary_key_columns JSONB DEFAULT '[]',
    
    -- Lifecycle
    is_active BOOLEAN DEFAULT true,
    is_published BOOLEAN DEFAULT false,
    version INT DEFAULT 1,
    generation_source VARCHAR(50) DEFAULT 'catalog', -- catalog, import, manual
    
    -- Timestamps
    last_synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_core_model_per_datasource UNIQUE (tenant_id, datasource_id, model_name)
);

CREATE INDEX IF NOT EXISTS idx_cube_core_models_tenant ON public.cube_core_models(tenant_id, datasource_id);
CREATE INDEX IF NOT EXISTS idx_cube_core_models_catalog ON public.cube_core_models(catalog_node_id);
CREATE INDEX IF NOT EXISTS idx_cube_core_models_active ON public.cube_core_models(tenant_id, datasource_id, is_active) WHERE is_active = true;

COMMENT ON TABLE public.cube_core_models IS 'Auto-generated Cube models from catalog metadata - base layer';
COMMENT ON COLUMN public.cube_core_models.yaml_hash IS 'SHA256 hash for detecting changes in regeneration';

-- ============================================================================
-- 2. Core Measures (Generated from Catalog Columns)
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.cube_core_measures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    core_model_id UUID NOT NULL REFERENCES public.cube_core_models(id) ON DELETE CASCADE,
    
    -- Source linkage
    catalog_column_id UUID REFERENCES public.catalog_node(id) ON DELETE SET NULL,
    metadata_column_id UUID REFERENCES public.metadata_columns(id) ON DELETE SET NULL,
    
    -- Measure definition
    measure_name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    
    -- SQL and type
    measure_type VARCHAR(50) NOT NULL, -- count, sum, avg, min, max, countDistinct, countDistinctApprox, runningTotal
    sql_expression TEXT NOT NULL,
    data_type VARCHAR(50) DEFAULT 'number', -- number, string, boolean, time
    
    -- Formatting
    format_type VARCHAR(50), -- currency, percent, number
    format_meta JSONB DEFAULT '{}', -- precision, prefix, suffix, etc.
    
    -- Rollup compatibility
    rolling_window JSONB, -- { trailing: '30 days', offset: 'start' }
    drill_members JSONB DEFAULT '[]',
    
    -- Flags
    is_visible BOOLEAN DEFAULT true,
    is_cumulative BOOLEAN DEFAULT false,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_core_measure UNIQUE (core_model_id, measure_name)
);

CREATE INDEX IF NOT EXISTS idx_cube_core_measures_model ON public.cube_core_measures(core_model_id);

-- ============================================================================
-- 3. Core Dimensions (Generated from Catalog Columns)
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.cube_core_dimensions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    core_model_id UUID NOT NULL REFERENCES public.cube_core_models(id) ON DELETE CASCADE,
    
    -- Source linkage
    catalog_column_id UUID REFERENCES public.catalog_node(id) ON DELETE SET NULL,
    metadata_column_id UUID REFERENCES public.metadata_columns(id) ON DELETE SET NULL,
    
    -- Dimension definition
    dimension_name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    
    -- SQL and type
    dimension_type VARCHAR(50) NOT NULL, -- string, number, boolean, time, geo
    sql_expression TEXT NOT NULL,
    
    -- Time dimension specific
    is_time_dimension BOOLEAN DEFAULT false,
    granularities JSONB DEFAULT '[]', -- ['day', 'week', 'month', 'quarter', 'year']
    
    -- Relationships
    primary_key BOOLEAN DEFAULT false,
    foreign_key_to UUID REFERENCES public.cube_core_models(id) ON DELETE SET NULL,
    
    -- UI hints
    case_sensitive BOOLEAN DEFAULT true,
    is_visible BOOLEAN DEFAULT true,
    suggestFilterValues BOOLEAN DEFAULT true,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_core_dimension UNIQUE (core_model_id, dimension_name)
);

CREATE INDEX IF NOT EXISTS idx_cube_core_dimensions_model ON public.cube_core_dimensions(core_model_id);
CREATE INDEX IF NOT EXISTS idx_cube_core_dimensions_time ON public.cube_core_dimensions(core_model_id, is_time_dimension) WHERE is_time_dimension = true;

-- ============================================================================
-- 4. Custom Model Extensions (Tenant-specific overrides)
-- ============================================================================
-- Custom models can extend core models or define entirely new models

CREATE TABLE IF NOT EXISTS public.cube_custom_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    datasource_id UUID NOT NULL REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
    
    -- Extension relationship
    extends_core_model_id UUID REFERENCES public.cube_core_models(id) ON DELETE SET NULL,
    
    -- Custom model identity (if not extending)
    model_name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    
    -- Override/extend mode
    extension_mode VARCHAR(50) DEFAULT 'extend', -- extend, override, standalone
    
    -- Custom SQL (overrides core if present)
    sql_table VARCHAR(500),
    sql_expression TEXT,
    data_source VARCHAR(100),
    
    -- Custom YAML (merged with core)
    custom_yaml TEXT,
    
    -- Full merged YAML (cached)
    merged_yaml TEXT,
    merged_yaml_hash VARCHAR(64),
    
    -- Metadata overrides
    refresh_key_sql TEXT,
    custom_joins JSONB DEFAULT '[]', -- Custom join definitions
    
    -- Lifecycle
    is_active BOOLEAN DEFAULT true,
    is_published BOOLEAN DEFAULT false,
    version INT DEFAULT 1,
    
    -- Ownership
    created_by UUID REFERENCES public.users(id),
    updated_by UUID REFERENCES public.users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_custom_model_per_datasource UNIQUE (tenant_id, datasource_id, model_name)
);

CREATE INDEX IF NOT EXISTS idx_cube_custom_models_tenant ON public.cube_custom_models(tenant_id, datasource_id);
CREATE INDEX IF NOT EXISTS idx_cube_custom_models_extends ON public.cube_custom_models(extends_core_model_id);

COMMENT ON TABLE public.cube_custom_models IS 'Tenant-specific Cube model customizations - extends or overrides core';
COMMENT ON COLUMN public.cube_custom_models.extension_mode IS 'extend: add to core, override: replace core, standalone: no core dependency';

-- ============================================================================
-- 5. Custom Measures (Extend/Override Core)
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.cube_custom_measures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    custom_model_id UUID NOT NULL REFERENCES public.cube_custom_models(id) ON DELETE CASCADE,
    
    -- Override linkage
    overrides_core_measure_id UUID REFERENCES public.cube_core_measures(id) ON DELETE SET NULL,
    
    -- Measure definition
    measure_name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    
    -- SQL and type
    measure_type VARCHAR(50) NOT NULL,
    sql_expression TEXT NOT NULL,
    data_type VARCHAR(50) DEFAULT 'number',
    
    -- Formatting
    format_type VARCHAR(50),
    format_meta JSONB DEFAULT '{}',
    
    -- Advanced
    rolling_window JSONB,
    drill_members JSONB DEFAULT '[]',
    filters JSONB DEFAULT '[]', -- Pre-filter conditions
    
    -- Flags
    is_visible BOOLEAN DEFAULT true,
    is_calculated BOOLEAN DEFAULT false, -- Derived from other measures
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_custom_measure UNIQUE (custom_model_id, measure_name)
);

CREATE INDEX IF NOT EXISTS idx_cube_custom_measures_model ON public.cube_custom_measures(custom_model_id);

-- ============================================================================
-- 6. Custom Dimensions (Extend/Override Core)
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.cube_custom_dimensions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    custom_model_id UUID NOT NULL REFERENCES public.cube_custom_models(id) ON DELETE CASCADE,
    
    -- Override linkage
    overrides_core_dimension_id UUID REFERENCES public.cube_core_dimensions(id) ON DELETE SET NULL,
    
    -- Dimension definition
    dimension_name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    
    -- SQL and type
    dimension_type VARCHAR(50) NOT NULL,
    sql_expression TEXT NOT NULL,
    
    -- Time dimension specific
    is_time_dimension BOOLEAN DEFAULT false,
    granularities JSONB DEFAULT '[]',
    
    -- UI hints
    case_sensitive BOOLEAN DEFAULT true,
    is_visible BOOLEAN DEFAULT true,
    suggestFilterValues BOOLEAN DEFAULT true,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_custom_dimension UNIQUE (custom_model_id, dimension_name)
);

CREATE INDEX IF NOT EXISTS idx_cube_custom_dimensions_model ON public.cube_custom_dimensions(custom_model_id);

-- ============================================================================
-- 7. RBAC/ABAC Security Policies (Pre-computed for Performance)
-- ============================================================================
-- Cached security policies to avoid runtime computation overhead

CREATE TABLE IF NOT EXISTS public.cube_security_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    datasource_id UUID NOT NULL REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
    
    -- Policy identity
    policy_name VARCHAR(255) NOT NULL,
    policy_type VARCHAR(50) NOT NULL, -- rbac, abac, row_level, column_level, data_masking
    description TEXT,
    
    -- Scope - which models/cubes this applies to
    applies_to_models JSONB DEFAULT '["*"]', -- Model names or ["*"] for all
    applies_to_measures JSONB DEFAULT '["*"]',
    applies_to_dimensions JSONB DEFAULT '["*"]',
    
    -- RBAC configuration
    required_roles JSONB DEFAULT '[]', -- Roles that grant access
    denied_roles JSONB DEFAULT '[]', -- Roles that deny access
    
    -- ABAC configuration
    attribute_conditions JSONB DEFAULT '{}', -- CEL or JSON-path conditions
    -- Example: { "user.department": { "in": ["finance", "analytics"] }, "user.level": { "gte": 3 } }
    
    -- Row-level security (for queryRewrite)
    row_filter_sql TEXT, -- WHERE clause template
    row_filter_params JSONB DEFAULT '{}', -- Parameter mappings from securityContext
    -- Example: row_filter_sql = "region IN (SELECT region FROM user_regions WHERE user_id = ${user_id})"
    
    -- Column-level security
    column_visibility JSONB DEFAULT '{}', -- { "column_name": "hidden" | "masked" | "visible" }
    
    -- Data masking rules
    masking_rules JSONB DEFAULT '{}', -- { "column_name": { "type": "partial", "show_last": 4 } }
    
    -- Priority and combination
    priority INT DEFAULT 100, -- Higher = evaluated first
    combine_mode VARCHAR(50) DEFAULT 'most_restrictive', -- most_restrictive, least_restrictive, all_must_pass
    
    -- Pre-computed cache
    compiled_policy JSONB, -- Pre-evaluated policy for fast lookup
    policy_hash VARCHAR(64), -- For cache invalidation
    
    -- Lifecycle
    is_active BOOLEAN DEFAULT true,
    valid_from TIMESTAMPTZ DEFAULT NOW(),
    valid_until TIMESTAMPTZ,
    
    created_by UUID REFERENCES public.users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_security_policy UNIQUE (tenant_id, datasource_id, policy_name)
);

CREATE INDEX IF NOT EXISTS idx_cube_security_policies_tenant ON public.cube_security_policies(tenant_id, datasource_id);
CREATE INDEX IF NOT EXISTS idx_cube_security_policies_active ON public.cube_security_policies(tenant_id, datasource_id, is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_cube_security_policies_type ON public.cube_security_policies(policy_type);

COMMENT ON TABLE public.cube_security_policies IS 'Pre-computed RBAC/ABAC security policies for efficient Cube query filtering';
COMMENT ON COLUMN public.cube_security_policies.compiled_policy IS 'Pre-evaluated policy JSON for O(1) lookup during query execution';

-- ============================================================================
-- 8. User Security Context Cache
-- ============================================================================
-- Cache computed security contexts per user session for performance

CREATE TABLE IF NOT EXISTS public.cube_security_context_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    datasource_id UUID NOT NULL REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
    
    -- Cached security context (passed to Cube.js)
    security_context JSONB NOT NULL,
    -- Example:
    -- {
    --   "tenant_id": "uuid",
    --   "user_id": "uuid",
    --   "roles": ["analyst", "finance_viewer"],
    --   "attributes": { "department": "finance", "region": "US-WEST" },
    --   "row_filters": { "Orders": "region = 'US-WEST'" },
    --   "visible_measures": { "Orders": ["count", "total_amount"] },
    --   "visible_dimensions": { "Orders": ["status", "category"] },
    --   "masked_columns": { "Customers": { "email": "partial" } }
    -- }
    
    -- Cache metadata
    computed_from_policies JSONB DEFAULT '[]', -- Policy IDs that contributed
    context_hash VARCHAR(64) NOT NULL, -- For cache invalidation
    
    -- TTL
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_user_context_cache UNIQUE (tenant_id, user_id, datasource_id)
);

CREATE INDEX IF NOT EXISTS idx_security_context_cache_lookup ON public.cube_security_context_cache(tenant_id, user_id, datasource_id);
CREATE INDEX IF NOT EXISTS idx_security_context_cache_expiry ON public.cube_security_context_cache(expires_at);

-- Cleanup job: DELETE FROM cube_security_context_cache WHERE expires_at < NOW();

-- ============================================================================
-- 9. Model Wizard State (Track wizard progress)
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.cube_model_wizard_state (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    datasource_id UUID NOT NULL REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    
    -- Wizard session
    session_id VARCHAR(100) NOT NULL,
    wizard_type VARCHAR(50) NOT NULL, -- new_model, extend_model, edit_model
    
    -- Current step
    current_step INT DEFAULT 1,
    total_steps INT DEFAULT 6,
    step_name VARCHAR(100),
    
    -- Accumulated state
    wizard_state JSONB NOT NULL DEFAULT '{}',
    -- Example:
    -- {
    --   "step1_source": { "source_type": "catalog", "catalog_node_id": "uuid", "table_name": "orders" },
    --   "step2_identity": { "model_name": "Orders", "display_name": "Customer Orders", "description": "..." },
    --   "step3_measures": [{ "name": "count", "type": "count", "sql": "*" }],
    --   "step4_dimensions": [{ "name": "status", "type": "string", "sql": "${TABLE}.status" }],
    --   "step5_relationships": [{ "join_to": "Customers", "sql": "${TABLE}.customer_id = ${Customers}.id" }],
    --   "step6_security": { "row_filter": "tenant_id = '${SECURITY_CONTEXT.tenant_id}'" }
    -- }
    
    -- Validation state
    validation_errors JSONB DEFAULT '[]',
    is_valid BOOLEAN DEFAULT false,
    
    -- Preview
    preview_yaml TEXT,
    
    -- Timestamps
    last_step_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ DEFAULT NOW() + INTERVAL '24 hours',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_wizard_session UNIQUE (session_id)
);

CREATE INDEX IF NOT EXISTS idx_wizard_state_user ON public.cube_model_wizard_state(tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_wizard_state_expiry ON public.cube_model_wizard_state(expires_at);

-- ============================================================================
-- 10. Model Generation History (Audit trail)
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.cube_model_generation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    datasource_id UUID NOT NULL REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
    
    -- Target model
    core_model_id UUID REFERENCES public.cube_core_models(id) ON DELETE SET NULL,
    custom_model_id UUID REFERENCES public.cube_custom_models(id) ON DELETE SET NULL,
    model_name VARCHAR(255) NOT NULL,
    
    -- Generation details
    generation_type VARCHAR(50) NOT NULL, -- catalog_sync, wizard_create, wizard_edit, import, api
    source_description TEXT,
    
    -- Before/After
    previous_yaml TEXT,
    new_yaml TEXT,
    diff_summary JSONB, -- { added_measures: [], removed_measures: [], changed_dimensions: [] }
    
    -- Result
    success BOOLEAN DEFAULT true,
    error_message TEXT,
    
    -- Actor
    triggered_by UUID REFERENCES public.users(id),
    triggered_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_model_gen_history_tenant ON public.cube_model_generation_history(tenant_id, datasource_id);
CREATE INDEX IF NOT EXISTS idx_model_gen_history_time ON public.cube_model_generation_history(triggered_at DESC);

-- ============================================================================
-- 11. Pre-aggregation Suggestions (AI-assisted)
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.cube_preagg_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    datasource_id UUID NOT NULL REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
    
    -- Target model
    model_id UUID NOT NULL, -- Can be core or custom
    model_type VARCHAR(50) NOT NULL, -- core, custom
    model_name VARCHAR(255) NOT NULL,
    
    -- Suggestion
    suggestion_name VARCHAR(255) NOT NULL,
    suggested_measures JSONB DEFAULT '[]',
    suggested_dimensions JSONB DEFAULT '[]',
    suggested_time_dimension VARCHAR(255),
    suggested_granularity VARCHAR(50),
    suggested_partition_granularity VARCHAR(50),
    
    -- Reasoning
    suggestion_reason TEXT,
    expected_speedup_factor NUMERIC(5,2),
    estimated_storage_mb NUMERIC(10,2),
    query_patterns_matched JSONB DEFAULT '[]', -- Top queries this would optimize
    
    -- Generated YAML
    preagg_yaml TEXT NOT NULL,
    
    -- Status
    status VARCHAR(50) DEFAULT 'pending', -- pending, accepted, rejected, applied
    applied_at TIMESTAMPTZ,
    applied_by UUID REFERENCES public.users(id),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_preagg_suggestions_tenant ON public.cube_preagg_suggestions(tenant_id, datasource_id);
CREATE INDEX IF NOT EXISTS idx_preagg_suggestions_model ON public.cube_preagg_suggestions(model_id);
CREATE INDEX IF NOT EXISTS idx_preagg_suggestions_status ON public.cube_preagg_suggestions(status);

-- ============================================================================
-- Helper Functions
-- ============================================================================

-- Function to get effective security context for a user
CREATE OR REPLACE FUNCTION get_cube_security_context(
    p_tenant_id UUID,
    p_user_id UUID,
    p_datasource_id UUID
) RETURNS JSONB AS $$
DECLARE
    v_cached JSONB;
    v_context JSONB;
    v_policies RECORD;
    v_row_filters JSONB := '{}';
    v_visible_measures JSONB := '{}';
    v_visible_dimensions JSONB := '{}';
    v_roles JSONB;
    v_attributes JSONB;
BEGIN
    -- Check cache first
    SELECT security_context INTO v_cached
    FROM public.cube_security_context_cache
    WHERE tenant_id = p_tenant_id 
      AND user_id = p_user_id 
      AND datasource_id = p_datasource_id
      AND expires_at > NOW();
    
    IF v_cached IS NOT NULL THEN
        RETURN v_cached;
    END IF;
    
    -- Get user roles and attributes (assuming they exist in users table or related)
    SELECT COALESCE(u.roles, '[]'::jsonb), COALESCE(u.attributes, '{}'::jsonb)
    INTO v_roles, v_attributes
    FROM public.users u
    WHERE u.id = p_user_id;
    
    -- Aggregate policies
    FOR v_policies IN
        SELECT * FROM public.cube_security_policies
        WHERE tenant_id = p_tenant_id
          AND datasource_id = p_datasource_id
          AND is_active = true
          AND (valid_until IS NULL OR valid_until > NOW())
        ORDER BY priority DESC
    LOOP
        -- Process row filters
        IF v_policies.row_filter_sql IS NOT NULL THEN
            v_row_filters := v_row_filters || jsonb_build_object(
                COALESCE((v_policies.applies_to_models->>0), '*'),
                v_policies.row_filter_sql
            );
        END IF;
        
        -- Process column visibility
        IF v_policies.column_visibility IS NOT NULL AND v_policies.column_visibility != '{}'::jsonb THEN
            -- Merge visibility rules
            v_visible_measures := v_visible_measures || v_policies.column_visibility;
        END IF;
    END LOOP;
    
    -- Build context
    v_context := jsonb_build_object(
        'tenant_id', p_tenant_id,
        'datasource_id', p_datasource_id,
        'user_id', p_user_id,
        'roles', v_roles,
        'attributes', v_attributes,
        'row_filters', v_row_filters,
        'visible_measures', v_visible_measures,
        'visible_dimensions', v_visible_dimensions,
        'computed_at', NOW()
    );
    
    -- Cache it (1 hour TTL)
    INSERT INTO public.cube_security_context_cache 
        (tenant_id, user_id, datasource_id, security_context, context_hash, expires_at)
    VALUES 
        (p_tenant_id, p_user_id, p_datasource_id, v_context, md5(v_context::text), NOW() + INTERVAL '1 hour')
    ON CONFLICT (tenant_id, user_id, datasource_id) 
    DO UPDATE SET 
        security_context = EXCLUDED.security_context,
        context_hash = EXCLUDED.context_hash,
        expires_at = EXCLUDED.expires_at,
        created_at = NOW();
    
    RETURN v_context;
END;
$$ LANGUAGE plpgsql;

-- Function to invalidate security context cache for a tenant
CREATE OR REPLACE FUNCTION invalidate_security_context_cache(
    p_tenant_id UUID,
    p_datasource_id UUID DEFAULT NULL
) RETURNS INT AS $$
DECLARE
    v_count INT;
BEGIN
    IF p_datasource_id IS NULL THEN
        DELETE FROM public.cube_security_context_cache
        WHERE tenant_id = p_tenant_id;
    ELSE
        DELETE FROM public.cube_security_context_cache
        WHERE tenant_id = p_tenant_id AND datasource_id = p_datasource_id;
    END IF;
    
    GET DIAGNOSTICS v_count = ROW_COUNT;
    RETURN v_count;
END;
$$ LANGUAGE plpgsql;

-- Trigger to invalidate cache when policies change
CREATE OR REPLACE FUNCTION trigger_invalidate_security_cache() RETURNS TRIGGER AS $$
BEGIN
    PERFORM invalidate_security_context_cache(
        COALESCE(NEW.tenant_id, OLD.tenant_id),
        COALESCE(NEW.datasource_id, OLD.datasource_id)
    );
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_security_policy_cache_invalidate ON public.cube_security_policies;
CREATE TRIGGER trg_security_policy_cache_invalidate
    AFTER INSERT OR UPDATE OR DELETE ON public.cube_security_policies
    FOR EACH ROW EXECUTE FUNCTION trigger_invalidate_security_cache();

-- ============================================================================
-- Views for Easy Querying
-- ============================================================================

-- Unified view of all models (core + custom merged)
CREATE OR REPLACE VIEW public.cube_all_models AS
SELECT 
    'core' as model_layer,
    cm.id,
    cm.tenant_id,
    cm.datasource_id,
    cm.model_name,
    cm.display_name,
    cm.description,
    cm.sql_table,
    cm.data_source,
    cm.generated_yaml as yaml,
    cm.is_active,
    cm.is_published,
    cm.version,
    NULL::uuid as extends_core_model_id,
    cm.created_at,
    cm.updated_at
FROM public.cube_core_models cm

UNION ALL

SELECT 
    'custom' as model_layer,
    cust.id,
    cust.tenant_id,
    cust.datasource_id,
    cust.model_name,
    cust.display_name,
    cust.description,
    cust.sql_table,
    cust.data_source,
    COALESCE(cust.merged_yaml, cust.custom_yaml) as yaml,
    cust.is_active,
    cust.is_published,
    cust.version,
    cust.extends_core_model_id,
    cust.created_at,
    cust.updated_at
FROM public.cube_custom_models cust;

COMMENT ON VIEW public.cube_all_models IS 'Unified view of core and custom Cube models';

COMMIT;
