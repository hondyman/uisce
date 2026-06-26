-- =============================================
-- Metric: value_at_risk
-- Category: market_risk
-- Governance: golden
-- Engine: postgres
-- Generated on: Sat Sep 13 17:25:55 EDT 2025
-- =============================================

-- View Definition
CREATE VIEW value_at_risk AS SELECT STDDEV(pr.daily_return) * SQRT(hp.holding_days::DOUBLE PRECISION) * cf.confidence_multiplier AS value FROM portfolio_returns pr CROSS JOIN (SELECT holding_days FROM holding_periods LIMIT 1) hp CROSS JOIN (SELECT confidence_multiplier FROM confidence_factors LIMIT 1) cf GROUP BY pr.entity_id, pr.as_of_date;

-- Preaggregation Strategy
CREATE MATERIALIZED VIEW mv_var AS SELECT * FROM value_at_risk WITH DATA; REFRESH MATERIALIZED VIEW CONCURRENTLY mv_var;

-- Performance Notes: Concurrent refresh to avoid blocking queries

-- Grant permissions (customize as needed)
-- GRANT SELECT ON value_at_risk TO reporting_users;

