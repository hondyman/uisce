-- =============================================
-- Metric: non_performing_loan_ratio
-- DirectQuery Compatibility: High - Conditional ratio
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW non_performing_loan_ratio AS SELECT SUM(CASE WHEN l.performance_status = 'non_performing' THEN l.outstanding_balance ELSE 0 END) / SUM(l.outstanding_balance) AS value FROM loans l GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON non_performing_loan_ratio TO reporting_users;

