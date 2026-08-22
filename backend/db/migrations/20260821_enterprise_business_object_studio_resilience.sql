-- ============================================================================
-- Enterprise Business Object Studio Resilience Migration
-- 1. Fan-Out & Chasm Trap Cardinality Fields
-- 2. Base Filter Predicate & Tenant Invariants
-- 3. Schema Drift Invalidation Tracking
-- 4. Additivity Scope (Semi-Additive / Non-Additive Metric Locks)
-- 5. Tiered Storage Pair Bindings (Hot StarRocks / Cold Iceberg Seam)
-- 6. Maker-Checker State Machine & Versioning
-- ============================================================================

-- Enhance business_object_bindings
ALTER TABLE public.business_object_bindings
    ADD COLUMN IF NOT EXISTS base_filter_predicate JSONB DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS hot_driving_node_id UUID REFERENCES public.catalog_node(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS cold_driving_node_id UUID REFERENCES public.catalog_node(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS temporal_watermark_column TEXT,
    ADD COLUMN IF NOT EXISTS join_cardinality VARCHAR(20) DEFAULT '1:1', -- 1:1, N:1, 1:N, M:N
    ADD COLUMN IF NOT EXISTS two_stage_cte_enabled BOOLEAN DEFAULT true,
    ADD COLUMN IF NOT EXISTS version_id INT DEFAULT 1;

-- Also enhance legacy business_object_binding if it exists
DO $$ 
BEGIN 
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'business_object_binding') THEN
        ALTER TABLE public.business_object_binding
            ADD COLUMN IF NOT EXISTS base_filter_predicate JSONB DEFAULT '{}'::jsonb,
            ADD COLUMN IF NOT EXISTS hot_driving_node_id UUID REFERENCES public.catalog_node(id) ON DELETE SET NULL,
            ADD COLUMN IF NOT EXISTS cold_driving_node_id UUID REFERENCES public.catalog_node(id) ON DELETE SET NULL,
            ADD COLUMN IF NOT EXISTS temporal_watermark_column TEXT,
            ADD COLUMN IF NOT EXISTS join_cardinality VARCHAR(20) DEFAULT '1:1',
            ADD COLUMN IF NOT EXISTS two_stage_cte_enabled BOOLEAN DEFAULT true,
            ADD COLUMN IF NOT EXISTS version_id INT DEFAULT 1;
    END IF;
END $$;

-- Enhance bo_fields / business_object_fields
DO $$ 
BEGIN 
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'bo_fields') THEN
        ALTER TABLE public.bo_fields
            ADD COLUMN IF NOT EXISTS additivity_scope VARCHAR(50) DEFAULT 'FULLY_ADDITIVE', -- FULLY_ADDITIVE, SEMI_ADDITIVE_ACROSS_TIME, NON_ADDITIVE
            ADD COLUMN IF NOT EXISTS binding_status VARCHAR(50) DEFAULT 'RESOLVED', -- RESOLVED, DRIFT_DEGRADED, UNRESOLVED
            ADD COLUMN IF NOT EXISTS drift_detected_at TIMESTAMPTZ,
            ADD COLUMN IF NOT EXISTS drift_details JSONB DEFAULT '{}'::jsonb,
            ADD COLUMN IF NOT EXISTS temporal_aggregation_rule TEXT;
    END IF;

    IF EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'business_object_fields') THEN
        ALTER TABLE public.business_object_fields
            ADD COLUMN IF NOT EXISTS additivity_scope VARCHAR(50) DEFAULT 'FULLY_ADDITIVE',
            ADD COLUMN IF NOT EXISTS binding_status VARCHAR(50) DEFAULT 'RESOLVED',
            ADD COLUMN IF NOT EXISTS drift_detected_at TIMESTAMPTZ,
            ADD COLUMN IF NOT EXISTS drift_details JSONB DEFAULT '{}'::jsonb,
            ADD COLUMN IF NOT EXISTS temporal_aggregation_rule TEXT;
    END IF;
END $$;

-- Enhance business_objects table with Maker-Checker lifecycle
ALTER TABLE public.business_objects
    ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'PUBLISHED', -- DRAFT, PENDING_APPROVAL, PUBLISHED, DEPRECATED
    ADD COLUMN IF NOT EXISTS active_version_id INT DEFAULT 1,
    ADD COLUMN IF NOT EXISTS approved_by UUID,
    ADD COLUMN IF NOT EXISTS approved_at TIMESTAMPTZ;

-- Indices for drift detection & status queries
CREATE INDEX IF NOT EXISTS idx_bo_status ON public.business_objects (tenant_id, status);
