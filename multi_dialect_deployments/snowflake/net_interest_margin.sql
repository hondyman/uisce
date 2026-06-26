-- =============================================
-- Metric: net_interest_margin
-- Category: profitability
-- Governance: golden
-- Engine: snowflake
-- Generated on: Sat Sep 13 17:25:55 EDT 2025
-- =============================================

-- View Definition
CREATE VIEW net_interest_margin AS SELECT (SUM(ii.amount) - SUM(ie.amount)) / NULLIF(AVG(a.average_balance), 0) AS value FROM interest_income ii JOIN interest_expense ie ON ii.entity_id = ie.entity_id AND ii.as_of_date = ie.as_of_date JOIN assets a ON ii.entity_id = a.entity_id AND ii.as_of_date = a.as_of_date GROUP BY ii.entity_id, ii.as_of_date;

-- Preaggregation Strategy
ALTER TABLE net_interest_margin CLUSTER BY (entity_id, as_of_date);

-- Performance Notes: Automatic clustering for query optimization

-- Grant permissions (customize as needed)
-- GRANT SELECT ON net_interest_margin TO reporting_users;

