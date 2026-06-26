-- =============================================
-- Metric: credit_risk_concentration
-- DirectQuery Compatibility: Medium - Conditional aggregation may require tuning
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW credit_risk_concentration AS SELECT SUM(CASE WHEN l.principal_amount > ct.large_borrower_threshold THEN l.principal_amount ELSE 0 END) / SUM(l.principal_amount) AS value FROM loans l CROSS JOIN (SELECT large_borrower_threshold FROM concentration_thresholds) ct GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON credit_risk_concentration TO reporting_users;

