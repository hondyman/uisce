-- =============================================
-- Metric: closing_rate_translation
-- DirectQuery Compatibility: High - Closing rate translation
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW closing_rate_translation AS SELECT bfc.balance_foreign_currency * frc.fx_rate_closing AS value FROM balance_foreign_currency bfc JOIN fx_rate_closing frc ON bfc.entity_id = frc.entity_id AND bfc.as_of_date = frc.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON closing_rate_translation TO reporting_users;

