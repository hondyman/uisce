# Monitoring Specification: Hybrid Trino-Iceberg Analytics

## Overview
This document defines the key performance indicators (KPIs), Service Level Objectives (SLOs), and Alerting rules for the production analytics stack.

## Architecture Components
- **Ingest**: Debezium -> Redpanda -> RisingWave
- **Storage**: Iceberg (MinIO/S3)
- **Compute**: Trino (Query), Cube.dev (Semantic API)

## SLOs & Targets
| Component | Metric | Target | P95 |
| -- | -- | -- | -- |
| **CDC Pipeline** | End-to-End Lag | < 60s | 30s |
| **RisingWave** | MV Refresh Latency | < 5s | 2s |
| **Trino** | Interactive Query Latency | < 3s | 1.5s |
| **Trino** | Ad-hoc Query Latency | < 30s | 10s |
| **Cube.dev** | API Response (Cached) | < 500ms | 200ms |

## Grafana Dashboard Panels

### Row 1: Health & Latency
- **End-to-End Latency**: `sum(kafka_lag_seconds) + risingwave_barrier_latency`
- **Trino Active Queries**: `trino_queries{state="RUNNING"}`
- **Cube Cache Hit Ratio**: `cube_cache_hits / (cube_cache_hits + cube_cache_misses)`

### Row 2: Resources
- **Trino Worker Memory**: `trino_worker_heap_used_percent` (Alert > 85%)
- **Redpanda Partition Lag**: `max(kafka_consumer_group_lag)`
- **RisingWave Barrier Lag**: `risingwave_stream_barrier_manager_current_lag`

### Row 3: Governance & Anomalies
- **Semantic Anomalies**: `rate(semantic_events{type="ANOMALY"}[5m])`
- **Pre-Agg Freshness**: `time() - to_timestamp(iceberg_partition_last_update)`

## Alerting Rules (Prometheus)

```yaml
groups:
- name: critical_alerts
  rules:
  - alert: TrinoMemoryHigh
    expr: trino_worker_heap_used_percent > 85
    for: 5m
    labels:
        severity: critical
    annotations:
        summary: "Trino worker memory critically high"

  - alert: HighCDCLag
    expr: kafka_consumer_group_lag > 10000
    for: 10m
    labels:
        severity: warning
    annotations:
        summary: "CDC ingestion is lagging behind > 10k messages"
```
