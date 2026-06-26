-- =============================================
-- Metric: deposit_stability_ratio
-- DirectQuery Compatibility: High - Conditional sum with CASE statement
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW deposit_stability_ratio AS SELECT SUM(CASE WHEN d.stability_classification = 'stable' THEN d.balance ELSE 0 END) / SUM(d.balance) AS value FROM deposits d GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON deposit_stability_ratio TO reporting_users;

