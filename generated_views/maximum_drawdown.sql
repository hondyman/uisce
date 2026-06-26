-- =============================================
-- Metric: maximum_drawdown
-- DirectQuery Compatibility: High - Simple minimum
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW maximum_drawdown AS SELECT MIN(dd.drawdown) AS value FROM rolling_drawdowns dd GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON maximum_drawdown TO reporting_users;

