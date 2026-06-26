-- =============================================
-- Cross-Engine Validation Script
-- Ensures metric consistency across all engines
-- =============================================

-- This script should be run against each engine
-- Results should be compared for consistency

-- Configuration
\set ENGINE_NAME 'sqlserver'  -- Change per engine: sqlserver, oracle, postgres, snowflake, iceberg
\set TEST_ENTITY_ID 'TEST_ENTITY_001'
\set TEST_DATE '2024-12-31'

-- =============================================
-- Metric Validation Tests
-- =============================================

-- Test 1: Net Interest Margin
WITH test_results AS (
    SELECT
        :'ENGINE_NAME' as engine,
        entity_id,
        as_of_date,
        value as net_interest_margin,
        CASE
            WHEN value IS NULL THEN 'NULL_VALUE'
            WHEN value < -100 OR value > 100 THEN 'OUTLIER'
            ELSE 'VALID'
        END as validation_status
    FROM net_interest_margin
    WHERE entity_id = :'TEST_ENTITY_ID'
      AND as_of_date = :'TEST_DATE'
)
SELECT * FROM test_results;

-- Test 2: Sharpe Ratio
WITH test_results AS (
    SELECT
        :'ENGINE_NAME' as engine,
        entity_id,
        as_of_date,
        value as sharpe_ratio,
        CASE
            WHEN value IS NULL THEN 'NULL_VALUE'
            WHEN ABS(value) > 10 THEN 'EXTREME_VALUE'
            ELSE 'VALID'
        END as validation_status
    FROM sharpe_ratio
    WHERE entity_id = :'TEST_ENTITY_ID'
      AND as_of_date = :'TEST_DATE'
)
SELECT * FROM test_results;

-- Test 3: Value at Risk
WITH test_results AS (
    SELECT
        :'ENGINE_NAME' as engine,
        entity_id,
        as_of_date,
        value as value_at_risk,
        CASE
            WHEN value IS NULL THEN 'NULL_VALUE'
            WHEN value > 0 THEN 'POSITIVE_VaR'
            WHEN value < -100 THEN 'EXTREME_LOSS'
            ELSE 'VALID'
        END as validation_status
    FROM value_at_risk
    WHERE entity_id = :'TEST_ENTITY_ID'
      AND as_of_date = :'TEST_DATE'
)
SELECT * FROM test_results;

-- =============================================
-- Performance Benchmarking
-- =============================================

-- Query execution time measurement
\timing on

-- Benchmark Net Interest Margin
SELECT COUNT(*) as record_count,
       AVG(value) as avg_value,
       STDDEV(value) as stddev_value
FROM net_interest_margin
WHERE as_of_date >= '2024-01-01';

-- Benchmark Sharpe Ratio with time intelligence
SELECT entity_id,
       AVG(CASE WHEN as_of_date >= '2024-01-01' THEN value END) as ytd_avg,
       COUNT(*) as total_records
FROM sharpe_ratio
GROUP BY entity_id;

\timing off

-- =============================================
-- Data Quality Checks
-- =============================================

-- Check for null values in critical metrics
SELECT
    'net_interest_margin' as metric,
    COUNT(*) as total_records,
    COUNT(CASE WHEN value IS NULL THEN 1 END) as null_values,
    ROUND(100.0 * COUNT(CASE WHEN value IS NULL THEN 1 END) / COUNT(*), 2) as null_percentage
FROM net_interest_margin

UNION ALL

SELECT
    'sharpe_ratio' as metric,
    COUNT(*) as total_records,
    COUNT(CASE WHEN value IS NULL THEN 1 END) as null_values,
    ROUND(100.0 * COUNT(CASE WHEN value IS NULL THEN 1 END) / COUNT(*), 2) as null_percentage
FROM sharpe_ratio

UNION ALL

SELECT
    'value_at_risk' as metric,
    COUNT(*) as total_records,
    COUNT(CASE WHEN value IS NULL THEN 1 END) as null_values,
    ROUND(100.0 * COUNT(CASE WHEN value IS NULL THEN 1 END) / COUNT(*), 2) as null_percentage
FROM value_at_risk;

-- =============================================
-- Cross-Engine Comparison Template
-- =============================================

-- Export results to compare across engines:
/*
COPY (
    SELECT
        'net_interest_margin' as metric,
        :'ENGINE_NAME' as engine,
        entity_id,
        as_of_date,
        ROUND(value::numeric, 6) as value
    FROM net_interest_margin
    WHERE entity_id = :'TEST_ENTITY_ID'
    ORDER BY as_of_date DESC
    LIMIT 10
) TO '/tmp/net_interest_margin_:ENGINE_NAME.csv' WITH CSV HEADER;
*/

