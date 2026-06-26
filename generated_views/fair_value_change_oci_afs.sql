-- =============================================
-- Metric: fair_value_change_oci_afs
-- DirectQuery Compatibility: High - FV change OCI
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW fair_value_change_oci_afs AS SELECT fvc.fair_value_current - acc.value AS value FROM fair_value_current fvc JOIN carrying_amount_rollforward acc ON fvc.entity_id = acc.entity_id AND fvc.as_of_date = acc.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON fair_value_change_oci_afs TO reporting_users;

