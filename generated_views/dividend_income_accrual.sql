-- =============================================
-- Metric: dividend_income_accrual
-- DirectQuery Compatibility: High - Dividend accrual
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW dividend_income_accrual AS SELECT SUM(de.dividend_amount) AS value FROM dividend_events de WHERE de.ex_date <= de.period_end GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON dividend_income_accrual TO reporting_users;

