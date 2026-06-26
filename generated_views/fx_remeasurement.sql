-- =============================================
-- Metric: fx_remeasurement
-- DirectQuery Compatibility: High - FX remeasurement
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW fx_remeasurement AS SELECT (fxc.fx_rate_closing - fxa.fx_rate_avg_period) * mbf.monetary_balance_foreign AS value FROM fx_rate_closing fxc JOIN fx_rate_avg_period fxa ON fxc.entity_id = fxa.entity_id AND fxc.as_of_date = fxa.as_of_date JOIN monetary_balance_foreign mbf ON fxc.entity_id = mbf.entity_id AND fxc.as_of_date = mbf.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON fx_remeasurement TO reporting_users;

