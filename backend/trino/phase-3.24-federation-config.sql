# Trino Federation Configuration for Phase 3.24
# Global federated catalog that unifies all region-scoped data sources

# ============================================================================
# FILE: /etc/trino/catalog/federation.properties
# ============================================================================
# Trino Federated Catalog Configuration
# Deploy to: Global Trino cluster and all region-scoped Trino instances

connector.name=postgres
connection-url=jdbc:postgresql://{{ CONTROL_PLANE_DB_HOST }}:5432/semlayer
connection-user={{ DB_USER }}
connection-password={{ DB_PASSWORD }}

# Connection pool settings
connection-pool.max-size=100
connection-pool.min-idle=10
connection-pool.connection-timeout=30s

# Enable metadata caching
metadata-cache-ttl=10m
metadata-cache-refresh-interval=5m

# Hide internal schema
hide-internal-schemas=true

# ============================================================================
# REGION-SPECIFIC CATALOGS
# ============================================================================

# File: /etc/trino/catalog/iceberg_us_east.properties
connector.name=iceberg
iceberg.catalog.type=hive_metastore
hive.metastore.uri=thrift://hive-metastore-us-east.semlayer.internal:9083
warehouse=s3://semlayer-us-east/warehouse/
s3.endpoint=https://s3.us-east-1.amazonaws.com
s3.aws-access-key-id={{ AWS_ACCESS_KEY_US_EAST }}
s3.aws-secret-access-key={{ AWS_SECRET_KEY_US_EAST }}
s3.region=us-east-1

# File: /etc/trino/catalog/iceberg_eu_west.properties
connector.name=iceberg
iceberg.catalog.type=hive_metastore
hive.metastore.uri=thrift://hive-metastore-eu-west.semlayer.internal:9083
warehouse=s3://semlayer-eu-west/warehouse/
s3.endpoint=https://s3.eu-west-1.amazonaws.com
s3.aws-access-key-id={{ AWS_ACCESS_KEY_EU_WEST }}
s3.aws-secret-access-key={{ AWS_SECRET_KEY_EU_WEST }}
s3.region=eu-west-1

# File: /etc/trino/catalog/iceberg_apac.properties
connector.name=iceberg
iceberg.catalog.type=hive_metastore
hive.metastore.uri=thrift://hive-metastore-apac.semlayer.internal:9083
warehouse=s3://semlayer-apac/warehouse/
s3.endpoint=https://s3.ap-southeast-1.amazonaws.com
s3.aws-access-key-id={{ AWS_ACCESS_KEY_APAC }}
s3.aws-secret-access-key={{ AWS_SECRET_KEY_APAC }}
s3.region=ap-southeast-1

# ============================================================================
# SQL VIEWS FOR FEDERATION (Create in federation catalog)
# ============================================================================

-- Global Feature Drift View (Union across all regions)
CREATE OR REPLACE VIEW federation.global_feature_drift AS
SELECT 
    feature_id,
    ts,
    method,
    score,
    p_value,
    baseline_window,
    eval_window,
    'us-east' AS region
FROM iceberg_us_east.semlayer.feature_drift_metrics_us_east
UNION ALL
SELECT 
    feature_id,
    ts,
    method,
    score,
    p_value,
    baseline_window,
    eval_window,
    'eu-west' AS region
FROM iceberg_eu_west.semlayer.feature_drift_metrics_eu_west
UNION ALL
SELECT 
    feature_id,
    ts,
    method,
    score,
    p_value,
    baseline_window,
    eval_window,
    'apac' AS region
FROM iceberg_apac.semlayer.feature_drift_metrics_apac;

-- Global Feature Importance View
CREATE OR REPLACE VIEW federation.global_feature_importance AS
SELECT 
    feature_id,
    model_id,
    ts,
    method,
    importance,
    stability,
    trend,
    rank_position,
    'us-east' AS region
