-- =============================================
-- Metric: patient_satisfaction_score
-- DirectQuery Compatibility: High - Simple average
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW patient_satisfaction_score AS SELECT AVG(ps.rating) AS value FROM patient_surveys ps GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON patient_satisfaction_score TO reporting_users;

