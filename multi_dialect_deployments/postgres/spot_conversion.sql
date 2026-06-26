-- =============================================
-- Metric: spot_conversion
-- Category: conversion
-- Governance: golden
-- Engine: postgres
-- Generated on: Sat Sep 13 17:25:55 EDT 2025
-- =============================================

-- View Definition
CREATE VIEW spot_conversion AS SELECT (asc.amount_source_currency * sfr.spot_fx_rate)::DECIMAL(20,4) AS value FROM amount_source_currency asc JOIN spot_fx_rate sfr ON asc.entity_id = sfr.entity_id AND asc.as_of_date = sfr.as_of_date;

-- Preaggregation Strategy
CREATE MATERIALIZED VIEW mv_spot AS SELECT * FROM spot_conversion WITH DATA; CREATE UNIQUE INDEX idx_spot_unique ON mv_spot(entity_id, as_of_date);

-- Performance Notes: Unique constraints for data integrity

-- Grant permissions (customize as needed)
-- GRANT SELECT ON spot_conversion TO reporting_users;

