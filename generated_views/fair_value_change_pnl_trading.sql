-- =============================================
-- Metric: fair_value_change_pnl_trading
-- DirectQuery Compatibility: High - FV change P&L
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW fair_value_change_pnl_trading AS SELECT fvc.fair_value_current - fvp.fair_value_prior AS value FROM fair_value_current fvc JOIN fair_value_prior fvp ON fvc.entity_id = fvp.entity_id AND fvc.as_of_date = fvp.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON fair_value_change_pnl_trading TO reporting_users;

