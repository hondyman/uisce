-- =============================================
-- Metric: fx_translation_reserve_oci
-- DirectQuery Compatibility: High - CTA reserve
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW fx_translation_reserve_oci AS SELECT COALESCE(ctap.cta_prior, 0) + ctam.cta_movement AS value FROM cta_prior ctap JOIN cta_movement ctam ON ctap.entity_id = ctam.entity_id AND ctap.as_of_date = ctam.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON fx_translation_reserve_oci TO reporting_users;

