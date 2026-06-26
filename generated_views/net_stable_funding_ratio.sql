-- =============================================
-- Metric: net_stable_funding_ratio
-- DirectQuery Compatibility: High - NSFR calculation
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW net_stable_funding_ratio AS SELECT SUM(asf.amount) / SUM(rsf.amount) AS value FROM available_stable_funding asf JOIN required_stable_funding rsf ON asf.entity_id = rsf.entity_id AND asf.as_of_date = rsf.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON net_stable_funding_ratio TO reporting_users;

