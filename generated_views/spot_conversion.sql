-- =============================================
-- Metric: spot_conversion
-- DirectQuery Compatibility: High - Spot conversion
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW spot_conversion AS SELECT asc.amount_source_currency * sfr.spot_fx_rate AS value FROM amount_source_currency asc JOIN spot_fx_rate sfr ON asc.entity_id = sfr.entity_id AND asc.as_of_date = sfr.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON spot_conversion TO reporting_users;

