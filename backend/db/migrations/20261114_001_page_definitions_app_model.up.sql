-- Page application model (variables, governed queries, tab state, chrome) -
-- CorePageDefinition.app in frontend/src/types/pageStudio.ts. One jsonb
-- column for the whole model so it grows without a migration per field.
-- NULL = a plain Business Object page, which behaves exactly as before.
ALTER TABLE public.page_definitions
    ADD COLUMN IF NOT EXISTS app_model JSONB;
