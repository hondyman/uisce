-- =============================================
-- Metric: win_rate
-- DirectQuery Compatibility: High - Win rate calculation
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW win_rate AS SELECT COUNT(DISTINCT CASE WHEN t.profit_loss > 0 THEN t.id ELSE NULL END) / COUNT(DISTINCT t.id) AS value FROM trades t GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON win_rate TO reporting_users;

