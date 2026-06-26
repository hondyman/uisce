-- =============================================
-- Metric: sortino_ratio
-- DirectQuery Compatibility: Medium - Downside deviation
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW sortino_ratio AS SELECT (AVG(pr.daily_return) - rfr.risk_free_rate) / STDDEV_POP(CASE WHEN pr.daily_return < rfr.risk_free_rate THEN pr.daily_return ELSE NULL END) AS value FROM portfolio_returns pr CROSS JOIN (SELECT risk_free_rate FROM risk_free_rates LIMIT 1) rfr GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON sortino_ratio TO reporting_users;

