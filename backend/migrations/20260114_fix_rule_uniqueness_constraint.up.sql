-- Fix unique constraint to include datasource_id
-- This allows the same rule name to be used in different datasources within the same tenant

ALTER TABLE IF EXISTS catalog_validation_rules 
DROP CONSTRAINT IF EXISTS unique_rule_per_tenant;

-- Re-add constraint with datasource_id scope
-- Note: we use coalesce for datasource_id to handle potential nulls if that's allowed (though schema says NOT NULL)
DO $do$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.table_constraints 
    WHERE constraint_name = 'unique_rule_per_tenant' AND table_name = 'catalog_validation_rules'
  ) THEN
    ALTER TABLE IF EXISTS catalog_validation_rules
      ADD CONSTRAINT unique_rule_per_tenant UNIQUE (tenant_id, datasource_id, rule_name);
  END IF;
END
$do$;
