-- =============================================
-- Metric: hedging_effectiveness
-- DirectQuery Compatibility: High - Hedging effectiveness
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW hedging_effectiveness AS SELECT hpl.hedge_profit_loss / epl.exposure_profit_loss AS value FROM hedge_profit_loss hpl JOIN exposure_profit_loss epl ON hpl.entity_id = epl.entity_id AND hpl.as_of_date = epl.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON hedging_effectiveness TO reporting_users;

