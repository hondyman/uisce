-- =============================================
-- Metric: equity_method_carrying_value
-- DirectQuery Compatibility: Medium - Complex equity method calculation
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW equity_method_carrying_value AS SELECT ocv.opening_carrying_value + emsop.value - dr.dividends_received + oca.oci_adjustment + ftr.value AS value FROM opening_carrying_value ocv JOIN equity_method_share_of_profit emsop ON ocv.entity_id = emsop.entity_id AND ocv.as_of_date = emsop.as_of_date JOIN dividends_received dr ON ocv.entity_id = dr.entity_id AND ocv.as_of_date = dr.as_of_date JOIN oci_adjustment oca ON ocv.entity_id = oca.entity_id AND ocv.as_of_date = oca.as_of_date JOIN fx_translation_reserve_oci ftr ON ocv.entity_id = ftr.entity_id AND ocv.as_of_date = ftr.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON equity_method_carrying_value TO reporting_users;

