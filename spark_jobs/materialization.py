"""
Phase 3.21: Spark Feature Materialization
Incremental materialization using watermarks and idempotent writes.
"""

from pyspark.sql import SparkSession, DataFrame
from pyspark.sql.functions import (
    window, col, sum as spark_sum, avg,
    count, count_distinct, max as spark_max, min as spark_min,
    expr, lag, from_utc_timestamp, current_timestamp,
    dense_rank, percentile_approx
)
from pyspark.sql.window import Window
from datetime import datetime, timedelta
import logging

logger = logging.getLogger(__name__)

class FeatureMaterializationJob:
    """Base class for feature materialization jobs"""
    
    def __init__(self, spark: SparkSession, feature_id: str, tenant_id: str = "default", region: str = "us-east-1"):
        self.spark = spark
        self.feature_id = feature_id
        self.tenant_id = tenant_id
        self.region = region
        self.watermark = None
    
    def read_watermark(self) -> datetime:
        """Read last processed watermark from feature_watermarks table"""
        try:
            query = f"""
            SELECT last_processed FROM feature_watermarks
            WHERE feature_id = '{self.feature_id}'
            """
            result = self.spark.sql(query).collect()
            if result:
                self.watermark = result[0][0]
                logger.info(f"Read watermark for {self.feature_id}: {self.watermark}")
            else:
                # First run: use 30 days ago
                self.watermark = datetime.utcnow() - timedelta(days=30)
                logger.info(f"No watermark found, using default: {self.watermark}")
            return self.watermark
        except Exception as e:
            logger.error(f"Failed to read watermark: {str(e)}")
            raise
    
    def update_watermark(self, new_watermark: datetime) -> None:
        """Update watermark after successful materialization"""
        try:
            query = f"""
            INSERT INTO feature_watermarks (feature_id, last_processed, last_processed_batch_id)
            VALUES ('{self.feature_id}', '{new_watermark.isoformat()}', 'batch-{datetime.utcnow().isoformat()}')
            ON CONFLICT (feature_id) DO UPDATE
            SET last_processed = EXCLUDED.last_processed,
                last_processed_batch_id = EXCLUDED.last_processed_batch_id
            """
            self.spark.sql(query)
            logger.info(f"Updated watermark for {self.feature_id} to {new_watermark}")
        except Exception as e:
            logger.error(f"Failed to update watermark: {str(e)}")
            raise
    
    def materialize(self, feature_df: DataFrame, feature_table: str) -> None:
        """
        Write feature DataFrame to Iceberg table using MERGE (idempotent).
        
        Strategy: INSERT OVERWRITE on feature_date partitions
        """
        try:
            logger.info(f"Materializing {len(feature_df.columns)} features to {feature_table}")
            
            # Write to Iceberg
            feature_df.writeTo(feature_table) \
                .tableProperty("format-version", "2") \
                .partitionedBy("feature_date", "tenant_id", "region") \
                .option("write-format", "parquet") \
                .mode("append") \
                .save()
            
            logger.info(f"Successfully materialized {feature_table}")
        except Exception as e:
            logger.error(f"Materialization failed: {str(e)}")
            raise

class MonthlyRevenueFeature(FeatureMaterializationJob):
    """Materializes monthly revenue feature (30-day rolling sum)"""
    
    def run(self):
        """Execute materialization"""
        try:
            # Read watermark
            watermark = self.read_watermark()
            
            # Run materialization query
            query = f"""
            SELECT
                tenant_id,
                region,
                DATE_TRUNC('day', event_time) as feature_date,
                SUM(amount) as revenue_30d,
                COUNT(*) as order_count,
                AVG(amount) as avg_order_value
            FROM iceberg.ops.orders
            WHERE event_time >= '{watermark.isoformat()}'
                AND event_time < CURRENT_TIMESTAMP
                AND status = 'paid'
                AND tenant_id = '{self.tenant_id}'
                AND region = '{self.region}'
            GROUP BY tenant_id, region, DATE_TRUNC('day', event_time)
            """
            
            feature_df = self.spark.sql(query) \
                .withColumn("computed_at", current_timestamp()) \
                .withColumn("feature_id", expr(f"'{self.feature_id}'"))
            
            # Materialize
            feature_table = f"iceberg.features.orders.revenue_30d_v1"
            self.materialize(feature_df, feature_table)
            
            # Update watermark
            new_watermark = datetime.utcnow()
            self.update_watermark(new_watermark)
            
            logger.info(f"Feature {self.feature_id} materialized successfully")
            return {"status": "success", "rows": feature_df.count()}
        except Exception as e:
            logger.error(f"Feature materialization failed: {str(e)}")
            raise

class P99LatencyFeature(FeatureMaterializationJob):
    """Materializes P99 latency feature from request logs"""
    
    def run(self):
        """Execute materialization"""
        try:
            watermark = self.read_watermark()
            
            # Compute P99 latency per tenant/region/hour
            query = f"""
            SELECT
                tenant_id,
                region,
                DATE_TRUNC('hour', event_time) as feature_date,
                PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY latency_ms) as p99_latency_ms,
                PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms) as p95_latency_ms,
                PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY latency_ms) as p50_latency_ms,
                COUNT(*) as request_count,
                SUM(CASE WHEN status >= 500 THEN 1 ELSE 0 END) / COUNT(*) as error_rate
            FROM iceberg.ops.request_logs
            WHERE event_time >= '{watermark.isoformat()}'
                AND event_time < CURRENT_TIMESTAMP
                AND tenant_id = '{self.tenant_id}'
                AND region = '{self.region}'
            GROUP BY tenant_id, region, DATE_TRUNC('hour', event_time)
            """
            
            feature_df = self.spark.sql(query) \
                .withColumn("computed_at", current_timestamp()) \
                .withColumn("feature_id", expr(f"'{self.feature_id}'"))
            
            # Materialize
            feature_table = f"iceberg.features.latency.p99_latency_v1"
            self.materialize(feature_df, feature_table)
            
            # Update watermark
            new_watermark = datetime.utcnow()
            self.update_watermark(new_watermark)
            
            logger.info(f"Feature {self.feature_id} materialized successfully")
            return {"status": "success", "rows": feature_df.count()}
        except Exception as e:
            logger.error(f"Feature materialization failed: {str(e)}")
            raise

