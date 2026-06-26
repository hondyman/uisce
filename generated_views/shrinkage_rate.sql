-- =============================================
-- Metric: shrinkage_rate
-- DirectQuery Compatibility: High - Shrinkage rate
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW shrinkage_rate AS SELECT SUM(isv.value) / SUM(iv.amount) AS value FROM inventory_shrinkage isv JOIN inventory_value iv ON isv.entity_id = iv.entity_id AND isv.as_of_date = iv.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON shrinkage_rate TO reporting_users;

