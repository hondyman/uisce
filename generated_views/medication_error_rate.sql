-- =============================================
-- Metric: medication_error_rate
-- DirectQuery Compatibility: High - Medication error rate
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW medication_error_rate AS SELECT (SUM(me.count) / SUM(da.count)) * 1000 AS value FROM medication_errors me JOIN doses_administered da ON me.entity_id = da.entity_id AND me.as_of_date = da.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON medication_error_rate TO reporting_users;