FROM iceberg_us_east.semlayer.feature_importance_us_east
UNION ALL
SELECT 
    feature_id,
    model_id,
    ts,
    method,
    importance,
    stability,
    trend,
    rank_position,
    'eu-west' AS region
FROM iceberg_eu_west.semlayer.feature_importance_eu_west
UNION ALL
SELECT 
    feature_id,
    model_id,
    ts,
    method,
    importance,
    stability,
    trend,
    rank_position,
    'apac' AS region
FROM iceberg_apac.semlayer.feature_importance_apac;

-- Global Time-Series Features View
CREATE OR REPLACE VIEW federation.global_ts_features AS
SELECT 
    feature_id,
    ts,
    horizon,
    forecast_value,
    lower_bound,
    upper_bound,
    anomaly,
    anomaly_score,
    trend,
    acf_lag1,
    pacf_lag1,
    'us-east' AS region
FROM iceberg_us_east.semlayer.feature_ts_features_us_east
UNION ALL
SELECT 
    feature_id,
    ts,
    horizon,
    forecast_value,
    lower_bound,
    upper_bound,
    anomaly,
    anomaly_score,
    trend,
    acf_lag1,
    pacf_lag1,
    'eu-west' AS region
FROM iceberg_eu_west.semlayer.feature_ts_features_eu_west
UNION ALL
SELECT 
    feature_id,
    ts,
    horizon,
    forecast_value,
    lower_bound,
    upper_bound,
    anomaly,
    anomaly_score,
    trend,
    acf_lag1,
    pacf_lag1,
    'apac' AS region
FROM iceberg_apac.semlayer.feature_ts_features_apac;

-- Global Discovery Candidates View
CREATE OR REPLACE VIEW federation.global_feature_discovery AS
SELECT 
    candidate_id,
    feature_name,
    source_database,
    source_field,
    data_type,
    completeness,
    cardinality,
    business_value,
    technical_score,
    discovery_method,
    status,
    properties,
    'us-east' AS region
FROM iceberg_us_east.semlayer.feature_discovery_us_east
UNION ALL
SELECT 
    candidate_id,
    feature_name,
    source_database,
    source_field,
    data_type,
    completeness,
    cardinality,
    business_value,
    technical_score,
    discovery_method,
    status,
    properties,
    'eu-west' AS region
FROM iceberg_eu_west.semlayer.feature_discovery_eu_west
UNION ALL
SELECT 
    candidate_id,
    feature_name,
    source_database,
    source_field,
    data_type,
    completeness,
    cardinality,
    business_value,
    technical_score,
    discovery_method,
    status,
    properties,
    'apac' AS region
FROM iceberg_apac.semlayer.feature_discovery_apac;

-- Global Feature Freshness View (Latency Dashboard)
CREATE OR REPLACE VIEW federation.global_feature_freshness AS
SELECT 
    f.feature_id,
    f.feature_name,
    r.region_code,
    fs.last_materialized,
    CAST(EXTRACT(EPOCH FROM (now() - fs.last_materialized))/3600.0 AS DECIMAL(10,2)) AS hours_since_materialized,
    CASE 
        WHEN fs.last_materialized < now() - INTERVAL '24 hours' THEN 'stale'
        WHEN fs.last_materialized < now() - INTERVAL '6 hours' THEN 'aging'
        ELSE 'fresh'
    END AS freshness_status
FROM semlayer.global_feature_catalog f
CROSS JOIN (
    SELECT 'us-east' AS region_code
    UNION ALL
    SELECT 'eu-west'
    UNION ALL
    SELECT 'apac'
) r
LEFT JOIN semlayer.global_feature_status fs 
    ON f.feature_id = fs.feature_id AND r.region_code = fs.region_code
WHERE f.is_active = TRUE;

-- Global Drift Aggregation View
CREATE OR REPLACE VIEW federation.global_drift_aggregation AS
SELECT 
    gd.feature_id,
    COUNT(DISTINCT gd.region) AS regions_with_drift,
    MAX(gd.score) AS max_drift_score,
    AVG(gd.score) AS avg_drift_score,
    SUM(CASE WHEN gd.score > 0.05 THEN 1 ELSE 0 END) AS regions_exceeding_threshold,
    MAX(gd.ts) AS most_recent_check
