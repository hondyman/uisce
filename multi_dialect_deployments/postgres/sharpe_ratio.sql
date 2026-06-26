-- =============================================
-- Metric: sharpe_ratio
-- Category: risk_adjusted_performance
-- Governance: golden
-- Engine: postgres
-- Generated on: Sat Sep 13 17:25:55 EDT 2025
-- =============================================

-- View Definition
CREATE VIEW sharpe_ratio AS SELECT (AVG(pr.daily_return) - rfr.risk_free_rate) / NULLIF(STDDEV(pr.daily_return), 0) AS value FROM portfolio_returns pr CROSS JOIN (SELECT risk_free_rate FROM risk_free_rates ORDER BY as_of_date DESC LIMIT 1) rfr GROUP BY pr.entity_id, pr.as_of_date;

-- Preaggregation Strategy
CREATE MATERIALIZED VIEW mv_sharpe AS SELECT * FROM sharpe_ratio; CREATE INDEX CONCURRENTLY idx_sharpe_gist ON mv_sharpe USING GIST (entity_id, as_of_date);

-- Performance Notes: GiST indexes for multi-column optimization

-- Grant permissions (customize as needed)
-- GRANT SELECT ON sharpe_ratio TO reporting_users;

