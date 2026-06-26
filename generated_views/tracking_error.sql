-- =============================================
-- Metric: tracking_error
-- DirectQuery Compatibility: Medium - Standard deviation
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW tracking_error AS SELECT STDDEV_POP(ar.active_return) AS value FROM active_returns ar GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON tracking_error TO reporting_users;

