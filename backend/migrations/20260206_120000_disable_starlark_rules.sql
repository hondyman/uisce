-- Disable Starlark rules as the engine is removed
UPDATE catalog_validation_rules
SET is_active = false
WHERE rule_type = 'starlark';

-- Optional: You could delete them, but keeping them inactive is safer for now.
-- DELETE FROM catalog_validation_rules WHERE rule_type = 'starlark';
