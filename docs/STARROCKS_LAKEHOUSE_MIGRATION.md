# StarRocks Lakehouse Migration Guide

## Executive Summary

This document outlines the migration from ClickHouse to **StarRocks OSS** as your unified lakehouse analytics engine, running directly on Iceberg. This eliminates the need for a separate "hot" layer while maintaining sub-second query performance for wealth management workloads.

## Architecture: Before vs After

### Current State (Lambda Architecture)
```
┌─────────────────────────────────────────────────────────────────┐
│                     Current: Lambda Architecture                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   Hot Path (ClickHouse)          Cold Path (Iceberg/Trino)     │
│   ┌─────────────────┐            ┌─────────────────────────┐   │
│   │ trades_stream   │            │ trades_history          │   │
│   │ compliance_events│     CDC   │ compliance_history      │   │
│   │ audit_log       │ ────────▶  │ audit_archive           │   │
│   │ ledger_stream   │            │                         │   │
│   └─────────────────┘            └─────────────────────────┘   │
│          │                                │                     │
│          ▼                                ▼                     │
│   Real-time dashboards            Historical queries            │
│   (sub-second)                    (seconds-minutes)             │
│                                                                 │
│   PROBLEM: Two systems to manage, data sync complexity          │
└─────────────────────────────────────────────────────────────────┘
```

### Target State (Unified Lakehouse)
```
┌─────────────────────────────────────────────────────────────────┐
│              Target: StarRocks Unified Lakehouse                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│                   ┌───────────────────────────┐                 │
│                   │     StarRocks Cluster     │                 │
│                   │  (Query Engine + Cache)   │                 │
│                   └─────────────┬─────────────┘                 │
│                                 │                               │
│                   ┌─────────────▼─────────────┐                 │
│                   │    Iceberg Catalog        │                 │
│                   │  (Nessie / Hive / REST)   │                 │
│                   └─────────────┬─────────────┘                 │
│                                 │                               │
│      ┌──────────────────────────┼──────────────────────────┐   │
│      │                          │                          │   │
│      ▼                          ▼                          ▼   │
│ ┌──────────┐            ┌──────────────┐          ┌──────────┐ │
│ │ trades   │            │ compliance   │          │ audit    │ │
│ │ (Iceberg)│            │ (Iceberg)    │          │ (Iceberg)│ │
│ └──────────┘            └──────────────┘          └──────────┘ │
│                                                                 │
│             Object Storage (MinIO / S3)                        │
│                                                                 │
│   BENEFIT: Single source of truth, unified governance          │
└─────────────────────────────────────────────────────────────────┘
```

## StarRocks OSS Capabilities for Wealth Management

### Why StarRocks Works for 200M+ Trades/Day

| Requirement | StarRocks Solution |
|-------------|-------------------|
| **High-volume ingestion** | Streaming Load API, Flink connector |
| **Sub-second queries** | Vectorized execution, CBO optimizer |
| **Direct Iceberg access** | Native Iceberg catalog integration |
| **Multi-tenancy** | Resource groups, query queues, workload isolation |
| **High concurrency** | Designed for 1000s of concurrent queries |
| **Columnar analytics** | Native columnar storage + Parquet/ORC pushdown |

### Multi-Tenancy Model

StarRocks provides workload isolation through **Resource Groups**:

```sql
-- Create resource groups per tenant tier
CREATE RESOURCE GROUP tenant_premium
WITH (
    cpu_weight = 100,
    mem_limit = '50%',
    concurrency_limit = 100,
    type = 'normal'
);

CREATE RESOURCE GROUP tenant_standard
WITH (
    cpu_weight = 50,
    mem_limit = '25%',
    concurrency_limit = 50,
    type = 'normal'
);

-- Assign users/queries to resource groups
SET PROPERTY FOR 'tenant_001' 'resource_group' = 'tenant_premium';
```

For data isolation, use:
1. **Row-level filtering** with views
2. **Schema-per-tenant** for strict isolation
3. **Partition pruning** on `tenant_id`

