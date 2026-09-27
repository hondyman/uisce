-- 0012_mdm_rating_internal_override.down.sql
BEGIN;
DROP POLICY IF EXISTS rating_internal_override_tenant_write ON mdm.rating_internal_override;
DROP POLICY IF EXISTS rating_internal_override_tenant_read  ON mdm.rating_internal_override;
DROP TABLE IF EXISTS mdm.rating_internal_override CASCADE;
COMMIT;
