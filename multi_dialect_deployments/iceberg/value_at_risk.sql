-- =============================================
-- Metric: value_at_risk
-- Category: market_risk
-- Governance: golden
-- Engine: iceberg
-- Generated on: Sat Sep 13 17:25:55 EDT 2025
-- =============================================

-- View Definition
CREATE TABLE value_at_risk USING iceberg TBLPROPERTIES ('write.update.mode'='merge-on-read') AS SELECT pr.entity_id, pr.as_of_date, STDDEV(pr.daily_return) * SQRT(hp.holding_days) * cf.confidence_multiplier AS value FROM portfolio_returns pr CROSS JOIN (SELECT holding_days FROM holding_periods LIMIT 1) hp CROSS JOIN (SELECT confidence_multiplier FROM confidence_factors LIMIT 1) cf GROUP BY pr.entity_id, pr.as_of_date;

-- Preaggregation Strategy
ALTER TABLE value_at_risk SET TBLPROPERTIES ('write.merge.mode'='merge-on-read');

-- Performance Notes: Merge-on-read for frequent updates

-- Grant permissions (customize as needed)
-- GRANT SELECT ON value_at_risk TO reporting_users;

