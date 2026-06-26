-- =============================================
-- Metric: beta_coefficient
-- Category: market_risk
-- Governance: golden
-- Engine: oracle
-- Generated on: Sat Sep 13 17:25:55 EDT 2025
-- =============================================

-- View Definition
CREATE VIEW beta_coefficient AS SELECT SUM((pmr.portfolio_return - pm.avg_portfolio) * (pmr.market_return - mm.avg_market)) / NULLIF(SUM(POWER(pmr.market_return - mm.avg_market, 2)), 0) AS value FROM portfolio_market_returns pmr CROSS JOIN (SELECT AVG(portfolio_return) avg_portfolio FROM portfolio_market_returns) pm CROSS JOIN (SELECT AVG(market_return) avg_market FROM portfolio_market_returns) mm GROUP BY pmr.entity_id, pmr.as_of_date;

-- Preaggregation Strategy
CREATE MATERIALIZED VIEW mv_beta ENABLE QUERY REWRITE AS SELECT * FROM beta_coefficient;

-- Performance Notes: Query rewrite for complex statistical calculations

-- Grant permissions (customize as needed)
-- GRANT SELECT ON beta_coefficient TO reporting_users;

