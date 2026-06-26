-- =============================================
-- Metric: market_risk_capital
-- DirectQuery Compatibility: High - Market risk capital
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW market_risk_capital AS SELECT SUM(tp.var_amount) * cf.risk_multiplier AS value FROM trading_positions tp CROSS JOIN (SELECT risk_multiplier FROM confidence_factors LIMIT 1) cf GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON market_risk_capital TO reporting_users;

