-- =============================================
-- Metric: conversion_rate
-- DirectQuery Compatibility: High - Conversion rate
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW conversion_rate AS SELECT SUM(sp.count) / SUM(st.count) AS value FROM store_purchases sp JOIN store_traffic st ON sp.entity_id = st.entity_id AND sp.as_of_date = st.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON conversion_rate TO reporting_users;

