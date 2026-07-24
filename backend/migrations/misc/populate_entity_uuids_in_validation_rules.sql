-- Migration: Populate target_entity_id in validation rules from fabric_defn
-- This migration:
-- 1. Updates target_entity_id for each validation rule by matching target_entity name to fabric_defn model_key
-- 2. Populates target_entity_ids array with UUIDs for related entities
-- 3. Sets datasource_id from context (this may need manual population per tenant/datasource)

-- First, update target_entity_id by joining with fabric_defn
-- For rules with target_entity = 'employee', find the corresponding fabric_defn.id
UPDATE catalog_validation_rules cvr
SET target_entity_id = fd.id
FROM fabric_defn fd
WHERE cvr.target_entity_id IS NULL
  AND cvr.target_entity = fd.model_key
  AND fd.is_current = true
  AND cvr.tenant_id = fd.tenant_id;

-- For rules that don't have a direct single entity match,
-- we can populate target_entity_ids with UUIDs of all matching entities
-- This handles the case where target_entities array has multiple entity names
UPDATE catalog_validation_rules cvr
SET target_entity_ids = (
  SELECT ARRAY_AGG(fd.id ORDER BY fd.id)
  FROM fabric_defn fd
  WHERE fd.is_current = true
    AND fd.tenant_id = cvr.tenant_id
    AND (
      -- Match single entity
      (cvr.target_entity = fd.model_key AND ARRAY_LENGTH(cvr.target_entities, 1) = 1)
      OR
      -- Match any entity in the target_entities array
      (cvr.target_entities && ARRAY[fd.model_key]::text[])
    )
)
WHERE cvr.target_entity_ids = ARRAY[]::uuid[] OR cvr.target_entity_ids IS NULL;

-- Log results
SELECT 
  COUNT(*) as total_rules,
  COUNT(CASE WHEN target_entity_id IS NOT NULL THEN 1 END) as rules_with_entity_id,
  COUNT(CASE WHEN target_entity_ids != ARRAY[]::uuid[] THEN 1 END) as rules_with_entity_ids
FROM catalog_validation_rules;
