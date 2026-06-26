-- =============================================
-- Metric: sharpe_ratio
-- Category: risk_adjusted_performance
-- Governance: golden
-- Engine: oracle
-- Generated on: Sat Sep 13 17:25:55 EDT 2025
-- =============================================

-- View Definition
CREATE VIEW sharpe_ratio AS SELECT (AVG(pr.daily_return) - rfr.risk_free_rate) / NULLIF(STDDEV(pr.daily_return), 0) AS value FROM portfolio_returns pr CROSS JOIN (SELECT risk_free_rate FROM risk_free_rates WHERE rownum = 1 ORDER BY as_of_date DESC) rfr GROUP BY pr.entity_id, pr.as_of_date;

-- Preaggregation Strategy
CREATE BITMAP INDEX idx_sharpe_entity ON sharpe_ratio(entity_id); CREATE BITMAP INDEX idx_sharpe_date ON sharpe_ratio(as_of_date);

-- Performance Notes: Bitmap indexes for low-cardinality dimensions

-- Grant permissions (customize as needed)
-- GRANT SELECT ON sharpe_ratio TO reporting_users;

