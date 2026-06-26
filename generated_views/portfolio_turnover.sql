-- =============================================
-- Metric: portfolio_turnover
-- DirectQuery Compatibility: High - Turnover calculation
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW portfolio_turnover AS SELECT SUM(ABS(t.transaction_amount)) / (2 * AVG(pv.total_value)) AS value FROM trades t JOIN portfolio_values pv ON t.entity_id = pv.entity_id AND t.as_of_date = pv.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON portfolio_turnover TO reporting_users;

