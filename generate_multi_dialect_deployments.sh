#!/bin/bash

# Multi-Dialect SQL Deployment Generator
# Generates engine-specific SQL deployment scripts from unified mappings

echo "🚀 Multi-Dialect SQL Deployment Generator"
echo "=========================================="

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MAPPING_FILE="$SCRIPT_DIR/multi_dialect_metric_mappings.json"
OUTPUT_DIR="$SCRIPT_DIR/multi_dialect_deployments"
mkdir -p "$OUTPUT_DIR"

# Check for jq
if ! command -v jq &> /dev/null; then
    echo "❌ jq is required. Installing..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        brew install jq
    else
        echo "Please install jq manually for your system"
        exit 1
    fi
fi

echo "📁 Output directory: $OUTPUT_DIR"

# Function to generate deployment scripts for each engine
generate_engine_scripts() {
    local mapping_file="$1"
    local output_dir="$2"

    # Get list of engines
    local engines
    engines=$(jq -r '.multi_dialect_metric_mappings.metrics[0].dialects | keys[]' "$mapping_file")

    for engine in $engines; do
        echo "🔧 Generating scripts for: $engine"

        local engine_dir="$output_dir/$engine"
        mkdir -p "$engine_dir"

        # Create master deployment script for this engine
        local master_script="$engine_dir/deploy_all_metrics.sql"

        cat > "$master_script" << EOF
-- =============================================
-- Master Deployment Script for $engine
-- Generated from Multi-Dialect Metric Mappings
-- Generated on: $(date)
-- =============================================

-- Prerequisites:
-- 1. Create semantic layer schema
-- 2. Deploy base tables
-- 3. Set up permissions

EOF

        # Process each metric
        jq -r --arg engine "$engine" '.multi_dialect_metric_mappings.metrics[] | @base64' "$mapping_file" | while read -r encoded; do
            # Decode the metric
            local metric
            metric=$(echo "$encoded" | base64 --decode)

            local node_id
            node_id=$(echo "$metric" | jq -r '.node_id')
            local category
            category=$(echo "$metric" | jq -r '.category')
            local governance
            governance=$(echo "$metric" | jq -r '.governance')

            echo "  📊 Processing metric: $node_id"

            # Get engine-specific implementation
            local view_definition
            view_definition=$(echo "$metric" | jq -r --arg engine "$engine" '.dialects[$engine].view_definition')
            local preaggregation
            preaggregation=$(echo "$metric" | jq -r --arg engine "$engine" '.dialects[$engine].preaggregation')
            local performance_notes
            performance_notes=$(echo "$metric" | jq -r --arg engine "$engine" '.dialects[$engine].performance_notes')

            # Create individual metric script
            local metric_script="$engine_dir/${node_id}.sql"

            cat > "$metric_script" << EOF
-- =============================================
-- Metric: $node_id
-- Category: $category
-- Governance: $governance
-- Engine: $engine
-- Generated on: $(date)
-- =============================================

-- View Definition
$view_definition

-- Preaggregation Strategy
$preaggregation

-- Performance Notes: $performance_notes

-- Grant permissions (customize as needed)
-- GRANT SELECT ON $node_id TO reporting_users;

EOF

            # Add to master script
            cat >> "$master_script" << EOF

-- Deploy $node_id ($governance - $category)
\\i ${node_id}.sql

EOF

        done

        # Add validation section to master script
        cat >> "$master_script" << EOF

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

EOF

        echo "  ✅ Created master script: $master_script"
    done
}

# Function to generate cross-engine validation script
generate_validation_script() {
    local mapping_file="$1"
    local output_dir="$2"

    local validation_script="$output_dir/cross_engine_validation.sql"

    cat > "$validation_script" << 'EOF'
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

EOF

    echo "✅ Created validation script: $validation_script"
}

