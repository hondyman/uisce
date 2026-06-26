-- =============================================
-- Metric: underwriting_profit
-- DirectQuery Compatibility: High - Simple arithmetic
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW underwriting_profit AS SELECT SUM(ep.amount) - SUM(c.amount) - SUM(ue.amount) AS value FROM earned_premiums ep JOIN claims c ON ep.entity_id = c.entity_id AND ep.as_of_date = c.as_of_date JOIN underwriting_expenses ue ON ep.entity_id = ue.entity_id AND ep.as_of_date = ue.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON underwriting_profit TO reporting_users;

