-- 20260825_okf_attested_knowledge_engine.up.sql
-- Core vs Custom OKF Attested Knowledge Engine Schema

CREATE TABLE IF NOT EXISTS public.okf_concept_manifest (
    concept_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL, -- 00000000-0000-0000-0000-000000000000 for CORE (gold_copy), specific UUID for CUSTOM
    concept_key VARCHAR(150) NOT NULL,
    concept_type VARCHAR(50) NOT NULL, -- concept/business-object, concept/semantic-term, concept/attested-calculation, concept/regulatory-rule
    version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    tenant_scope VARCHAR(20) NOT NULL DEFAULT 'custom', -- 'core' or 'custom'
    frontmatter_payload JSONB NOT NULL,
    content_markdown TEXT NOT NULL,
    merkle_seal VARCHAR(64) NOT NULL,
    verified_by VARCHAR(100),
    verification_timestamp TIMESTAMPTZ,
    catalog_node_id UUID REFERENCES public.catalog_node(id) ON DELETE SET NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_tenant_concept UNIQUE(tenant_id, concept_key, version)
);

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
CREATE INDEX IF NOT EXISTS idx_okf_scope ON public.okf_concept_manifest(tenant_scope, concept_key);