## Schema Migration: ClickHouse → StarRocks + Iceberg

### Trades Stream → Iceberg Table

**ClickHouse (current):**
```sql
CREATE TABLE trades_stream (
    event_time DateTime64(3),
    trade_id UUID,
    tenant_id UUID,
    portfolio_id UUID,
    ...
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(event_time)
ORDER BY (portfolio_id, event_time, trade_id);
```

**Iceberg (target):**
```sql
-- Create via Trino/Spark SQL
CREATE TABLE iceberg.wealth.trades (
    event_time TIMESTAMP(6) WITH TIME ZONE,
    trade_id VARCHAR,
    tenant_id VARCHAR,
    portfolio_id VARCHAR,
    desk_id VARCHAR,
    symbol VARCHAR,
    side VARCHAR,
    quantity DECIMAL(18, 4),
    price DECIMAL(18, 4),
    notional DECIMAL(18, 4),
    currency VARCHAR,
    basis_id VARCHAR
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY[
        'day(event_time)',
        'bucket(tenant_id, 16)'  -- Shard by tenant for parallel scans
    ],
    sorted_by = ARRAY['portfolio_id', 'event_time']
);
```

**StarRocks External Table (for direct query):**
```sql
-- StarRocks queries Iceberg directly
CREATE EXTERNAL CATALOG iceberg_catalog
PROPERTIES (
    "type" = "iceberg",
    "iceberg.catalog.type" = "rest",
    "iceberg.catalog.uri" = "http://nessie:19120/api/v1",
    "iceberg.catalog.warehouse" = "s3://lakehouse/warehouse"
);

-- Query directly
SELECT * FROM iceberg_catalog.wealth.trades
WHERE tenant_id = 'tenant_001'
  AND event_time >= CURRENT_DATE - INTERVAL 7 DAY;
```

### Materialized Views in StarRocks

Replace ClickHouse `AggregatingMergeTree` with StarRocks materialized views:

```sql
-- Daily P&L aggregation (auto-refreshed)
CREATE MATERIALIZED VIEW daily_pnl_mv
DISTRIBUTED BY HASH(portfolio_id)
REFRESH ASYNC START('2024-01-01 00:00:00') EVERY (INTERVAL 5 MINUTE)
AS
SELECT
    DATE_TRUNC('day', event_time) as trade_date,
    tenant_id,
    portfolio_id,
    desk_id,
    currency,
    COUNT(*) as total_trades,
    SUM(ABS(quantity)) as total_volume,
    SUM(notional) as total_notional
FROM iceberg_catalog.wealth.trades
GROUP BY 1, 2, 3, 4, 5;
```

## Iceberg Table Design for 200M Trades/Day

### Partitioning Strategy

```sql
-- Partition by day + tenant bucket for optimal pruning
CREATE TABLE iceberg.wealth.trades (
    ...
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY[
        'day(event_time)',           -- ~200M rows/partition/day per tenant
        'bucket(tenant_id, 32)'      -- Spread across 32 buckets
    ],
    -- Target 256MB files for optimal scan performance
    write.target-file-size-bytes = 268435456,
    -- Enable delete files for updates
    write.delete.mode = 'merge-on-read',
    write.update.mode = 'merge-on-read'
);
```

### Compaction Strategy

```sql
-- Run compaction to optimize small files
CALL iceberg.system.rewrite_data_files(
    table => 'wealth.trades',
    strategy => 'sort',
    sort_order => 'portfolio_id ASC, event_time ASC',
    options => map(
        'target-file-size-bytes', '268435456',
        'min-input-files', '5'
    )
);

-- Expire old snapshots (keep 7 days for replay)
CALL iceberg.system.expire_snapshots(
    table => 'wealth.trades',
    older_than => TIMESTAMP '2024-01-01 00:00:00',
    retain_last => 168  -- 7 days of hourly snapshots
);
```

## Go Code Migration

### Current: ClickHouse Exporter

The existing `backend/internal/analytics/exporter.go` uses ClickHouse for real-time queries.

