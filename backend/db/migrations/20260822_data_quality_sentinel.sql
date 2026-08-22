-- 20260822_data_quality_sentinel.sql
-- Integrates statistical profiling and quality gatekeeper state into Business Object fields

DO $$ 
BEGIN 
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'business_object_field') THEN
        ALTER TABLE public.business_object_field
            ADD COLUMN IF NOT EXISTS quality_status VARCHAR(50) DEFAULT 'UNPROFILED',
            ADD COLUMN IF NOT EXISTS quality_profile JSONB DEFAULT '{}'::jsonb,
            ADD COLUMN IF NOT EXISTS default_fallback_value TEXT,
            ADD COLUMN IF NOT EXISTS last_profiled_at TIMESTAMPTZ;
    END IF;

    IF EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'bo_fields') THEN
        ALTER TABLE public.bo_fields
            ADD COLUMN IF NOT EXISTS quality_status VARCHAR(50) DEFAULT 'UNPROFILED',
            ADD COLUMN IF NOT EXISTS quality_profile JSONB DEFAULT '{}'::jsonb,
            ADD COLUMN IF NOT EXISTS default_fallback_value TEXT,
            ADD COLUMN IF NOT EXISTS last_profiled_at TIMESTAMPTZ;
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS public.catalog_data_quality_audit (
    audit_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    bo_id UUID NOT NULL,
    field_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    sample_size INT NOT NULL,
    null_count INT NOT NULL,
    distinct_count INT NOT NULL,
    null_ratio NUMERIC(6, 4) NOT NULL,
    uniqueness_ratio NUMERIC(6, 4) NOT NULL,
    conformance_ratio NUMERIC(6, 4) NOT NULL,
    quality_gate_passed BOOLEAN NOT NULL,
    blocking_reasons JSONB DEFAULT '[]'::jsonb,
    profiled_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bo_field_quality 
ON public.business_object_field (bo_id, quality_status);

CREATE INDEX IF NOT EXISTS idx_dq_audit_lookup 
ON public.catalog_data_quality_audit (tenant_id, bo_id, profiled_at DESC);
