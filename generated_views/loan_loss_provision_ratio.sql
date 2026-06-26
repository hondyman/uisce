-- =============================================
-- Metric: loan_loss_provision_ratio
-- DirectQuery Compatibility: High - Standard ratio calculation
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW loan_loss_provision_ratio AS SELECT SUM(lp.provision_amount) / SUM(l.outstanding_balance) AS value FROM loan_provisions lp JOIN loans l ON lp.entity_id = l.entity_id AND lp.as_of_date = l.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON loan_loss_provision_ratio TO reporting_users;

