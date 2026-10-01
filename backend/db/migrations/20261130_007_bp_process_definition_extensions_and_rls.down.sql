-- 20261130_007_bp_process_definition_extensions_and_rls.down.sql
SET search_path = public;

DROP POLICY IF EXISTS bp_process_def_read ON public.bp_process_definition;
DROP POLICY IF EXISTS bp_process_def_write ON public.bp_process_definition;
ALTER TABLE public.bp_process_definition DISABLE ROW LEVEL SECURITY;

ALTER TABLE public.bp_process_definition 
    DROP COLUMN IF EXISTS source_type,
    DROP COLUMN IF EXISTS base_definition_id,
    DROP COLUMN IF EXISTS base_version,
    DROP COLUMN IF EXISTS extensions_json;
