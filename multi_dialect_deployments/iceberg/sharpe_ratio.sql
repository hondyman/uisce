-- =============================================
-- Metric: sharpe_ratio
-- Category: risk_adjusted_performance
-- Governance: golden
-- Engine: iceberg
-- Generated on: Sat Sep 13 17:25:55 EDT 2025
-- =============================================

-- View Definition
CREATE TABLE sharpe_ratio USING iceberg PARTITIONED BY (entity_id) AS SELECT pr.entity_id, pr.as_of_date, (AVG(pr.daily_return) - rfr.risk_free_rate) / NULLIF(STDDEV(pr.daily_return), 0) AS value FROM portfolio_returns pr CROSS JOIN (SELECT risk_free_rate FROM risk_free_rates ORDER BY as_of_date DESC LIMIT 1) rfr GROUP BY pr.entity_id, pr.as_of_date;

-- Preaggregation Strategy
ALTER TABLE sharpe_ratio ADD PARTITION FIELD entity_id;

-- Performance Notes: Partition by entity for tenant isolation

-- Grant permissions (customize as needed)
-- GRANT SELECT ON sharpe_ratio TO reporting_users;

