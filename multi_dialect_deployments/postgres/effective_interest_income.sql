-- =============================================
-- Metric: effective_interest_income
-- Category: income_recognition
-- Governance: golden
-- Engine: postgres
-- Generated on: Sat Sep 13 17:25:55 EDT 2025
-- =============================================

-- View Definition
CREATE VIEW effective_interest_income AS SELECT (oca.opening_carrying_amount * eir.effective_interest_rate)::DECIMAL(20,4) AS value FROM opening_carrying_amounts oca JOIN effective_interest_rates eir ON oca.entity_id = eir.entity_id AND oca.as_of_date = eir.as_of_date;

-- Preaggregation Strategy
CREATE MATERIALIZED VIEW mv_eii AS SELECT * FROM effective_interest_income; CREATE INDEX idx_eii_brin ON mv_eii USING BRIN (as_of_date);

-- Performance Notes: BRIN indexes for time-series data

-- Grant permissions (customize as needed)
-- GRANT SELECT ON effective_interest_income TO reporting_users;

