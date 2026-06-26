-- Rollback tenant support from abbreviations table

-- Drop tenant-specific indexes
DROP INDEX IF EXISTS sml.idx_abbreviations_tenant_abbr_lookup;
DROP INDEX IF EXISTS sml.idx_abbreviations_tenant;
DROP INDEX IF EXISTS sml.idx_abbreviations_tenant_abbr;

-- Recreate original unique constraint on abbreviation and remove tenant column if table present
DO $do$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class WHERE relkind='r' AND relname='abbreviation_lookup' AND pg_table_is_visible(oid)) THEN
    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_abbreviations_abbr') THEN
      EXECUTE 'CREATE UNIQUE INDEX idx_abbreviations_abbr ON sml.abbreviation_lookup(UPPER(abbreviation))';
    END IF;

    EXECUTE 'ALTER TABLE sml.abbreviation_lookup DROP COLUMN IF EXISTS tenant_id';
  END IF;
END
$do$;
