-- Add tenant support to abbreviations table
-- Core abbreviations belong to 'uisce' tenant and are available to all tenants
-- Custom abbreviations belong to specific tenants

-- Add tenant_id column with default 'uisce' for existing data
ALTER TABLE IF EXISTS sml.abbreviation_lookup 
ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(255) NOT NULL DEFAULT 'uisce';

-- Drop the old unique constraint on abbreviation
DROP INDEX IF EXISTS sml.idx_abbreviations_abbr;

-- Create new unique constraint on (tenant_id, abbreviation)
-- This allows different tenants to have the same abbreviation with different meanings
DO $do$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class WHERE relkind='r' AND relname='abbreviation_lookup') THEN
    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_abbreviations_tenant_abbr') THEN
      EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS idx_abbreviations_tenant_abbr ON sml.abbreviation_lookup(tenant_id, UPPER(abbreviation))';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_abbreviations_tenant') THEN
      EXECUTE 'CREATE INDEX IF NOT EXISTS idx_abbreviations_tenant ON sml.abbreviation_lookup(tenant_id)';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_abbreviations_tenant_abbr_lookup') THEN
      EXECUTE 'CREATE INDEX IF NOT EXISTS idx_abbreviations_tenant_abbr_lookup ON sml.abbreviation_lookup(tenant_id, abbreviation)';
    END IF;
  END IF;
END
$do$;