# Function to generate deployment documentation
generate_documentation() {
    local mapping_file="$1"
    local output_dir="$2"

    local readme_file="$output_dir/README.md"

    cat > "$readme_file" << 'EOF'
# Multi-Dialect SQL Deployment Guide

## Overview

This directory contains engine-specific SQL deployment scripts for the unified financial services semantic layer. Each supported database engine (SQL Server, Oracle, PostgreSQL, Snowflake, Iceberg) has its own subdirectory with tailored implementations.

## Supported Engines

### SQL Server
- **Dialect**: T-SQL
- **Features**: Columnstore indexes, query store, in-memory OLTP
- **Best For**: Enterprise Windows environments, high-performance analytics

### Oracle
- **Dialect**: PL/SQL
- **Features**: Materialized views with query rewrite, advanced partitioning
- **Best For**: Large-scale enterprise deployments, complex transaction processing

### PostgreSQL
- **Dialect**: Standard SQL with extensions
- **Features**: Rich indexing options, concurrent materialized views
- **Best For**: Open-source environments, web applications, geospatial data

### Snowflake
- **Dialect**: Snowflake SQL
- **Features**: Automatic clustering, dynamic tables, multi-cluster warehouses
- **Best For**: Cloud-native analytics, pay-per-use pricing

### Iceberg (via Trino/Spark)
- **Dialect**: Trino SQL / Spark SQL
- **Features**: Schema evolution, time travel, ACID transactions
- **Best For**: Data lakehouse architectures, streaming analytics

## Directory Structure

```
multi_dialect_deployments/
├── sqlserver/
│   ├── deploy_all_metrics.sql
│   ├── net_interest_margin.sql
│   ├── sharpe_ratio.sql
│   └── ...
├── oracle/
│   ├── deploy_all_metrics.sql
│   ├── net_interest_margin.sql
│   └── ...
├── postgres/
│   ├── deploy_all_metrics.sql
│   └── ...
├── snowflake/
│   └── ...
├── iceberg/
│   └── ...
├── cross_engine_validation.sql
└── README.md
```

## Deployment Process

### Phase 1: Prerequisites
1. Create semantic layer schema in target engine
2. Deploy base tables and relationships
3. Set up appropriate permissions and roles

### Phase 2: Metric Deployment
```bash
# For SQL Server
cd sqlserver
sqlcmd -S your_server -d your_database -i deploy_all_metrics.sql

# For PostgreSQL
cd postgres
psql -d your_database -f deploy_all_metrics.sql

# For Oracle
cd oracle
sqlplus user/password@database @deploy_all_metrics.sql
```

### Phase 3: Validation
```bash
# Run cross-engine validation
psql -d your_database -f ../cross_engine_validation.sql
```

## Performance Optimization Strategies

### SQL Server
- Use columnstore indexes for analytical workloads
- Implement query store for performance monitoring
- Consider in-memory tables for frequently accessed data

### Oracle
- Leverage materialized view query rewrite
- Use partitioning for large datasets
- Implement result caching for static reference data

### PostgreSQL
- Use BRIN indexes for time-series data
- Implement partial indexes for filtered queries
- Consider pg_stat_statements for query analysis

### Snowflake
- Configure automatic clustering on commonly filtered columns
- Use dynamic tables for automated refresh
- Optimize warehouse sizing based on workload

### Iceberg
- Design partition strategies for query patterns
- Use Z-ordering for multi-dimensional queries
- Implement table maintenance for optimal file sizes

## Monitoring and Maintenance

### Health Checks
- Monitor query performance trends
- Track data freshness and refresh times
- Alert on metric calculation failures

### Optimization Opportunities
- Regularly review and update statistics
- Rebuild indexes based on usage patterns
- Archive historical data as needed

## Troubleshooting

### Common Issues

1. **Query Performance Degradation**
   - Check index usage and rebuild if necessary
   - Review query execution plans
   - Consider materialized view refresh strategies

2. **Data Consistency Issues**
   - Validate source data quality
   - Check for null handling edge cases
   - Review calculation logic for boundary conditions

3. **Deployment Failures**
   - Verify schema permissions
   - Check for circular dependencies
   - Validate engine-specific syntax

### Support Resources

- **SQL Server**: Query Store, Execution Plans, DMVs
- **Oracle**: AWR reports, ASH, SQL Monitor
- **PostgreSQL**: pg_stat_statements, EXPLAIN ANALYZE
- **Snowflake**: Query Profile, Query History
- **Iceberg**: Table metadata, query planning

## Best Practices

1. **Testing**: Always test in development environment first
2. **Backup**: Ensure proper backups before major deployments
3. **Monitoring**: Implement comprehensive monitoring from day one
4. **Documentation**: Keep deployment and configuration documentation current
5. **Version Control**: Track changes to metric definitions and deployment scripts

## Security Considerations

- Implement proper access controls
- Use parameterized queries to prevent SQL injection
- Encrypt sensitive data at rest and in transit
- Regularly audit access patterns and permissions

---

**Generated on**: $(date)
**Source**: Unified Financial Services Super Bundle
**Metrics**: 82+ financial metrics
**Engines**: 5 database platforms
EOF

    echo "✅ Created documentation: $readme_file"
}

# Generate all deployment artifacts
echo "🔧 Generating deployment scripts..."
generate_engine_scripts "$MAPPING_FILE" "$OUTPUT_DIR"

echo "🔍 Generating validation scripts..."
generate_validation_script "$MAPPING_FILE" "$OUTPUT_DIR"

echo "📚 Generating documentation..."
generate_documentation "$MAPPING_FILE" "$OUTPUT_DIR"

echo ""
echo "🎉 Multi-Dialect Deployment Generation Complete!"
echo "==============================================="
echo "📂 Generated files in: $OUTPUT_DIR"
echo ""
echo "📋 What's been created:"
echo "• Engine-specific deployment directories (sqlserver, oracle, postgres, snowflake, iceberg)"
echo "• Individual metric SQL files for each engine"
echo "• Master deployment scripts per engine"
echo "• Cross-engine validation framework"
echo "• Comprehensive documentation"
echo ""
echo "🚀 Next Steps:"
echo "1. Choose your target database engine(s)"
echo "2. Review the generated SQL files for your environment"
echo "3. Deploy base tables and relationships first"
echo "4. Run the master deployment script for your engine"
echo "5. Execute validation tests across engines"
echo "6. Set up monitoring and performance optimization"
echo ""
echo "💡 Pro Tips:"
echo "• Start with a single engine for initial testing"
echo "• Use the validation script to ensure consistency"
echo "• Monitor performance and adjust preaggregation strategies"
echo "• Keep deployment scripts in version control"
echo ""
echo "🔧 Customization Notes:"
echo "• Adjust table names and schemas as needed"
echo "• Modify preaggregation strategies based on data volumes"
echo "• Update permission grants for your security model"
echo "• Consider engine-specific optimizations for your workload"
