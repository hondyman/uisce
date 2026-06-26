-- =============================================
-- Metric: effective_interest_income
-- Category: income_recognition
-- Governance: golden
-- Engine: iceberg
-- Generated on: Sat Sep 13 17:25:55 EDT 2025
-- =============================================

-- View Definition
CREATE TABLE effective_interest_income USING iceberg PARTITIONED BY (entity_id) AS SELECT oca.entity_id, oca.as_of_date, oca.opening_carrying_amount * eir.effective_interest_rate AS value FROM opening_carrying_amounts oca JOIN effective_interest_rates eir ON oca.entity_id = eir.entity_id AND oca.as_of_date = eir.as_of_date;

-- Preaggregation Strategy
ALTER TABLE effective_interest_income ADD PARTITION FIELD entity_id;

-- Performance Notes: Partition by entity for tenant-based access patterns

-- Grant permissions (customize as needed)
-- GRANT SELECT ON effective_interest_income TO reporting_users;

