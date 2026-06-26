-- =============================================
-- Metric: carrying_amount_rollforward
-- DirectQuery Compatibility: Medium - Complex rollforward, may need optimization
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW carrying_amount_rollforward AS SELECT oca.opening_carrying_amount + eii.value - ccr.coupon_cash_received + pda.value - clac.ecl_closing + clac.ecl_opening + fr.fx_remeasurement AS value FROM opening_carrying_amounts oca JOIN effective_interest_income eii ON oca.entity_id = eii.entity_id AND oca.as_of_date = eii.as_of_date JOIN coupon_cash_received ccr ON oca.entity_id = ccr.entity_id AND oca.as_of_date = ccr.as_of_date JOIN premium_discount_amortization pda ON oca.entity_id = pda.entity_id AND oca.as_of_date = pda.as_of_date JOIN credit_loss_allowance_change clac ON oca.entity_id = clac.entity_id AND oca.as_of_date = clac.as_of_date JOIN fx_remeasurement fr ON oca.entity_id = fr.entity_id AND oca.as_of_date = fr.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON carrying_amount_rollforward TO reporting_users;

