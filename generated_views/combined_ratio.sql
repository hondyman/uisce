-- =============================================
-- Metric: combined_ratio
-- DirectQuery Compatibility: High - Insurance combined ratio
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW combined_ratio AS SELECT (SUM(c.amount) + SUM(ue.amount)) / SUM(ep.amount) AS value FROM claims c JOIN underwriting_expenses ue ON c.entity_id = ue.entity_id AND c.as_of_date = ue.as_of_date JOIN earned_premiums ep ON c.entity_id = ep.entity_id AND c.as_of_date = ep.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON combined_ratio TO reporting_users;

