-- =============================================
-- Metric: capital_adequacy_ratio
-- DirectQuery Compatibility: High - Regulatory ratio
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW capital_adequacy_ratio AS SELECT SUM(rc.amount) / SUM(rwa.weighted_balance) AS value FROM regulatory_capital rc JOIN risk_weighted_assets rwa ON rc.entity_id = rwa.entity_id AND rc.as_of_date = rwa.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON capital_adequacy_ratio TO reporting_users;

