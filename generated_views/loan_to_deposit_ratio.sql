-- =============================================
-- Metric: loan_to_deposit_ratio
-- DirectQuery Compatibility: High - Simple ratio
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW loan_to_deposit_ratio AS SELECT SUM(l.outstanding_balance) / SUM(d.balance) AS value FROM loans l JOIN deposits d ON l.entity_id = d.entity_id AND l.as_of_date = d.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON loan_to_deposit_ratio TO reporting_users;

