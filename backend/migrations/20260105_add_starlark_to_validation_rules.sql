-- Add script_content column to catalog_validation_rules
ALTER TABLE IF EXISTS public.catalog_validation_rules
ADD COLUMN IF NOT EXISTS script_content TEXT;

-- Drop existing check constraint on rule_type
ALTER TABLE IF EXISTS public.catalog_validation_rules
DROP CONSTRAINT IF EXISTS catalog_validation_rules_rule_type_check;

-- Add new check constraint including 'starlark'
ALTER TABLE IF EXISTS public.catalog_validation_rules
ADD CONSTRAINT catalog_validation_rules_rule_type_check
CHECK (rule_type = ANY (ARRAY['business_logic'::text, 'field_format'::text, 'cardinality'::text, 'uniqueness'::text, 'referential_integrity'::text, 'starlark'::text, 'required_field'::text]));
