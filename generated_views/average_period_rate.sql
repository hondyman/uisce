-- =============================================
-- Metric: average_period_rate
-- DirectQuery Compatibility: High - Average period rate
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW average_period_rate AS SELECT AVG(fr.fx_rate) AS value FROM fx_rates fr WHERE fr.date >= fr.period_start AND fr.date <= fr.period_end GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON average_period_rate TO reporting_users;