class ErrorRate24hFeature(FeatureMaterializationJob):
    """Materializes 24-hour accumulated error rate feature"""
    
    def run(self):
        """Execute materialization"""
        try:
            watermark = self.read_watermark()
            
            query = f"""
            SELECT
                tenant_id,
                region,
                DATE_TRUNC('day', event_time) as feature_date,
                SUM(CASE WHEN status >= 400 THEN 1 ELSE 0 END) / COUNT(*) * 100 as error_rate_pct,
                SUM(CASE WHEN status >= 500 THEN 1 ELSE 0 END) / COUNT(*) * 100 as server_error_rate_pct,
                COUNT(*) as total_requests
            FROM iceberg.ops.request_logs
            WHERE event_time >= '{watermark.isoformat()}'
                AND event_time < CURRENT_TIMESTAMP
                AND event_time >= CURRENT_TIMESTAMP - INTERVAL 24 HOURS
                AND tenant_id = '{self.tenant_id}'
                AND region = '{self.region}'
            GROUP BY tenant_id, region, DATE_TRUNC('day', event_time)
            """
            
            feature_df = self.spark.sql(query) \
                .withColumn("computed_at", current_timestamp()) \
                .withColumn("feature_id", expr(f"'{self.feature_id}'"))
            
            # Materialize
            feature_table = f"iceberg.features.errors.error_rate_24h_v1"
            self.materialize(feature_df, feature_table)
            
            # Update watermark
            new_watermark = datetime.utcnow()
            self.update_watermark(new_watermark)
            
            logger.info(f"Feature {self.feature_id} materialized successfully")
            return {"status": "success", "rows": feature_df.count()}
        except Exception as e:
            logger.error(f"Feature materialization failed: {str(e)}")
            raise

class ActiveConflictsFeature(FeatureMaterializationJob):
    """Materializes count of active unresolved conflicts"""
    
    def run(self):
        """Execute materialization"""
        try:
            watermark = self.read_watermark()
            
            query = f"""
            SELECT
                tenant_id,
                region,
                CURRENT_DATE as feature_date,
                COUNT(*) as active_conflict_count,
                COUNT(DISTINCT conflict_type) as num_conflict_types,
                MIN(created_at) as oldest_conflict_ts
            FROM iceberg.ops.conflicts
            WHERE status = 'active'
                AND tenant_id = '{self.tenant_id}'
                AND region = '{self.region}'
            GROUP BY tenant_id, region, CURRENT_DATE
            """
            
            feature_df = self.spark.sql(query) \
                .withColumn("computed_at", current_timestamp()) \
                .withColumn("feature_id", expr(f"'{self.feature_id}'"))
            
            # Materialize
            feature_table = f"iceberg.features.conflicts.active_count_v1"
            self.materialize(feature_df, feature_table)
            
            # Update watermark
            new_watermark = datetime.utcnow()
            self.update_watermark(new_watermark)
            
            logger.info(f"Feature {self.feature_id} materialized successfully")
            return {"status": "success", "rows": feature_df.count()}
        except Exception as e:
            logger.error(f"Feature materialization failed: {str(e)}")
            raise

def run_materialization_job(feature_id: str, spark: SparkSession, tenant_id: str = "default", region: str = "us-east-1"):
    """
    Factory function to run materialization job based on feature_id.
    
    Usage from Spark submit:
        spark-submit \
          --conf spark.sql.catalog.iceberg=org.apache.iceberg.spark.SparkCatalog \
          --conf spark.sql.catalog.iceberg.type=hive \
          --conf spark.sql.extensions=org.apache.iceberg.spark.extensions.IcebergSparkSessionExtensions \
          materialization.py
    """
    
    job_map = {
        "feature:orders.monthly_revenue_v1": MonthlyRevenueFeature,
        "feature:latency.p99_ms_v1": P99LatencyFeature,
        "feature:errors.http_rate_v1": ErrorRate24hFeature,
        "feature:conflicts.active_count_v1": ActiveConflictsFeature
    }
    
    job_class = job_map.get(feature_id)
    if not job_class:
        raise ValueError(f"Unknown feature_id: {feature_id}")
    
    job = job_class(spark, feature_id, tenant_id, region)
    return job.run()

if __name__ == "__main__":
    spark = SparkSession.builder \
        .appName("FeatureMaterialization") \
        .config("spark.sql.catalog.iceberg", "org.apache.iceberg.spark.SparkCatalog") \
        .config("spark.sql.catalog.iceberg.type", "hive") \
        .config("spark.sql.extensions", "org.apache.iceberg.spark.extensions.IcebergSparkSessionExtensions") \
        .getOrCreate()
    
    import sys
    feature_id = sys.argv[1] if len(sys.argv) > 1 else "feature:orders.monthly_revenue_v1"
    tenant_id = sys.argv[2] if len(sys.argv) > 2 else "default"
    region = sys.argv[3] if len(sys.argv) > 3 else "us-east-1"
    
    result = run_materialization_job(feature_id, spark, tenant_id, region)
    print(f"Materialization complete: {result}")
