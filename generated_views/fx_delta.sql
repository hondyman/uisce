-- =============================================
-- Metric: fx_delta
-- DirectQuery Compatibility: High - FX delta
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW fx_delta AS SELECT pv.portfolio_value * 0.01 AS value FROM portfolio_value pv;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON fx_delta TO reporting_users;

