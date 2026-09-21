-- Rollback for 20260920_002e_oms_order_price_type_checks.up.sql

BEGIN;

ALTER TABLE oms.orders DROP CONSTRAINT IF EXISTS limit_price_requires_limit_order_type;
ALTER TABLE oms.orders DROP CONSTRAINT IF EXISTS stop_price_requires_stop_order_type;

DROP FUNCTION IF EXISTS oms.is_limit_order_type(INT);
DROP FUNCTION IF EXISTS oms.is_stop_order_type(INT);

COMMIT;
