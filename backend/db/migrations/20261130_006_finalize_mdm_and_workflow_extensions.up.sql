-- 20261130_006_finalize_mdm_and_workflow_extensions.up.sql
-- 1. Remove zero-UUID from semantic_surv_read policy to enforce pure core + tenant RLS
-- 2. Formalize app_user grants for semantic_survivorship_rules and mdm_batch_rule_snapshot
-- 3. Add workflow extension columns (source_type, base_definition_id, base_version, extensions_json)
SET search_path = public;

-- 1. Update semantic_survivorship_rules read policy to remove zero-UUID
DROP POLICY IF EXISTS semantic_surv_read ON public.semantic_survivorship_rules;
CREATE POLICY semantic_surv_read ON public.semantic_survivorship_rules
    FOR SELECT
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR tenant_id = public.get_core_tenant_id()
    );

-- 2. Grant permissions to app_user role if it exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON public.semantic_survivorship_rules TO app_user;
        GRANT SELECT, INSERT, UPDATE, DELETE ON public.mdm_batch_rule_snapshot TO app_user;
        GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO app_user;
    END IF;
END $$;

-- 3. Add workflow extension columns to business process tables
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'business_processes') THEN
        ALTER TABLE public.business_processes 
            ADD COLUMN IF NOT EXISTS source_type VARCHAR(20) DEFAULT 'CORE',
            ADD COLUMN IF NOT EXISTS base_definition_id UUID REFERENCES public.business_processes(id),
            ADD COLUMN IF NOT EXISTS base_version INT,
            ADD COLUMN IF NOT EXISTS extensions_json JSONB;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'business_process_definition') THEN
        ALTER TABLE public.business_process_definition 
            ADD COLUMN IF NOT EXISTS source_type VARCHAR(20) DEFAULT 'CORE',
            ADD COLUMN IF NOT EXISTS base_definition_id UUID REFERENCES public.business_process_definition(id),
            ADD COLUMN IF NOT EXISTS base_version INT,
            ADD COLUMN IF NOT EXISTS extensions_json JSONB;
    END IF;
END $$;
