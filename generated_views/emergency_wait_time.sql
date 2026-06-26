-- =============================================
-- Metric: emergency_wait_time
-- DirectQuery Compatibility: High - Average wait time
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW emergency_wait_time AS SELECT AVG(ewt.minutes) AS value FROM emergency_wait_times ewt GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON emergency_wait_time TO reporting_users;

