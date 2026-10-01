SET search_path = public;

DROP INDEX IF EXISTS public.idx_bp_process_definition_trigger;

ALTER TABLE public.bp_process_definition 
DROP COLUMN IF EXISTS trigger_json;
