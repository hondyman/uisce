-- =============================================
-- Metric: spot_conversion
-- Category: conversion
-- Governance: golden
-- Engine: iceberg
-- Generated on: Sat Sep 13 17:25:55 EDT 2025
-- =============================================

-- View Definition
CREATE TABLE spot_conversion USING iceberg TBLPROPERTIES ('write.update.mode'='copy-on-write') AS SELECT asc.entity_id, asc.as_of_date, asc.amount_source_currency * sfr.spot_fx_rate AS value FROM amount_source_currency asc JOIN spot_fx_rate sfr ON asc.entity_id = sfr.entity_id AND asc.as_of_date = sfr.as_of_date;

-- Preaggregation Strategy
ALTER TABLE spot_conversion SET TBLPROPERTIES ('write.update.mode'='copy-on-write');

-- Performance Notes: Copy-on-write for immutable FX rates

-- Grant permissions (customize as needed)
-- GRANT SELECT ON spot_conversion TO reporting_users;