### Target: StarRocks + Iceberg Client

```go
// backend/internal/analytics/starrocks_client.go
package analytics

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    _ "github.com/go-sql-driver/mysql" // StarRocks uses MySQL protocol
)

// StarRocksClient handles analytics queries via StarRocks
type StarRocksClient struct {
    db *sql.DB
}

// NewStarRocksClient creates a new StarRocks connection
func NewStarRocksClient(dsn string) (*StarRocksClient, error) {
    // StarRocks uses MySQL wire protocol
    // DSN format: user:password@tcp(host:port)/database
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to StarRocks: %w", err)
    }
    
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)
    
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping StarRocks: %w", err)
    }
    
    return &StarRocksClient{db: db}, nil
}

// QueryTrades queries trades from Iceberg via StarRocks
func (c *StarRocksClient) QueryTrades(ctx context.Context, tenantID string, lookbackMinutes int) ([]Trade, error) {
    query := `
        SELECT 
            event_time, trade_id, portfolio_id, desk_id,
            symbol, side, quantity, price, notional, currency
        FROM iceberg_catalog.wealth.trades
        WHERE tenant_id = ?
          AND event_time >= DATE_SUB(NOW(), INTERVAL ? MINUTE)
        ORDER BY event_time DESC
        LIMIT 10000
    `
    
    rows, err := c.db.QueryContext(ctx, query, tenantID, lookbackMinutes)
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()
    
    var trades []Trade
    for rows.Next() {
        var t Trade
        if err := rows.Scan(
            &t.EventTime, &t.TradeID, &t.PortfolioID, &t.DeskID,
            &t.Symbol, &t.Side, &t.Quantity, &t.Price, &t.Notional, &t.Currency,
        ); err != nil {
            return nil, fmt.Errorf("scan failed: %w", err)
        }
        trades = append(trades, t)
    }
    
    return trades, nil
}

// QueryDailyPnL uses the materialized view for fast aggregates
func (c *StarRocksClient) QueryDailyPnL(ctx context.Context, tenantID, portfolioID string, days int) ([]DailyPnL, error) {
    query := `
        SELECT 
            trade_date, portfolio_id, desk_id, currency,
            total_trades, total_volume, total_notional
        FROM daily_pnl_mv
        WHERE tenant_id = ?
          AND portfolio_id = ?
          AND trade_date >= DATE_SUB(CURRENT_DATE(), INTERVAL ? DAY)
        ORDER BY trade_date DESC
    `
    
    rows, err := c.db.QueryContext(ctx, query, tenantID, portfolioID, days)
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()
    
    var results []DailyPnL
    for rows.Next() {
        var p DailyPnL
        if err := rows.Scan(
            &p.TradeDate, &p.PortfolioID, &p.DeskID, &p.Currency,
            &p.TotalTrades, &p.TotalVolume, &p.TotalNotional,
        ); err != nil {
            return nil, fmt.Errorf("scan failed: %w", err)
        }
        results = append(results, p)
    }
    
    return results, nil
}

// Close closes the database connection
func (c *StarRocksClient) Close() error {
    return c.db.Close()
}
```

## Docker Compose: StarRocks Cluster

Replace ClickHouse service with StarRocks:

