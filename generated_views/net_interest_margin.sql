-- =============================================
-- Metric: net_interest_margin
-- DirectQuery Compatibility: High - Simple aggregations fold well
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW net_interest_margin AS SELECT (SUM(ii.amount) - SUM(ie.amount)) / AVG(a.average_balance) AS value FROM interest_income ii, interest_expense ie, assets a GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON net_interest_margin TO reporting_users;

