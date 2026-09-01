-- backend/db/migrations/000063_fix_identity_profile_mappings.down.sql
DROP INDEX IF EXISTS security.idx_idp_mappings;
DROP TABLE IF EXISTS security.identity_profile_mappings;
