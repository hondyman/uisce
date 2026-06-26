-- =============================================
-- Metric: effective_interest_income
-- DirectQuery Compatibility: High - EIR calculation
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW effective_interest_income AS SELECT oca.opening_carrying_amount * eir.effective_interest_rate AS value FROM opening_carrying_amounts oca JOIN effective_interest_rates eir ON oca.entity_id = eir.entity_id AND oca.as_of_date = eir.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON effective_interest_income TO reporting_users;

