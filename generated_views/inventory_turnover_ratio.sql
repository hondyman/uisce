-- =============================================
-- Metric: inventory_turnover_ratio
-- DirectQuery Compatibility: High - Inventory turnover
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW inventory_turnover_ratio AS SELECT SUM(s.cogs) / AVG(il.value) AS value FROM sales_data s JOIN inventory_levels il ON s.entity_id = il.entity_id AND s.as_of_date = il.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON inventory_turnover_ratio TO reporting_users;

