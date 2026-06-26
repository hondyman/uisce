-- =============================================
-- Metric: information_ratio
-- DirectQuery Compatibility: Medium - Information ratio
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW information_ratio AS SELECT AVG(ar.active_return) / STDDEV_POP(ar.active_return) AS value FROM active_returns ar GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON information_ratio TO reporting_users;

