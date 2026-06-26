-- Migration: Add support for Effective Dating Facilitator
-- Date: 2026-01-09

-- 1. Add enable_history flag to Business Objects
ALTER TABLE business_objects 
ADD COLUMN IF NOT EXISTS enable_history BOOLEAN DEFAULT FALSE;

-- 2. Add an index for performance on historical queries if needed
-- Assuming transactional data tables will have valid_from/valid_to columns
-- These should be added to the target tables by the client as needed.

-- Example for a target table:
-- ALTER TABLE trn_emp_data ADD COLUMN valid_from TIMESTAMP NOT NULL DEFAULT NOW();
-- ALTER TABLE trn_emp_data ADD COLUMN valid_to TIMESTAMP NOT NULL DEFAULT '9999-12-31 23:59:59';
-- CREATE INDEX idx_emp_temporal ON trn_emp_data (valid_from, valid_to);

-- 3. Update existing records (Optional - defaulting to FALSE is safe)
-- UPDATE business_objects SET enable_history = FALSE WHERE enable_history IS NULL;

-- 4. Verify Semantic Term flags
-- Since semantic terms are stored in catalog_node.properties (JSONB),
-- no schema change is required there. The Go models will handle the new 'is_effective_dated' key.
