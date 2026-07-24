-- Migration: Add Hierarchical Validation Support
-- Date: 2025-10-20
-- Description: Adds full support for hierarchical validation of sub-entities.

-- Add hierarchy field path support
ALTER TABLE validation_rules 
ADD COLUMN IF NOT EXISTS field_path TEXT[] DEFAULT ARRAY[]::TEXT[];

-- Add aggregation support
ALTER TABLE validation_rules
ADD COLUMN IF NOT EXISTS aggregation_type VARCHAR(50),
ADD COLUMN IF NOT EXISTS aggregation_field VARCHAR(255);

-- Add sub-entity depth tracking
ALTER TABLE validation_rules
ADD COLUMN IF NOT EXISTS hierarchy_depth INT DEFAULT 0;

-- Create index for hierarchy queries
CREATE INDEX IF NOT EXISTS idx_validation_rules_hierarchy 
ON validation_rules(tenant_id, datasource_id, field_path);

-- Create index for depth queries
CREATE INDEX IF NOT EXISTS idx_validation_rules_hierarchy_depth 
ON validation_rules(tenant_id, datasource_id, hierarchy_depth);
-- Sample hierarchical rules moved into guarded block further below. They will only be inserted if the
-- `validation_rules.name` column exists to avoid errors when running against partial schemas.

-- Make inserts idempotent: avoid duplicates by using a unique constraint check when possible.
DO $$
BEGIN
  -- If the validation_rules table has a 'name' column, perform an existence check and insert sample rules.
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'validation_rules' AND column_name = 'name'
  ) THEN
    -- If a similar rule (by tenant, datasource and name) doesn't exist, insert it.
    IF NOT EXISTS (
      SELECT 1 FROM validation_rules vr
      WHERE vr.tenant_id = '00000000-0000-0000-0000-000000000000'
        AND vr.datasource_id = '11111111-1111-1111-1111-111111111111'
        AND vr.name = 'Line Item Quantity Check'
    ) THEN
      INSERT INTO validation_rules (
        tenant_id,
        datasource_id,
        name,
        entity,
        description,
        severity,
        condition,
        field_path,
        hierarchy_depth,
        is_active,
        created_at,
        updated_at
      ) VALUES (
        '00000000-0000-0000-0000-000000000000',
        '11111111-1111-1111-1111-111111111111',
        'Line Item Quantity Check',
        'Order',
        'Validates that line item quantities are reasonable',
        'error',
        '{
          "type": "hierarchy",
          "sub_entity": "line_items",
          "field": "qty",
          "operator": "greater_than",
          "value": 0,
          "parent_field": "total",
          "parent_operator": "greater_equal"
        }'::jsonb,
        ARRAY['line_items'],
        1,
        true,
        NOW(),
        NOW()
      );
    END IF;

    IF NOT EXISTS (
      SELECT 1 FROM validation_rules vr
      WHERE vr.tenant_id = '00000000-0000-0000-0000-000000000000'
        AND vr.datasource_id = '11111111-1111-1111-1111-111111111111'
        AND vr.name = 'Order Total Must Match Line Items'
    ) THEN
      INSERT INTO validation_rules (
        tenant_id,
        datasource_id,
        name,
        entity,
        description,
        severity,
        condition,
        field_path,
        aggregation_type,
        aggregation_field,
        hierarchy_depth,
        is_active,
        created_at,
        updated_at
      ) VALUES (
        '00000000-0000-0000-0000-000000000000',
        '11111111-1111-1111-1111-111111111111',
        'Order Total Must Match Line Items',
        'Order',
        'Validates order total matches sum of line items',
        'error',
        '{
          "type": "hierarchy_aggregate",
          "sub_entity": "line_items",
          "aggregation": "sum",
          "aggregation_field": "price",
          "parent_field": "total",
          "operator": "equals_aggregate"
        }'::jsonb,
        ARRAY['line_items'],
        NULL,
        NULL,
        1,
        true,
        NOW(),
        NOW()
      );
    END IF;
  ELSE
    RAISE NOTICE 'validation_rules.name column not present; skipping sample hierarchical inserts';
  END IF;
END$$;