FROM federation.global_feature_drift gd
WHERE gd.ts > now() - INTERVAL '24 hours'
GROUP BY gd.feature_id;

-- Global Feature Ranking View (Importance aggregation)
CREATE OR REPLACE VIEW federation.global_feature_ranking AS
SELECT 
    fi.feature_id,
    AVG(fi.importance) AS avg_importance,
    MAX(fi.importance) AS max_importance,
    STDDEV(fi.importance) AS importance_stddev,
    COUNT(DISTINCT fi.region) AS regions_ranked,
    MAX(fi.ts) AS most_recent_rank
FROM federation.global_feature_importance fi
WHERE fi.ts > now() - INTERVAL '7 days'
GROUP BY fi.feature_id
ORDER BY avg_importance DESC;

-- Region Comparison View
CREATE OR REPLACE VIEW federation.region_comparison AS
SELECT 
    f.feature_id,
    f.feature_name,
    MAX(CASE WHEN fs.region_code = 'us-east' THEN fs.last_drift_score END) AS drift_us_east,
    MAX(CASE WHEN fs.region_code = 'eu-west' THEN fs.last_drift_score END) AS drift_eu_west,
    MAX(CASE WHEN fs.region_code = 'apac' THEN fs.last_drift_score END) AS drift_apac,
    MAX(CASE WHEN fs.region_code = 'us-east' THEN fs.last_importance_score END) AS importance_us_east,
    MAX(CASE WHEN fs.region_code = 'eu-west' THEN fs.last_importance_score END) AS importance_eu_west,
    MAX(CASE WHEN fs.region_code = 'apac' THEN fs.last_importance_score END) AS importance_apac,
    MAX(CASE WHEN fs.region_code = 'us-east' THEN fs.status END) AS status_us_east,
    MAX(CASE WHEN fs.region_code = 'eu-west' THEN fs.status END) AS status_eu_west,
    MAX(CASE WHEN fs.region_code = 'apac' THEN fs.status END) AS status_apac
FROM semlayer.global_feature_catalog f
LEFT JOIN semlayer.global_feature_status fs ON f.feature_id = fs.feature_id
WHERE f.is_active = TRUE
GROUP BY f.feature_id, f.feature_name;

-- ============================================================================
# Trino Query Examples (for global queries)
# ============================================================================

-- Example 1: Find all drifted features across regions
SELECT feature_id, region, score, ts
FROM federation.global_feature_drift
WHERE score > 0.05
  AND ts > now() - INTERVAL '24 hours'
ORDER BY score DESC;

-- Example 2: Top 20 features by average importance (global)
SELECT feature_id, avg_importance, max_importance, regions_ranked
FROM federation.global_feature_ranking
LIMIT 20;

-- Example 3: Compare feature drift by region
SELECT feature_id, drift_us_east, drift_eu_west, drift_apac
FROM federation.region_comparison
WHERE drift_us_east IS NOT NULL OR drift_eu_west IS NOT NULL OR drift_apac IS NOT NULL
ORDER BY COALESCE(drift_us_east, 0) + COALESCE(drift_eu_west, 0) + COALESCE(drift_apac, 0) DESC;

-- Example 4: Feature freshness across all regions
SELECT feature_id, feature_name, region_code, freshness_status, hours_since_materialized
FROM federation.global_feature_freshness
WHERE freshness_status != 'fresh'
ORDER BY hours_since_materialized DESC;

-- Example 5: Discovery candidates approved rate by region
SELECT 
    region,
    COUNT(*) AS total_candidates,
    SUM(CASE WHEN status = 'approved' THEN 1 ELSE 0 END) AS approved,
    ROUND(100.0 * SUM(CASE WHEN status = 'approved' THEN 1 ELSE 0 END) / COUNT(*), 1) AS approval_rate
