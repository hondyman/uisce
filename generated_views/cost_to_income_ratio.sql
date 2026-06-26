-- =============================================
-- Metric: cost_to_income_ratio
-- DirectQuery Compatibility: High - Simple ratio
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW cost_to_income_ratio AS SELECT SUM(e.amount) / SUM(i.amount) AS value FROM operating_expenses e JOIN operating_income i ON e.entity_id = i.entity_id AND e.as_of_date = i.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON cost_to_income_ratio TO reporting_users;

