-- =============================================
-- Metric: policyholder_surplus_ratio
-- DirectQuery Compatibility: High - Surplus ratio
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW policyholder_surplus_ratio AS SELECT SUM(ps.amount) / SUM(l.amount) AS value FROM policyholder_surplus ps JOIN liabilities l ON ps.entity_id = l.entity_id AND ps.as_of_date = l.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON policyholder_surplus_ratio TO reporting_users;

