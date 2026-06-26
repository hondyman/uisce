-- =============================================
-- Metric: readmission_rate
-- DirectQuery Compatibility: High - Readmission rate
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW readmission_rate AS SELECT SUM(r.count) / SUM(pd.count) AS value FROM readmissions r JOIN patient_discharges pd ON r.entity_id = pd.entity_id AND r.as_of_date = pd.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON readmission_rate TO reporting_users;

