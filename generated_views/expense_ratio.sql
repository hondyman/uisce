-- =============================================
-- Metric: expense_ratio
-- DirectQuery Compatibility: High - Insurance expense ratio
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW expense_ratio AS SELECT SUM(ue.amount) / SUM(wp.amount) AS value FROM underwriting_expenses ue JOIN written_premiums wp ON ue.entity_id = wp.entity_id AND ue.as_of_date = wp.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON expense_ratio TO reporting_users;

