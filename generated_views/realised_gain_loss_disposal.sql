-- =============================================
-- Metric: realised_gain_loss_disposal
-- DirectQuery Compatibility: High - Realised G/L
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW realised_gain_loss_disposal AS SELECT cr.consideration_received - cas.carrying_amount_at_sale AS value FROM consideration_received cr JOIN carrying_amount_at_sale cas ON cr.entity_id = cas.entity_id AND cr.as_of_date = cas.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON realised_gain_loss_disposal TO reporting_users;

