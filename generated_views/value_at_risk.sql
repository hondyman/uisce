-- =============================================
-- Metric: value_at_risk
-- DirectQuery Compatibility: Medium - Statistical functions may need optimization
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW value_at_risk AS SELECT STDDEV_POP(pr.daily_return) * SQRT(hp.holding_days) * cf.confidence_multiplier AS value FROM portfolio_returns pr CROSS JOIN (SELECT holding_days FROM holding_periods LIMIT 1) hp CROSS JOIN (SELECT confidence_multiplier FROM confidence_factors LIMIT 1) cf GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON value_at_risk TO reporting_users;

