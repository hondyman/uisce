-- =============================================
-- Metric: impairment_loss_incurred
-- DirectQuery Compatibility: High - Impairment loss
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW impairment_loss_incurred AS SELECT GREATEST(0, ca.carrying_amount - ra.recoverable_amount) AS value FROM carrying_amount ca JOIN recoverable_amount ra ON ca.entity_id = ra.entity_id AND ca.as_of_date = ra.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON impairment_loss_incurred TO reporting_users;

