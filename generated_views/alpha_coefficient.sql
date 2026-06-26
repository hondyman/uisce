-- =============================================
-- Metric: alpha_coefficient
-- DirectQuery Compatibility: Medium - Alpha calculation
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW alpha_coefficient AS SELECT AVG(pr.daily_return) - (rfr.risk_free_rate + b.beta_coefficient * (AVG(mr.daily_return) - rfr.risk_free_rate)) AS value FROM portfolio_returns pr CROSS JOIN (SELECT risk_free_rate FROM risk_free_rates LIMIT 1) rfr CROSS JOIN (SELECT beta_coefficient FROM beta_coefficients LIMIT 1) b CROSS JOIN market_returns mr GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON alpha_coefficient TO reporting_users;

