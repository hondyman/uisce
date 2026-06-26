-- =============================================
-- Metric: beta_coefficient
-- DirectQuery Compatibility: Low - Complex correlation calculation, may need pre-computation
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW beta_coefficient AS SELECT SUM((pmr.portfolio_return - pm.portfolio_average) * (pmr.market_return - mm.market_average)) / SUM(POWER(pmr.market_return - mm.market_average, 2)) AS value FROM portfolio_market_returns pmr CROSS JOIN (SELECT AVG(portfolio_return) as portfolio_average FROM portfolio_market_returns) pm CROSS JOIN (SELECT AVG(market_return) as market_average FROM portfolio_market_returns) mm GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON beta_coefficient TO reporting_users;