```yaml
# docker-compose.yml (StarRocks section)
services:
  starrocks-fe:
    image: starrocks/fe-ubuntu:latest
    hostname: starrocks-fe
    ports:
      - "8030:8030"   # HTTP port
      - "9020:9020"   # RPC port
      - "9030:9030"   # MySQL port (query interface)
    volumes:
      - starrocks_fe_data:/opt/starrocks/fe/meta
    environment:
      - AWS_ACCESS_KEY_ID=minioadmin
      - AWS_SECRET_ACCESS_KEY=minioadmin
      - AWS_REGION=us-east-1
    networks:
      - semlayer-net
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8030/api/health"]
      interval: 30s
      timeout: 10s
      retries: 5

  starrocks-be:
    image: starrocks/be-ubuntu:latest
    hostname: starrocks-be
    ports:
      - "8040:8040"   # HTTP port
      - "9060:9060"   # RPC port
    volumes:
      - starrocks_be_data:/opt/starrocks/be/storage
    environment:
      - AWS_ACCESS_KEY_ID=minioadmin
      - AWS_SECRET_ACCESS_KEY=minioadmin
      - AWS_REGION=us-east-1
    depends_on:
      starrocks-fe:
        condition: service_healthy
    networks:
      - semlayer-net

  # Nessie catalog for Iceberg
  nessie:
    image: projectnessie/nessie:latest
    ports:
      - "19120:19120"
    environment:
      - NESSIE_VERSION_STORE_TYPE=ROCKSDB
    volumes:
      - nessie_data:/data
    networks:
      - semlayer-net

volumes:
  starrocks_fe_data:
  starrocks_be_data:
  nessie_data:
```

## Migration Checklist

### Phase 1: Infrastructure Setup
- [ ] Deploy StarRocks FE/BE cluster (docker-compose or K8s)
- [ ] Deploy Nessie catalog for Iceberg metadata
- [ ] Configure MinIO/S3 for Iceberg data files
- [ ] Create Iceberg external catalog in StarRocks

### Phase 2: Schema Migration
- [ ] Create Iceberg tables with proper partitioning
- [ ] Create StarRocks materialized views for hot aggregates
- [ ] Configure resource groups for multi-tenancy

### Phase 3: Data Migration
- [ ] Backfill historical data from ClickHouse → Iceberg
- [ ] Validate row counts and checksums
- [ ] Set up streaming ingestion (Flink/Kafka → Iceberg)

### Phase 4: Code Migration
- [ ] Replace ClickHouse Go client with StarRocks MySQL client
- [ ] Update query patterns for StarRocks SQL dialect
- [ ] Update docker-compose.yml
- [ ] Update CI/CD workflows

### Phase 5: Validation
- [ ] Run benchmark queries (P50/P95/P99)
- [ ] Validate multi-tenant isolation
- [ ] Test Temporal workflow replay with Iceberg snapshots
- [ ] Load test with simulated 200M trades/day

## Performance Tuning Tips

### StarRocks Query Optimization

```sql
-- Enable query profile for analysis
SET enable_profile = true;

-- Force partition pruning analysis
EXPLAIN SELECT * FROM iceberg_catalog.wealth.trades
WHERE tenant_id = 'tenant_001' AND event_time >= '2024-01-01';

-- Use hints for complex joins
SELECT /*+ SET_VAR(parallel_fragment_exec_instance_num=8) */
    t.*, p.portfolio_name
FROM iceberg_catalog.wealth.trades t
JOIN portfolios p ON t.portfolio_id = p.id
WHERE t.tenant_id = 'tenant_001';
```

### Iceberg Optimization

```bash
# Run compaction regularly (cron job or Airflow)
spark-sql --conf spark.sql.catalog.iceberg=org.apache.iceberg.spark.SparkCatalog \
  -e "CALL iceberg.system.rewrite_data_files('wealth.trades')"

# Monitor file sizes
spark-sql -e "SELECT file_path, file_size_in_bytes 
              FROM wealth.trades.files"
```

## When You Might Still Want ClickHouse

Keep ClickHouse only if you need:
1. **< 10ms P99 latency** for specific operational queries
2. **100k+ QPS** on simple aggregations
3. **Real-time log analytics** separate from lakehouse

For most wealth management dashboards and compliance queries, StarRocks + Iceberg will meet SLAs while simplifying your architecture.

## References

- [StarRocks Iceberg Catalog](https://docs.starrocks.io/docs/data_source/catalog/iceberg_catalog/)
- [StarRocks Resource Groups](https://docs.starrocks.io/docs/administration/resource_group/)
- [Apache Iceberg Table Spec](https://iceberg.apache.org/spec/)
- [Iceberg Partitioning Best Practices](https://iceberg.apache.org/docs/latest/partitioning/)
