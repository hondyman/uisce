-- Migration: Add UUID-based entity linking to validation rules
-- This migration allows validation rules to reference entities by UUID instead of just name
-- Maintains backward compatibility with existing name-based rules

-- Add target_entity_id column to store entity UUID
ALTER TABLE IF EXISTS catalog_validation_rules
ADD COLUMN IF NOT EXISTS target_entity_id UUID;

-- Add target_entity_ids for multi-entity support (array of UUIDs)
ALTER TABLE IF EXISTS catalog_validation_rules
ADD COLUMN IF NOT EXISTS target_entity_ids UUID[] DEFAULT ARRAY[]::UUID[];

-- Add datasource_id for scoping to specific datasources
ALTER TABLE IF EXISTS catalog_validation_rules
ADD COLUMN IF NOT EXISTS datasource_id UUID;

-- Add a check constraint to ensure at least one entity reference exists
ALTER TABLE IF EXISTS catalog_validation_rules
DROP CONSTRAINT IF EXISTS check_entity_reference;

ALTER TABLE IF EXISTS catalog_validation_rules
ADD CONSTRAINT check_entity_reference 
CHECK (
  target_entity IS NOT NULL 
  OR target_entity_id IS NOT NULL 
  OR COALESCE(array_length(target_entity_ids, 1), 0) > 0
);

-- Create index on target_entity_id for faster lookups
DO $do$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class WHERE relkind='r' AND relname='catalog_validation_rules') THEN
    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_validation_rules_entity_id') THEN
      CREATE INDEX idx_validation_rules_entity_id ON catalog_validation_rules(target_entity_id);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_validation_rules_entity_ids') THEN
      CREATE INDEX idx_validation_rules_entity_ids ON catalog_validation_rules USING GIN(target_entity_ids);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relkind='i' AND relname='idx_validation_rules_datasource') THEN
      CREATE INDEX idx_validation_rules_datasource ON catalog_validation_rules(datasource_id);
    END IF;
  END IF;
END
$do$;

-- Add comments
DO $do$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class WHERE relkind='r' AND relname='catalog_validation_rules') THEN
    EXECUTE 'COMMENT ON COLUMN catalog_validation_rules.target_entity_id IS ''UUID reference to the target entity in fabric_defn (preferred over target_entity)''';
    EXECUTE 'COMMENT ON COLUMN catalog_validation_rules.target_entity_ids IS ''Array of UUIDs for multi-entity rules (preferred over target_entities)''';
    EXECUTE 'COMMENT ON COLUMN catalog_validation_rules.datasource_id IS ''Datasource scope for the validation rule''';
  END IF;
END
$do$;

-- Create a view to help with entity lookups
DO $do$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class WHERE relkind = 'r' AND relname = 'catalog_validation_rules') THEN
    EXECUTE $$
      CREATE OR REPLACE VIEW validation_rules_with_entities AS
      SELECT 
        vr.id,
        vr.tenant_id,
        vr.datasource_id,
        vr.rule_name,
        vr.rule_type,
        vr.description,
        vr.target_entity,
        vr.target_entity_id,
        vr.target_entity_ids,
        vr.condition_json,
        vr.severity,
        vr.is_active,
        vr.created_by,
        vr.created_at,
        vr.updated_at,
        COALESCE(fd.model_key, vr.target_entity) as entity_key,
        fd.title as entity_name,
        fd.id as entity_uuid
      FROM 
        catalog_validation_rules vr
      LEFT JOIN 
        fabric_defn fd ON (
          (vr.target_entity_id = fd.id) 
          OR (vr.target_entity = fd.model_key AND fd.is_current = true)
        )
      WHERE 
        fd.is_current = true OR vr.target_entity_id IS NULL;
    $$;
  ELSE
    -- create an empty fallback view so downstream code depending on this view doesn't fail
    EXECUTE $$
      CREATE OR REPLACE VIEW validation_rules_with_entities AS
      SELECT NULL::UUID AS id
      WHERE false;
    $$;
  END IF;
END
$do$;

-- Audit trail updates
ALTER TABLE IF EXISTS catalog_validation_rules_audit
ADD COLUMN IF NOT EXISTS target_entity_id_changes JSONB;

DO $do$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class WHERE relkind='r' AND relname='catalog_validation_rules_audit') THEN
    EXECUTE 'COMMENT ON COLUMN catalog_validation_rules_audit.target_entity_id_changes IS ''Tracks changes to entity_id references''';
  END IF;
END
$do$;
