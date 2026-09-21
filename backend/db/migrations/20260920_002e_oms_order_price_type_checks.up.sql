-- oms: add CHECK constraints for limit_price/stop_price vs order_type on oms.orders.
--
-- Must run AFTER 002d (so the default status_id is already set) and
-- BEFORE 002c (so the rejection capability is in place when the trigger is removed).
--
-- Execution order: 002d → 002e → 002c
--
-- Postgres CHECK constraints cannot contain subqueries, so the logic is wrapped in
-- STABLE SQL functions.  Postgres allows STABLE functions in CHECKs; note that if
-- ref.order_type codes change after this migration, the constraint will NOT
-- re-validate existing rows (STABLE only affects execution plans, not validity).
-- This is acceptable for rarely-changing ref data.
--
-- Defense-in-depth note: the AUTHORITATIVE validation is now in the business-object
-- semantic-terms layer (reject at save time). The CHECK constraint here is the
-- DB-level backstop against direct SQL, data imports, and CDC replay. If the two
-- layers ever conflict, the business-object layer wins.
--
-- NOT VALID is used because existing rows may already violate the constraint
-- (the trigger enforced this only for app-layer writes, not direct SQL). After this
-- migration runs, run the VALIDATE step below in a quiet window:
--   ALTER TABLE oms.orders VALIDATE CONSTRAINT limit_price_requires_limit_order_type;
--   ALTER TABLE oms.orders VALIDATE CONSTRAINT stop_price_requires_stop_order_type;
-- Each VALIDATE takes an ACCESS EXCLUSIVE lock briefly — schedule during low traffic.

BEGIN;

-- Resolve limit-order-type IDs and guard against missing ref data
DO $$
DECLARE _id INT;
BEGIN
    SELECT id INTO _id FROM ref.order_type
    WHERE code = 'LIMIT' AND is_active = TRUE LIMIT 1;
    IF _id IS NULL THEN
        RAISE EXCEPTION 'ref.order_type LIMIT is missing or inactive';
    END IF;
END $$;

-- Create STABLE lookup functions (Postgres allows these in CHECK constraints)

CREATE OR REPLACE FUNCTION oms.is_limit_order_type(ot_id INT)
RETURNS BOOLEAN
LANGUAGE sql STABLE
AS $$
    SELECT EXISTS (
        SELECT 1 FROM ref.order_type
        WHERE id = ot_id
          AND code = ANY(ARRAY['LIMIT','STOP_LIMIT','LOC','LOO','HYBRID'])
          AND is_active = TRUE
    );
$$;

CREATE OR REPLACE FUNCTION oms.is_stop_order_type(ot_id INT)
RETURNS BOOLEAN
LANGUAGE sql STABLE
AS $$
    SELECT EXISTS (
        SELECT 1 FROM ref.order_type
        WHERE id = ot_id
          AND code = ANY(ARRAY['STOP','STOP_LIMIT'])
          AND is_active = TRUE
    );
$$;

-- limit_price requires limit-type order
ALTER TABLE oms.orders ADD CONSTRAINT limit_price_requires_limit_order_type
    CHECK (limit_price IS NULL OR oms.is_limit_order_type(order_type_id))
    NOT VALID;

-- stop_price requires stop-type order
ALTER TABLE oms.orders ADD CONSTRAINT stop_price_requires_stop_order_type
    CHECK (stop_price IS NULL OR oms.is_stop_order_type(order_type_id))
    NOT VALID;

COMMIT;

-- =============================================================================
-- POST-MIGRATION VALIDATION (run separately in a quiet window)
-- =============================================================================
-- The ADD CONSTRAINT ... NOT VALID above does not check existing rows.
-- To validate them without holding a long lock, run each VALIDATE separately:
--
-- BEGIN;
-- ALTER TABLE oms.orders VALIDATE CONSTRAINT limit_price_requires_limit_order_type;
-- COMMIT;
--
-- BEGIN;
-- ALTER TABLE oms.orders VALIDATE CONSTRAINT stop_price_requires_stop_order_type;
-- COMMIT;
--
-- Each VALIDATE takes a brief ACCESS EXCLUSIVE lock. If either fails with
-- a constraint violation, rows exist that the trigger was silently allowing
-- (direct SQL, imports, or CDC replay). Correct those rows and retry the VALIDATE.
-- =============================================================================
