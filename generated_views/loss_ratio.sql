-- =============================================
-- Metric: loss_ratio
-- DirectQuery Compatibility: High - Insurance loss ratio
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW loss_ratio AS SELECT SUM(c.amount) / SUM(ep.amount) AS value FROM claims c JOIN earned_premiums ep ON c.entity_id = ep.entity_id AND c.as_of_date = ep.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON loss_ratio TO reporting_users;

