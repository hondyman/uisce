-- =============================================
-- Metric: large_exposure_limit
-- DirectQuery Compatibility: High - Large exposure limit
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW large_exposure_limit AS SELECT MAX(ce.amount) / SUM(ec.amount) AS value FROM counterparty_exposures ce JOIN eligible_capital ec ON ce.entity_id = ec.entity_id AND ce.as_of_date = ec.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON large_exposure_limit TO reporting_users;

