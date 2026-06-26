-- =============================================
-- Metric: realised_fx_gain_loss
-- DirectQuery Compatibility: High - Realised FX G/L
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW realised_fx_gain_loss AS SELECT sab.settlement_amount_base - cab.contracted_amount_base AS value FROM settlement_amount_base sab JOIN contracted_amount_base cab ON sab.entity_id = cab.entity_id AND sab.as_of_date = cab.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON realised_fx_gain_loss TO reporting_users;

