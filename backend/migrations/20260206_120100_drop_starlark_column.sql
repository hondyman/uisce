-- Drop the Starlark script_content column as ASL (condition_json) is the replacement
ALTER TABLE catalog_validation_rules
DROP COLUMN IF EXISTS script_content;
