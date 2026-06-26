-- =============================================
-- Metric: average_cost_per_patient
-- DirectQuery Compatibility: High - Average cost
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW average_cost_per_patient AS SELECT SUM(tc.amount) / SUM(pc.count) AS value FROM treatment_costs tc JOIN patient_counts pc ON tc.entity_id = pc.entity_id AND tc.as_of_date = pc.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON average_cost_per_patient TO reporting_users;

