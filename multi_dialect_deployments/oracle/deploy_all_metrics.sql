-- =============================================
-- Master Deployment Script for oracle
-- Generated from Multi-Dialect Metric Mappings
-- Generated on: Sat Sep 13 17:25:55 EDT 2025
-- =============================================

-- Prerequisites:
-- 1. Create semantic layer schema
-- 2. Deploy base tables
-- 3. Set up permissions


-- Deploy net_interest_margin (golden - profitability)
\i net_interest_margin.sql


-- Deploy sharpe_ratio (golden - risk_adjusted_performance)
\i sharpe_ratio.sql


-- Deploy value_at_risk (golden - market_risk)
\i value_at_risk.sql


-- Deploy beta_coefficient (golden - market_risk)
\i beta_coefficient.sql


-- Deploy effective_interest_income (golden - income_recognition)
\i effective_interest_income.sql


-- Deploy spot_conversion (golden - conversion)
\i spot_conversion.sql


-- =============================================
-- Validation Queries
-- =============================================

-- Check deployed views
SELECT table_name, table_type
FROM information_schema.tables
WHERE table_schema = 'semantic_layer'
  AND table_type = 'VIEW'
ORDER BY table_name;

-- Sample validation query (customize per metric)
-- SELECT entity_id, as_of_date, value
-- FROM net_interest_margin
-- WHERE entity_id = 'SAMPLE_ENTITY'
-- ORDER BY as_of_date DESC
-- LIMIT 10;

-- =============================================
-- Performance Monitoring Setup
-- =============================================

-- Create monitoring table (if not exists)
CREATE TABLE IF NOT EXISTS metric_performance_log (
    metric_name VARCHAR(100),
    execution_time TIMESTAMP,
    query_duration INTERVAL,
    rows_returned INTEGER,
    engine_version VARCHAR(50)
);

-- Sample monitoring query
-- INSERT INTO metric_performance_log
-- SELECT
--     'net_interest_margin',
--     CURRENT_TIMESTAMP,
--     pg_last_query_duration(),
--     (SELECT COUNT(*) FROM net_interest_margin),
--     version()
-- FROM net_interest_margin
-- LIMIT 1;

