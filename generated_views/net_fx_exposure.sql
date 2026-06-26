-- =============================================
-- Metric: net_fx_exposure
-- DirectQuery Compatibility: High - Net FX exposure
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW net_fx_exposure AS SELECT fa.fx_assets - fl.fx_liabilities AS value FROM fx_assets fa JOIN fx_liabilities fl ON fa.entity_id = fl.entity_id AND fa.as_of_date = fl.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON net_fx_exposure TO reporting_users;

