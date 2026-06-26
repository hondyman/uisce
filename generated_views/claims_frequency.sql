-- =============================================
-- Metric: claims_frequency
-- DirectQuery Compatibility: High - Frequency calculation
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW claims_frequency AS SELECT COUNT(DISTINCT c.id) / SUM(p.exposure_amount) AS value FROM claims c JOIN policies p ON c.entity_id = p.entity_id AND c.as_of_date = p.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON claims_frequency TO reporting_users;

