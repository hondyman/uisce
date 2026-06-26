-- =============================================
-- Metric: oci_recycling_on_disposal
-- DirectQuery Compatibility: High - OCI recycling
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW oci_recycling_on_disposal AS SELECT LEAST(aocb.accumulated_oci_balance, rgld.value) AS value FROM accumulated_oci_balance aocb JOIN realised_gain_loss_disposal rgld ON aocb.entity_id = rgld.entity_id AND aocb.as_of_date = rgld.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON oci_recycling_on_disposal TO reporting_users;

