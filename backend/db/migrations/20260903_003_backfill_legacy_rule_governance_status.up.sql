-- 20260903_003_backfill_legacy_rule_governance_status.up.sql
-- Backfill governance_status for pre-Phase-4 legacy validation rules.
-- Legacy rules that predate governance tracking were actively enforced in production
-- and are grandfathered into 'published' status.
-- Normalizes any legacy 'review' aliases to 'submitted_for_review'.

UPDATE catalog_node
SET properties = jsonb_set(
    COALESCE(properties, '{}'::jsonb),
    '{governance_status}',
    '"published"'
)
WHERE node_type_id = (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule')
  AND (
      (properties->>'governance_status') IS NULL
      OR (properties->>'governance_status') = ''
  );

UPDATE catalog_node
SET properties = jsonb_set(
    properties,
    '{governance_status}',
    '"submitted_for_review"'
)
WHERE node_type_id = (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule')
  AND (properties->>'governance_status') = 'review';
