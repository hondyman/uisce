-- =============================================
-- Metric: leverage_ratio
-- DirectQuery Compatibility: High - Leverage ratio
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW leverage_ratio AS SELECT SUM(t1c.amount) / SUM(te.amount) AS value FROM tier1_capital t1c JOIN total_exposure te ON t1c.entity_id = te.entity_id AND t1c.as_of_date = te.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON leverage_ratio TO reporting_users;

