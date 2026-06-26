-- =============================================
-- Metric: value_at_risk
-- Category: market_risk
-- Governance: golden
-- Engine: oracle
-- Generated on: Sat Sep 13 17:25:55 EDT 2025
-- =============================================

-- View Definition
CREATE VIEW value_at_risk AS SELECT STDDEV(pr.daily_return) * SQRT(hp.holding_days) * cf.confidence_multiplier AS value FROM portfolio_returns pr CROSS JOIN (SELECT holding_days FROM holding_periods WHERE rownum = 1) hp CROSS JOIN (SELECT confidence_multiplier FROM confidence_factors WHERE rownum = 1) cf GROUP BY pr.entity_id, pr.as_of_date;

-- Preaggregation Strategy
CREATE MATERIALIZED VIEW mv_var REFRESH COMPLETE START WITH SYSDATE NEXT SYSDATE + 1/24 AS SELECT * FROM value_at_risk;

-- Performance Notes: Scheduled refresh for daily risk calculations

-- Grant permissions (customize as needed)
-- GRANT SELECT ON value_at_risk TO reporting_users;

