-- =============================================
-- Metric: customer_lifetime_value
-- DirectQuery Compatibility: High - CLV calculation
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW customer_lifetime_value AS SELECT SUM(cs.avg_value * cs.frequency * cs.years) AS value FROM customer_segments cs GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON customer_lifetime_value TO reporting_users;

