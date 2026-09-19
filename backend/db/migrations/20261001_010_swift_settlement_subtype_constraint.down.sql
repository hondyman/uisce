-- Rollback: restore the original 6-subtype constraint.
ALTER TABLE cash_flow.settlement
  DROP CONSTRAINT ck_cash_flow_subtype;

ALTER TABLE cash_flow.settlement
  ADD CONSTRAINT ck_cash_flow_subtype CHECK (subtype_code = ANY (ARRAY[
    'dividend',
    'coupon_fixed_income',
    'capital_call',
    'lp_distribution',
    'corporate_action',
    'expense_fee'
  ]));
