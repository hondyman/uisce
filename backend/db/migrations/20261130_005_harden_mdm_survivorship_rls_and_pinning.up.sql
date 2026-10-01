-- 20261130_005_harden_mdm_survivorship_rls_and_pinning.up.sql
-- Harden MDM survivorship RLS with SECURITY DEFINER core tenant function,
-- partial unique active index, and mdm_batch_rule_snapshot table.
SET search_path = public;

-- 1. Security definer helper to safely retrieve core tenant ID bypassing tenants table RLS
CREATE OR REPLACE FUNCTION public.get_core_tenant_id()
RETURNS uuid
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1;
$$;

-- 2. Update semantic_survivorship_rules read policy to use get_core_tenant_id()
DROP POLICY IF EXISTS semantic_surv_read ON public.semantic_survivorship_rules;
CREATE POLICY semantic_surv_read ON public.semantic_survivorship_rules
    FOR SELECT
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR tenant_id = public.get_core_tenant_id()
        OR tenant_id = '00000000-0000-0000-0000-000000000000'::uuid
    );

-- 3. Partial unique index to enforce single active rule per (tenant_id, entity_type, semantic_term_id)
CREATE UNIQUE INDEX IF NOT EXISTS uq_semantic_survivorship_active
    ON public.semantic_survivorship_rules (tenant_id, entity_type, semantic_term_id)
    WHERE is_active = true;

-- 4. Batch rule snapshot table for immutable, reproducible MDM survivorship execution
CREATE TABLE IF NOT EXISTS public.mdm_batch_rule_snapshot (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id VARCHAR(255) NOT NULL UNIQUE,
    tenant_id UUID NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    as_of TIMESTAMPTZ NOT NULL,
    pinned_rules_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mdm_batch_snapshot_tenant 
    ON public.mdm_batch_rule_snapshot (tenant_id, entity_type, as_of DESC);