FROM federation.global_feature_discovery
WHERE status IN ('approved', 'rejected')
GROUP BY region
ORDER BY approval_rate DESC;

-- Example 6: Time-series anomalies by region
SELECT 
    feature_id,
    region,
    SUM(CASE WHEN anomaly = TRUE THEN 1 ELSE 0 END) AS anomaly_count,
    MAX(anomaly_score) AS max_anomaly_score,
    COUNT(*) AS total_points
FROM federation.global_ts_features
WHERE ts > now() - INTERVAL '7 days'
GROUP BY feature_id, region
HAVING SUM(CASE WHEN anomaly = TRUE THEN 1 ELSE 0 END) > 0
ORDER BY max_anomaly_score DESC;

-- ============================================================================
# Trino Configuration File (config.properties)
# ============================================================================

discovery.uri=http://coordinator.semlayer.internal:8080

# Memory configuration
query.max-memory=100GB
query.max-memory-per-node=4GB
query.max-total-memory-per-node=5GB

# Timeout configuration
query.min-expire-age=30m
query.max-age=1h

# Federation query settings
optimizer.pushdown-subqueries-enabled=true
optimizer.pushdown-filter-into-scan=true
optimizer.optimize-plan-with-estimate-pushdown=true

# Catalog configuration
catalogmanager.client.timeout=30s

# Exchange configuration (for distributed queries)
exchange.compression-codec=SNAPPY
exchange.max-buffer-size=256MB

# Schedule configuration
scheduler.approximate-ordering-enabled=true
scheduler.min-schedule-split-batch-size=4

# Logging
log.output-file=/var/log/trino/query.log

# Performance tuning
optimization.join-reordering-strategy=COST_BASED
optimization.index-join-enabled=true
optimization.optimize-hash-generation=true

# ============================================================================
# Hive Metastore Configuration (per region)
# ============================================================================

# File: /etc/hive-metastore/hive-site-us-east.xml
<configuration>
    <property>
        <name>hive.metastore.warehouse.dir</name>
        <value>s3://semlayer-us-east/warehouse/</value>
    </property>
    <property>
        <name>javax.jdo.option.ConnectionDriverName</name>
        <value>org.postgresql.Driver</value>
    </property>
    <property>
        <name>javax.jdo.option.ConnectionURL</name>
        <value>jdbc:postgresql://postgres-us-east.semlayer.internal:5432/metastore</value>
    </property>
    <property>
        <name>hive.server2.thrift.bind.host</name>
        <value>0.0.0.0</value>
    </property>
    <property>
        <name>hive.server2.thrift.port</name>
        <value>10000</value>
    </property>
    <property>
        <name>hive.exec.max.dynamic.partitions</name>
        <value>2000</value>
    </property>
</configuration>

# ============================================================================
# Query Federation Test Queries
# ============================================================================

-- Verify federation is working
SELECT * FROM federation.global_feature_drift LIMIT 5;

-- Check federation view performance
EXPLAIN (FORMAT JSON)
SELECT feature_id, region, MAX(score) as max_drift
FROM federation.global_feature_drift
WHERE ts > now() - INTERVAL '24 hours'
GROUP BY feature_id, region;

-- Test cross-region join
SELECT 
    r.feature_id,
    r.avg_importance,
    d.regions_with_drift,
    d.max_drift_score
FROM federation.global_feature_ranking r
LEFT JOIN federation.global_drift_aggregation d ON r.feature_id = d.feature_id
WHERE r.avg_importance > 0.7
  AND d.regions_with_drift > 1;

-- Performance test: large federation query
SELECT 
    feature_id,
    COUNT(DISTINCT region) as region_count,
    COUNT(*) as total_records
FROM federation.global_ts_features
WHERE ts > now() - INTERVAL '30 days'
GROUP BY feature_id
HAVING COUNT(DISTINCT region) = 3;
