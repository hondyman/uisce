-- =============================================
-- Metric: tier1_capital_ratio
-- DirectQuery Compatibility: High - Tier 1 ratio
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW tier1_capital_ratio AS SELECT SUM(t1c.amount) / SUM(rwa.weighted_amount) AS value FROM tier1_capital t1c JOIN risk_weighted_assets rwa ON t1c.entity_id = rwa.entity_id AND t1c.as_of_date = rwa.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON tier1_capital_ratio TO reporting_users;

