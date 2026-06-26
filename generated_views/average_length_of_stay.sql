-- =============================================
-- Metric: average_length_of_stay
-- DirectQuery Compatibility: High - Average LOS
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW average_length_of_stay AS SELECT AVG(ha.days) AS value FROM hospital_admissions ha GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON average_length_of_stay TO reporting_users;

