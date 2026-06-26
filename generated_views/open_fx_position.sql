-- =============================================
-- Metric: open_fx_position
-- DirectQuery Compatibility: High - Open FX position
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW open_fx_position AS SELECT nfe.value - ha.hedged_amount AS value FROM net_fx_exposure nfe JOIN hedged_amount ha ON nfe.entity_id = ha.entity_id AND nfe.as_of_date = ha.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON open_fx_position TO reporting_users;

