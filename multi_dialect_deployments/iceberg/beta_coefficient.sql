-- =============================================
-- Metric: beta_coefficient
-- Category: market_risk
-- Governance: golden
-- Engine: iceberg
-- Generated on: Sat Sep 13 17:25:55 EDT 2025
-- =============================================

-- View Definition
CREATE TABLE beta_coefficient USING iceberg PARTITIONED BY (entity_id, year(as_of_date)) AS SELECT pmr.entity_id, pmr.as_of_date, SUM((pmr.portfolio_return - pm.avg_portfolio) * (pmr.market_return - mm.avg_market)) / NULLIF(SUM(POWER(pmr.market_return - mm.avg_market, 2)), 0) AS value FROM portfolio_market_returns pmr CROSS JOIN (SELECT AVG(portfolio_return) as avg_portfolio FROM portfolio_market_returns) pm CROSS JOIN (SELECT AVG(market_return) as avg_market FROM portfolio_market_returns) mm GROUP BY pmr.entity_id, pmr.as_of_date;

-- Preaggregation Strategy
ALTER TABLE beta_coefficient ADD PARTITION FIELD year(as_of_date);

-- Performance Notes: Multi-level partitioning for analytical workloads

-- Grant permissions (customize as needed)
-- GRANT SELECT ON beta_coefficient TO reporting_users;

