-- Add source_table column to support table-backed lookups
-- This allows lookups to pull values from a source table instead of pre-populated lookup_values

ALTER TABLE lookups 
ADD COLUMN IF NOT EXISTS source_table TEXT NULL;

-- Add comment explaining the column
COMMENT ON COLUMN lookups.source_table IS 'Optional: if set, lookup values come from this table instead of lookup_values table. Should contain id and name columns.';
