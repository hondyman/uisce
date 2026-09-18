-- Migration: Extend ck_cash_flow_subtype to include SWIFT settlement subtypes.
--
-- Migration 007 added dvp_securities and free_of_payment to oms.subtype_registry
-- but did not patch the CHECK constraint on cash_flow.settlement, which only allows
-- the original 6 STI subtypes. Any attempt to INSERT a dvp_securities or
-- free_of_payment row fails at runtime with a check-constraint violation.
-- This is the constraint that sqlmock tests cannot catch.
--
-- Idempotent: safe to re-run. Uses IF EXISTS / DO block to avoid errors on
-- repeat application (e.g. migration runner catching up to hand-applied migrations).

ALTER TABLE cash_flow.settlement
  DROP CONSTRAINT IF EXISTS ck_cash_flow_subtype;

ALTER TABLE cash_flow.settlement
  ADD CONSTRAINT ck_cash_flow_subtype CHECK (subtype_code = ANY (ARRAY[
    'dividend',
    'coupon_fixed_income',
    'capital_call',
    'lp_distribution',
    'corporate_action',
    'expense_fee',
    -- SWIFT settlement subtypes (added by migration 007/009):
    'dvp_securities',
    'free_of_payment'
  ]));

