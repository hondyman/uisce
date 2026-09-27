-- 0010_staging_rating_incoming.down.sql
BEGIN;
DROP POLICY IF EXISTS rating_inc_tenant_write ON staging.rating_incoming;
DROP POLICY IF EXISTS rating_inc_tenant_read  ON staging.rating_incoming;
DROP TABLE IF EXISTS staging.rating_incoming CASCADE;
COMMIT;
