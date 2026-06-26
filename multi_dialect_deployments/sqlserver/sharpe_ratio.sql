-- =============================================
-- Metric: sharpe_ratio
-- Category: risk_adjusted_performance
-- Governance: golden
-- Engine: sqlserver
-- Generated on: Sat Sep 13 17:25:56 EDT 2025
-- =============================================

-- View Definition
CREATE VIEW sharpe_ratio AS SELECT (AVG(pr.daily_return) - rfr.risk_free_rate) / NULLIF(STDEV(pr.daily_return), 0) AS value FROM portfolio_returns pr CROSS JOIN (SELECT TOP 1 risk_free_rate FROM risk_free_rates ORDER BY as_of_date DESC) rfr GROUP BY pr.entity_id, pr.as_of_date;

-- Preaggregation Strategy
CREATE NONCLUSTERED INDEX idx_sharpe ON sharpe_ratio(entity_id, as_of_date) INCLUDE (value);

-- Performance Notes: Include computed columns in covering indexes

-- Grant permissions (customize as needed)
-- GRANT SELECT ON sharpe_ratio TO reporting_users;

