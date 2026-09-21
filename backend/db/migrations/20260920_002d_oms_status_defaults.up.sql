-- oms: add explicit column defaults for status_id on the four oms tables.
--
-- Context:
--   oms.default_order_status() does TWO things:
--     1. Sets status_id to a sensible default on INSERT if null
--     2. RAISE EXCEPTION if limit_price is set on a non-limit order_type
--        (and same for stop_price / stop order_type)
--
-- This migration addresses item 1 only — column DEFAULTs.
-- The RAISE EXCEPTION validation (item 2) is replaced by CHECK constraints
-- in 20260920_002e_oms_order_price_type_checks.up.sql, which must run
-- BEFORE 002c so 002c can drop the trigger safely.
--
-- Execution order: 002d → 002e → 002c
--
-- Postgres does not allow subqueries in SET DEFAULT, so we resolve IDs inside
-- a DO block using format() with %L. A guard raises if the ref data is missing.
--
-- Rollback: drop the DEFAULTs (restore implicit NULL default, pre-existing state)

BEGIN;

-- oms.orders: status default 'NEW'
DO $$
DECLARE _id INT;
BEGIN
    SELECT id INTO _id FROM ref.order_status WHERE code = 'NEW' AND is_active = TRUE;
    IF _id IS NULL THEN
        RAISE EXCEPTION 'ref.order_status code NEW is missing or inactive — cannot set default';
    END IF;
    EXECUTE format('ALTER TABLE oms.orders ALTER COLUMN status_id SET DEFAULT %L', _id);
END $$;

-- oms.order_slice: status default 'WORKING'
DO $$
DECLARE _id INT;
BEGIN
    SELECT id INTO _id FROM ref.order_status WHERE code = 'WORKING' AND is_active = TRUE;
    IF _id IS NULL THEN
        RAISE EXCEPTION 'ref.order_status code WORKING is missing or inactive — cannot set default';
    END IF;
    EXECUTE format('ALTER TABLE oms.order_slice ALTER COLUMN status_id SET DEFAULT %L', _id);
END $$;

-- oms.allocation: status default 'PENDING'
DO $$
DECLARE _id INT;
BEGIN
    SELECT id INTO _id FROM ref.allocation_status WHERE code = 'PENDING' AND is_active = TRUE;
    IF _id IS NULL THEN
        RAISE EXCEPTION 'ref.allocation_status code PENDING is missing or inactive — cannot set default';
    END IF;
    EXECUTE format('ALTER TABLE oms.allocation ALTER COLUMN status_id SET DEFAULT %L', _id);
END $$;

-- oms.settlement: status default 'PENDING'
DO $$
DECLARE _id INT;
BEGIN
    SELECT id INTO _id FROM ref.settlement_status WHERE code = 'PENDING' AND is_active = TRUE;
    IF _id IS NULL THEN
        RAISE EXCEPTION 'ref.settlement_status code PENDING is missing or inactive — cannot set default';
    END IF;
    EXECUTE format('ALTER TABLE oms.settlement ALTER COLUMN status_id SET DEFAULT %L', _id);
END $$;

COMMIT;
