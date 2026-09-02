-- Adds the masking_pattern column that EntitlementsService.fetchEntitlementsForUser
-- already expects on bp_field_permissions (see add_masking_consolidation.sql step 1,
-- which was authored but never applied to this environment). Additive/idempotent.
ALTER TABLE bp_field_permissions ADD COLUMN IF NOT EXISTS masking_pattern VARCHAR(200);
