-- =============================================
-- Metric: reserve_adequacy_ratio
-- DirectQuery Compatibility: High - Reserve adequacy
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW reserve_adequacy_ratio AS SELECT SUM(lr.amount) / SUM(ap.projected_amount) AS value FROM loss_reserves lr JOIN actuarial_projections ap ON lr.entity_id = ap.entity_id AND lr.as_of_date = ap.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON reserve_adequacy_ratio TO reporting_users;

