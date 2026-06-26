-- =============================================
-- Metric: cta_balance
-- DirectQuery Compatibility: High - CTA balance
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW cta_balance AS SELECT COALESCE(ctap.cta_prior, 0) + ctam.cta_movement AS value FROM cta_prior ctap JOIN cta_movement ctam ON ctap.entity_id = ctam.entity_id AND ctap.as_of_date = ctam.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON cta_balance TO reporting_users;

