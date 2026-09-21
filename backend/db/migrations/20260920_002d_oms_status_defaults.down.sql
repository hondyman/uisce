-- Rollback for 20260920_002d_oms_status_defaults.up.sql
-- Restores NULL as the implicit default for status_id on all four oms tables.

BEGIN;

ALTER TABLE oms.orders      ALTER COLUMN status_id DROP DEFAULT;
ALTER TABLE oms.order_slice ALTER COLUMN status_id DROP DEFAULT;
ALTER TABLE oms.allocation  ALTER COLUMN status_id DROP DEFAULT;
ALTER TABLE oms.settlement  ALTER COLUMN status_id DROP DEFAULT;

COMMIT;
