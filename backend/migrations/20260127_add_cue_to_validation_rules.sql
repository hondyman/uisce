-- Add 'cue' to the rule_type check constraint
DO $do$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'catalog_validation_rules') THEN
    ALTER TABLE catalog_validation_rules DROP CONSTRAINT IF EXISTS catalog_validation_rules_rule_type_check;

    ALTER TABLE catalog_validation_rules ADD CONSTRAINT catalog_validation_rules_rule_type_check 
    CHECK (rule_type IN ('field_format', 'cardinality', 'uniqueness', 'referential_integrity', 'business_logic', 'starlark', 'cue'));
  ELSE
    RAISE NOTICE 'Skipping add_cue_to_validation_rules: table catalog_validation_rules not found.';
  END IF;
END
$do$;
