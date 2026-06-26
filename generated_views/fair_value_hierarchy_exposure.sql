-- =============================================
-- Metric: fair_value_hierarchy_exposure
-- DirectQuery Compatibility: Medium - FV hierarchy exposure
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW fair_value_hierarchy_exposure AS SELECT SUM(p.fair_value) AS value FROM positions p WHERE p.fv_hierarchy_level = (SELECT level_filter FROM level_filters LIMIT 1) GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON fair_value_hierarchy_exposure TO reporting_users;

