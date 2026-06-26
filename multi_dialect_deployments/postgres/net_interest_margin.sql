-- =============================================
-- Metric: net_interest_margin
-- Category: profitability
-- Governance: golden
-- Engine: postgres
-- Generated on: Sat Sep 13 17:25:55 EDT 2025
-- =============================================

-- View Definition
CREATE VIEW net_interest_margin AS SELECT (SUM(ii.amount) - SUM(ie.amount)) / NULLIF(AVG(a.average_balance)::DECIMAL, 0) AS value FROM interest_income ii JOIN interest_expense ie ON ii.entity_id = ie.entity_id AND ii.as_of_date = ie.as_of_date JOIN assets a ON ii.entity_id = a.entity_id AND ii.as_of_date = a.as_of_date GROUP BY ii.entity_id, ii.as_of_date;

-- Preaggregation Strategy
CREATE MATERIALIZED VIEW mv_nim AS SELECT * FROM net_interest_margin; CREATE INDEX idx_nim_entity_date ON mv_nim(entity_id, as_of_date);

-- Performance Notes: Materialized views with concurrent indexing

-- Grant permissions (customize as needed)
-- GRANT SELECT ON net_interest_margin TO reporting_users;

