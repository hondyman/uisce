-- =============================================
-- Metric: liquidity_coverage_ratio
-- DirectQuery Compatibility: High - LCR calculation
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW liquidity_coverage_ratio AS SELECT SUM(hqla.amount) / SUM(nco.amount) AS value FROM high_quality_liquid_assets hqla JOIN net_cash_outflows nco ON hqla.entity_id = nco.entity_id AND hqla.as_of_date = nco.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON liquidity_coverage_ratio TO reporting_users;

