-- =============================================
-- Metric: equity_method_share_of_profit
-- DirectQuery Compatibility: High - Equity method profit
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW equity_method_share_of_profit AS SELECT ini.investee_net_income * op.ownership_pct AS value FROM investee_net_income ini JOIN ownership_pct op ON ini.entity_id = op.entity_id AND ini.as_of_date = op.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON equity_method_share_of_profit TO reporting_users;

