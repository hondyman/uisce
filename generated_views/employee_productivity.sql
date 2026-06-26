-- =============================================
-- Metric: employee_productivity
-- DirectQuery Compatibility: High - Employee productivity
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW employee_productivity AS SELECT SUM(ss.revenue) / SUM(eh.hours) AS value FROM store_sales ss JOIN employee_hours eh ON ss.entity_id = eh.entity_id AND ss.as_of_date = eh.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON employee_productivity TO reporting_users;

