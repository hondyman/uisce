-- =============================================
-- Metric: physician_productivity
-- DirectQuery Compatibility: High - Physician productivity
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW physician_productivity AS SELECT SUM(pv.count) / SUM(ap.count) AS value FROM patient_visits pv JOIN active_physicians ap ON pv.entity_id = ap.entity_id AND pv.as_of_date = ap.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON physician_productivity TO reporting_users;

