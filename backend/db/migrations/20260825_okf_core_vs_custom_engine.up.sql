-- 20260825_okf_core_vs_custom_engine.up.sql
-- Unified Core vs Custom OKF Knowledge Manifest with Precedence Scoring and RLS

CREATE TABLE IF NOT EXISTS public.okf_concept_manifest (
    concept_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL, -- Core = '00000000-0000-0000-0000-000000000000', Custom = Tenant UUID
    concept_key VARCHAR(150) NOT NULL,
    concept_type VARCHAR(50) NOT NULL, -- concept/business-object, concept/semantic-term, concept/attested-calculation, concept/regulatory-rule
    version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    tenant_scope VARCHAR(20) NOT NULL DEFAULT 'custom',
    is_core BOOLEAN GENERATED ALWAYS AS (tenant_id = '00000000-0000-0000-0000-000000000000') STORED,
    precedence_score INT NOT NULL DEFAULT 100, -- Custom overrides = 10, Core defaults = 100
    frontmatter_payload JSONB NOT NULL,
    content_markdown TEXT NOT NULL,
    merkle_seal VARCHAR(64) NOT NULL,
    verified_by VARCHAR(100),
    verification_timestamp TIMESTAMPTZ,
    catalog_node_id UUID REFERENCES public.catalog_node(id) ON DELETE SET NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_concept_version UNIQUE(tenant_id, concept_key, version)
);

-- Ensure columns exist if table was previously created by an earlier migration
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'public' AND table_name = 'okf_concept_manifest' AND column_name = 'precedence_score'
    ) THEN
        ALTER TABLE public.okf_concept_manifest ADD COLUMN precedence_score INT NOT NULL DEFAULT 100;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'public' AND table_name = 'okf_concept_manifest' AND column_name = 'is_core'
    ) THEN
        ALTER TABLE public.okf_concept_manifest ADD COLUMN is_core BOOLEAN GENERATED ALWAYS AS (tenant_id = '00000000-0000-0000-0000-000000000000') STORED;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'public' AND table_name = 'okf_concept_manifest' AND column_name = 'tenant_scope'
    ) THEN
        ALTER TABLE public.okf_concept_manifest ADD COLUMN tenant_scope VARCHAR(20) NOT NULL DEFAULT 'custom';
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS public.okf_attested_calculations (
    calculation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    concept_id UUID NOT NULL REFERENCES public.okf_concept_manifest(concept_id) ON DELETE CASCADE,
    metric_key VARCHAR(100) NOT NULL,
    runtime_engine VARCHAR(30) NOT NULL, -- WASM, VECTOR_SIMD, CEL, SQL_PUSHDOWN
    ast_definition JSONB NOT NULL,
    input_terms JSONB NOT NULL,
    test_assertions JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_okf_lookup ON public.okf_concept_manifest(tenant_id, concept_type, is_active);
CREATE INDEX IF NOT EXISTS idx_okf_key_precedence ON public.okf_concept_manifest(concept_key, precedence_score, is_active);

-- Enable Row-Level Security
ALTER TABLE public.okf_concept_manifest ENABLE ROW LEVEL SECURITY;

DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE policyname = 'okf_tenant_isolation_policy' AND tablename = 'okf_concept_manifest'
    ) THEN
        CREATE POLICY okf_tenant_isolation_policy ON public.okf_concept_manifest
            FOR ALL
            USING (
                tenant_id = '00000000-0000-0000-0000-000000000000' -- Universal Core
                OR tenant_id = NULLIF(current_setting('uisce.current_tenant', true), '')::uuid -- Active Custom
            );
    END IF;
END $$;
