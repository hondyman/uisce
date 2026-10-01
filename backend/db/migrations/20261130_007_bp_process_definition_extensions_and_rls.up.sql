-- 20261130_007_bp_process_definition_extensions_and_rls.up.sql
-- 1. Add extension columns to public.bp_process_definition (the table Flow Builder and runtime workflows use)
-- 2. Enable and configure RLS on public.bp_process_definition
-- 3. Grant permissions to app_user
SET search_path = public;

-- 1. Add extension columns to bp_process_definition
ALTER TABLE public.bp_process_definition 
    ADD COLUMN IF NOT EXISTS source_type VARCHAR(20) DEFAULT 'CORE',
    ADD COLUMN IF NOT EXISTS base_definition_id UUID REFERENCES public.bp_process_definition(id),
    ADD COLUMN IF NOT EXISTS base_version INT,
    ADD COLUMN IF NOT EXISTS extensions_json JSONB;

-- 2. Enable Row Level Security on bp_process_definition
ALTER TABLE public.bp_process_definition ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS bp_process_def_read ON public.bp_process_definition;
CREATE POLICY bp_process_def_read ON public.bp_process_definition
    FOR SELECT
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')
        OR tenant_id = public.get_core_tenant_id()::text
        OR tenant_id = 'core'
        OR tenant_id = 'SYSTEM'
    );

DROP POLICY IF EXISTS bp_process_def_write ON public.bp_process_definition;
CREATE POLICY bp_process_def_write ON public.bp_process_definition
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')
    );

-- 3. Grant permissions to app_user
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON public.bp_process_definition TO app_user;
    END IF;
END $$;
