-- =============================================
-- Metric: spot_conversion
-- Category: conversion
-- Governance: golden
-- Engine: snowflake
-- Generated on: Sat Sep 13 17:25:56 EDT 2025
-- =============================================

-- View Definition
CREATE VIEW spot_conversion AS SELECT asc.amount_source_currency * sfr.spot_fx_rate AS value FROM amount_source_currency asc JOIN spot_fx_rate sfr ON asc.entity_id = sfr.entity_id AND asc.as_of_date = sfr.as_of_date;

-- Preaggregation Strategy
CREATE DYNAMIC TABLE dt_spot TARGET_LAG = '15 minutes' AS SELECT * FROM spot_conversion;

-- Performance Notes: Low latency for FX trading applications

-- Grant permissions (customize as needed)
-- GRANT SELECT ON spot_conversion TO reporting_users;

