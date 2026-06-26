-- =============================================
-- Metric: effective_interest_income
-- Category: income_recognition
-- Governance: golden
-- Engine: sqlserver
-- Generated on: Sat Sep 13 17:25:56 EDT 2025
-- =============================================

-- View Definition
CREATE VIEW effective_interest_income AS SELECT oca.opening_carrying_amount * eir.effective_interest_rate AS value FROM opening_carrying_amounts oca JOIN effective_interest_rates eir ON oca.entity_id = eir.entity_id AND oca.as_of_date = eir.as_of_date;

-- Preaggregation Strategy
CREATE CLUSTERED INDEX idx_eii ON effective_interest_income(entity_id, as_of_date);

-- Performance Notes: Clustered index on primary key columns

-- Grant permissions (customize as needed)
-- GRANT SELECT ON effective_interest_income TO reporting_users;

