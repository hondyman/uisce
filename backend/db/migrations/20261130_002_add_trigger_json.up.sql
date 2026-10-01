SET search_path = public;

ALTER TABLE public.bp_process_definition 
ADD COLUMN IF NOT EXISTS trigger_json JSONB;

CREATE INDEX IF NOT EXISTS idx_bp_process_definition_trigger 
ON public.bp_process_definition USING GIN (trigger_json);
