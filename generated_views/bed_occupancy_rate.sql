-- =============================================
-- Metric: bed_occupancy_rate
-- DirectQuery Compatibility: High - Bed occupancy
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW bed_occupancy_rate AS SELECT SUM(bo.count) / SUM(bc.count) AS value FROM bed_occupancy bo JOIN bed_capacity bc ON bo.entity_id = bc.entity_id AND bo.as_of_date = bc.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON bed_occupancy_rate TO reporting_users;

