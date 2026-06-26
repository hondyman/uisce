-- =============================================
-- Metric: claims_severity
-- DirectQuery Compatibility: High - Severity calculation
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW claims_severity AS SELECT SUM(c.amount) / COUNT(DISTINCT c.id) AS value FROM claims c GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON claims_severity TO reporting_users;

