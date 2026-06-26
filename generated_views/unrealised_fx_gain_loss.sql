-- =============================================
-- Metric: unrealised_fx_gain_loss
-- DirectQuery Compatibility: High - Unrealised FX G/L
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW unrealised_fx_gain_loss AS SELECT (cfr.current_fx_rate - pfr.prior_fx_rate) * opf.open_position_foreign AS value FROM current_fx_rate cfr JOIN prior_fx_rate pfr ON cfr.entity_id = pfr.entity_id AND cfr.as_of_date = pfr.as_of_date JOIN open_position_foreign opf ON cfr.entity_id = opf.entity_id AND cfr.as_of_date = opf.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON unrealised_fx_gain_loss TO reporting_users;

