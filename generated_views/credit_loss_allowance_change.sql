-- =============================================
-- Metric: credit_loss_allowance_change
-- DirectQuery Compatibility: High - ECL change
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW credit_loss_allowance_change AS SELECT eclc.ecl_closing - ecli.ecl_opening AS value FROM ecl_closing eclc JOIN ecl_opening ecli ON eclc.entity_id = ecli.entity_id AND eclc.as_of_date = ecli.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON credit_loss_allowance_change TO reporting_users;

