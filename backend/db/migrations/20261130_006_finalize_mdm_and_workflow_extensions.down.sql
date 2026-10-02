-- 20261130_006_finalize_mdm_and_workflow_extensions.down.sql
SET search_path = public;

-- Revert policy
DROP POLICY IF EXISTS semantic_surv_read ON public.semantic_survivorship_rules;
CREATE POLICY semantic_surv_read ON public.semantic_survivorship_rules
    FOR SELECT
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR tenant_id = public.get_core_tenant_id()
        OR tenant_id = '00000000-0000-0000-0000-000000000000'::uuid
    );

-- Drop columns if exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'business_processes') THEN
        ALTER TABLE public.business_processes 
            DROP COLUMN IF EXISTS source_type,
            DROP COLUMN IF EXISTS base_definition_id,
            DROP COLUMN IF EXISTS base_version,
            DROP COLUMN IF EXISTS extensions_json;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'business_process_definition') THEN
        ALTER TABLE public.business_process_definition 
            DROP COLUMN IF EXISTS source_type,
            DROP COLUMN IF EXISTS base_definition_id,
            DROP COLUMN IF EXISTS base_version,
            DROP COLUMN IF EXISTS extensions_json;
    END IF;
END $$;
