-- =============================================
-- Metric: infection_rate
-- DirectQuery Compatibility: High - Infection rate
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW infection_rate AS SELECT (SUM(hi.count) / SUM(pd.days)) * 1000 AS value FROM hospital_infections hi JOIN patient_days pd ON hi.entity_id = pd.entity_id AND hi.as_of_date = pd.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON infection_rate TO reporting_users;

