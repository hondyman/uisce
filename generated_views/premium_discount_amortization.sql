-- =============================================
-- Metric: premium_discount_amortization
-- DirectQuery Compatibility: High - Amortization calculation
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW premium_discount_amortization AS SELECT (eii.value - ccr.coupon_cash_received) AS value FROM effective_interest_income eii JOIN coupon_cash_received ccr ON eii.entity_id = ccr.entity_id AND eii.as_of_date = ccr.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON premium_discount_amortization TO reporting_users;

